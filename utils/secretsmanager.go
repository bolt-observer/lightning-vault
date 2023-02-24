package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/getsentry/sentry-go"

	"github.com/golang/glog"

	backoff "github.com/cenkalti/backoff/v4"
)

// MaxRetryTime is the maximum time we will retry AWS cals
const MaxRetryTime = 30 * time.Second

// Change enum
type Change int

// Change enum values
const (
	Undefined Change = iota
	Inserted
	Updated
)

// InsertOrUpdateSecretSignature is the signature of a function
type InsertOrUpdateSecretSignature func(ctx context.Context, name, value string) (string, Change, error)

// DeleteSecretSignature is the signature of a function
type DeleteSecretSignature func(ctx context.Context, name string) (string, error)

// DeleteSecret - deletes a secret - Deprecated since you cannot reuse same secret name in 7 days
func DeleteSecret(ctx context.Context, name string) (string, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		sentry.CaptureException(err)
		return "", err
	}

	svc := secretsmanager.NewFromConfig(cfg)
	resp, err := svc.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{SecretId: &name})

	if err != nil {
		sentry.CaptureException(err)
		return "", err
	}

	return *resp.ARN, nil
}

// InvalidateSecret - is used as a replacement for DeleteSecret
func InvalidateSecret(ctx context.Context, name string) (string, error) {
	arn := ""
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		sentry.CaptureException(err)
		return "", err
	}

	svc := secretsmanager.NewFromConfig(cfg)

	listInput := &secretsmanager.ListSecretsInput{
		MaxResults: aws.Int32(1),
		Filters:    []types.Filter{{Key: types.FilterNameStringTypeName, Values: []string{name}}},
	}

	list, err := svc.ListSecrets(ctx, listInput)
	if err != nil {
		glog.Errorf("Could not list secrets: %v", err)
		sentry.CaptureException(err)
		return "", err
	}

	if len(list.SecretList) == 1 {
		arn = *list.SecretList[0].ARN
	}

	if arn != "" {
		value := "{}"

		updateInput := &secretsmanager.UpdateSecretInput{
			SecretId:     &arn,
			SecretString: &value,
		}

		_, err := svc.UpdateSecret(ctx, updateInput)

		if err != nil {
			glog.Errorf("Could not update secret: %v", err)
			sentry.CaptureException(err)
			return "", err
		}

		return arn, nil
	}

	glog.Errorf("Could not invalidate secret that does not exist: %s", name)
	sentry.CaptureMessage(fmt.Sprintf("Could not invalidate secret that does not exist: %s", name))
	return "", fmt.Errorf("cannot invalidate secret that does not exist: %s", name)
}

// InsertOrUpdateSecretData struct
type InsertOrUpdateSecretData struct {
	Arn    string
	Change Change
}

// InsertOrUpdateSecretWithRetries - calls InsertOrUpdateSecret with retry logic
func InsertOrUpdateSecretWithRetries(ctx context.Context, name, value string) (string, Change, error) {
	back := backoff.NewExponentialBackOff()
	back.MaxElapsedTime = MaxRetryTime

	s, err := backoff.RetryNotifyWithData(func() (InsertOrUpdateSecretData, error) {
		arn, change, err := InsertOrUpdateSecret(ctx, name, value)
		return InsertOrUpdateSecretData{
			Arn:    arn,
			Change: change,
		}, err
	}, back, func(err error, d time.Duration) {
		glog.Warningf("Error inserting or updating secret")
	})

	return s.Arn, s.Change, err
}

// InvalidateSecretWithRetries calls InvalidateSecret with retry logic
func InvalidateSecretWithRetries(ctx context.Context, name string) (string, error) {
	back := backoff.NewExponentialBackOff()
	back.MaxElapsedTime = MaxRetryTime

	resp, err := backoff.RetryNotifyWithData(func() (string, error) {
		return InvalidateSecret(ctx, name)
	}, back, func(err error, d time.Duration) {
		glog.Warningf("Error invalidating secret")
	})

	return resp, err
}

