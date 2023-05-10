package utils

import (
	"context"
	"fmt"
	"hash/crc32"
	"strings"
	"time"

	"path/filepath"

	sapi "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	backoff "github.com/cenkalti/backoff/v4"
	"github.com/getsentry/sentry-go"
	"github.com/golang/glog"
	"google.golang.org/api/iterator"
)

// GcpSecretsManager struct.
type GcpSecretsManager struct {
}

// NewGcpSecretsManager creates a new GcpSecretsManager
func NewGcpSecretsManager() *GcpSecretsManager {
	return &GcpSecretsManager{}
}

// LoadSecrets - loads all secrets (used at startup)
func (s *GcpSecretsManager) LoadSecrets(ctx context.Context, prefix string) map[string]string {
	ret := make(map[string]string)
	for _, name := range listSecretsGcp(ctx, prefix) {
		key, value, err := getSecretGcp(ctx, name)
		if err != nil {
			glog.Errorf("Could not get secret: %v", err)
			sentry.CaptureException(err)
			continue
		}
		ret[key] = value
	}

	return ret
}

// InsertOrUpdateSecret - inserts or updates a secret
func (s *GcpSecretsManager) InsertOrUpdateSecret(ctx context.Context, name, value string) (string, Change, error) {
	back := backoff.NewExponentialBackOff()
	back.MaxElapsedTime = MaxRetryTime

	x, err := backoff.RetryNotifyWithData(func() (InsertOrUpdateSecretData, error) {
		arn, change, err := insertOrUpdateSecretGcp(ctx, name, value)
		return InsertOrUpdateSecretData{
			Arn:    arn,
			Change: change,
		}, err
	}, back, func(err error, d time.Duration) {
		glog.Warningf("Error inserting or updating secret")
	})

	return x.Arn, x.Change, err
}

// DeleteSecret - deletes a secret
func (s *GcpSecretsManager) DeleteSecret(ctx context.Context, name string) (string, error) {
	back := backoff.NewExponentialBackOff()
	back.MaxElapsedTime = MaxRetryTime

	resp, err := backoff.RetryNotifyWithData(func() (string, error) {
		return deleteSecretGcp(ctx, name)
	}, back, func(err error, d time.Duration) {
		glog.Warningf("Error invalidating secret")
	})

	return resp, err
}

func insertOrUpdateSecretGcp(ctx context.Context, name, value string) (string, Change, error) {
	ch := Inserted
	project, err := GetGCPProjectID()
	if err != nil {
		sentry.CaptureException(err)
		glog.Errorf("unable to load project: %v", err)
		return "", ch, err
	}
	client, err := sapi.NewClient(ctx)
	if err != nil {
		sentry.CaptureException(err)
		glog.Errorf("unable to load sdk: %v", err)
		return "", ch, err
	}

	defer client.Close()

	secretName := fmt.Sprintf("projects/%s/secrets/%s", project, name)
	payload := []byte(value)
	crc32c := crc32.MakeTable(crc32.Castagnoli)
	checksum := int64(crc32.Checksum(payload, crc32c))

	get := &secretmanagerpb.GetSecretRequest{
		Name: secretName,
	}

	var secret *secretmanagerpb.Secret

	secret, err = client.GetSecret(ctx, get)
	if err == nil {
		ch = Updated
	} else {
		ch = Inserted

		createSecretReq := &secretmanagerpb.CreateSecretRequest{
			Parent:   fmt.Sprintf("projects/%s", project),
			SecretId: name,
			Secret: &secretmanagerpb.Secret{
				Replication: &secretmanagerpb.Replication{
					Replication: &secretmanagerpb.Replication_Automatic_{
						Automatic: &secretmanagerpb.Replication_Automatic{},
					},
				},
			},
		}

		secret, err = client.CreateSecret(ctx, createSecretReq)
		if err != nil {
			return "", ch, err
		}
	}

	addSecretVersionReq := &secretmanagerpb.AddSecretVersionRequest{
		Parent: secret.Name,
		Payload: &secretmanagerpb.SecretPayload{
			Data:       payload,
			DataCrc32C: &checksum,
		},
	}

	_, err = client.AddSecretVersion(ctx, addSecretVersionReq)
	if err != nil {
		return "", ch, err
	}

	return secret.Name, ch, nil
}

func deleteSecretGcp(ctx context.Context, name string) (string, error) {
	project, err := GetGCPProjectID()
	if err != nil {
		sentry.CaptureException(err)
		glog.Errorf("unable to load project: %v", err)
		return "", err
	}
	client, err := sapi.NewClient(ctx)
	if err != nil {
		sentry.CaptureException(err)
		glog.Errorf("unable to load sdk: %v", err)
		return "", err
	}

	defer client.Close()

	req := &secretmanagerpb.DeleteSecretRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s", project, name),
	}
	err = client.DeleteSecret(ctx, req)
	if err != nil {
		return "", err
	}
	return name, nil
}

func listSecretsGcp(ctx context.Context, prefix string) []string {
	ret := make([]string, 0)
	project, err := GetGCPProjectID()
	if err != nil {
		sentry.CaptureException(err)
		glog.Errorf("unable to load project: %v", err)
		return ret
	}
	client, err := sapi.NewClient(ctx)
	if err != nil {
		sentry.CaptureException(err)
		glog.Errorf("unable to load sdk: %v", err)
		return ret
	}

	defer client.Close()

	var result *sapi.SecretIterator = nil

	for {
		token := ""
		if result != nil {
			token = result.PageInfo().Token
		}
		input := &secretmanagerpb.ListSecretsRequest{
			Parent:    fmt.Sprintf("projects/%s", project),
			Filter:    fmt.Sprintf("name:%s", prefix),
			PageSize:  100,
			PageToken: token,
		}

		result = client.ListSecrets(ctx, input)
		for {
			data, err := result.Next()
			if data == nil || err == iterator.Done {
				break
			}

			name := getLastSegment(data.Name)
			if strings.HasPrefix(name, prefix) {
				ret = append(ret, name)
			}
		}

		if result == nil || result.PageInfo().Token == "" {
			break
		}
	}

	return ret
}

func getSecretGcp(ctx context.Context, name string) (string, string, error) {
	project, err := GetGCPProjectID()
	if err != nil {
		sentry.CaptureException(err)
		glog.Errorf("unable to load project: %v", err)
		return "", "", err
	}
	client, err := sapi.NewClient(ctx)
	if err != nil {
		sentry.CaptureException(err)
		glog.Errorf("unable to load sdk: %v", err)
		return "", "", err
	}

	defer client.Close()

	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", project, name),
	}

	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		sentry.CaptureException(err)
		glog.Errorf("unable to load sdk: %v", err)
		return "", "", err
	}

	return name, string(result.Payload.Data), nil
}

func getLastSegment(path string) string {
	return filepath.Base(path)
}