// InsertOrUpdateSecret - inserts or updates a secret
func InsertOrUpdateSecret(ctx context.Context, name, value string) (string, Change, error) {
	arn := ""
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		sentry.CaptureException(err)
		return "", Undefined, err
	}

	svc := secretsmanager.NewFromConfig(cfg)

	listInput := &secretsmanager.ListSecretsInput{
		MaxResults: aws.Int32(1),
		Filters:    []types.Filter{{Key: types.FilterNameStringTypeName, Values: []string{name}}},
	}

	list, err := svc.ListSecrets(ctx, listInput)
	if err != nil {
		glog.Errorf("Could not list secrets: %v", err)
		sentry.CaptureException(err)
		return "", Undefined, err
	}

	if len(list.SecretList) == 1 {
		arn = *list.SecretList[0].ARN
	}

	if arn == "" {
		createInput := &secretsmanager.CreateSecretInput{
			Name:         &name,
			SecretString: &value,
		}

		resp, err := svc.CreateSecret(ctx, createInput)

		if err != nil {
			glog.Errorf("Could not create secret: %v", err)
			sentry.CaptureException(err)
			return "", Undefined, err
		}

		return *resp.ARN, Inserted, nil
	}
	/* Due to tombstones */
	change := Updated

	getInput := &secretsmanager.GetSecretValueInput{
		SecretId: &arn,
	}

	result, err := svc.GetSecretValue(ctx, getInput)
	if err == nil {
		if *result.SecretString == "{}" {
			change = Inserted
		}
	}
	/* Due to tombstones */

	updateInput := &secretsmanager.UpdateSecretInput{
		SecretId:     &arn,
		SecretString: &value,
	}

	resp, err := svc.UpdateSecret(ctx, updateInput)

	if err != nil {
		glog.Errorf("Could not update secret: %v", err)
		sentry.CaptureException(err)
		return "", Undefined, err
	}

	if arn != *resp.ARN {
		glog.Errorf("Secret ARN changed: %v vs %v", arn, resp.ARN)
		sentry.CaptureMessage(fmt.Sprintf("Secret ARN changed: %v vs %v", arn, resp.ARN))
		return "", Undefined, err
	}

	return *resp.ARN, change, nil
}

// LoadSecrets - loads all secrets (used at startup)
func LoadSecrets(ctx context.Context, prefix string) map[string]string {
	ret := make(map[string]string)
	for _, arn := range listSecrets(ctx, prefix) {
		key, value, err := GetSecret(ctx, arn)
		if err != nil {
			glog.Errorf("Could not get secret: %v", err)
			sentry.CaptureException(err)
			continue
		}
		ret[key] = value
	}

	return ret
}

// GetSecret - gets secret by arn
func GetSecret(ctx context.Context, arn string) (string, string, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		sentry.CaptureException(err)
		return "", "", err
	}

	svc := secretsmanager.NewFromConfig(cfg)

	input := &secretsmanager.GetSecretValueInput{
		SecretId: &arn,
	}

	result, err := svc.GetSecretValue(ctx, input)
	if err != nil {
		return "", "", err
	}

	return *result.Name, *result.SecretString, nil
}

func listSecrets(ctx context.Context, prefix string) []string {
	ret := make([]string, 0)
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		sentry.CaptureException(err)
		glog.Errorf("unable to load sdk: %v", err)
		return ret
	}

	svc := secretsmanager.NewFromConfig(cfg)

	result := &secretsmanager.ListSecretsOutput{
		NextToken: nil,
	}

	for {
		input := &secretsmanager.ListSecretsInput{
			MaxResults: aws.Int32(100),
			NextToken:  result.NextToken,
			Filters:    []types.Filter{{Key: types.FilterNameStringTypeName, Values: []string{prefix}}},
		}

		result, err = svc.ListSecrets(ctx, input)
		if err != nil {
			sentry.CaptureException(err)
			glog.Errorf("Could not list secrets: %v", err)
			return ret
		}

		for _, v := range result.SecretList {
			ret = append(ret, *v.ARN)
		}

		if result.NextToken == nil {
			break
		}
	}

	return ret
}
