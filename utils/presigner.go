package utils

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"

	utils "github.com/bolt-observer/go_common/utils"
)

// We use AWS pre-signed URLs here which can be used as an effective way to use IAM authentication for a custom app,
// read https://ahermosilla.com/cloud/2020/11/17/leveraging-aws-signed-requests.html for more details.

const (
	// PresignHeader - HTTP Header for pre-signed requests
	PresignHeader = "X-Amazon-Presigned-Getcalleridentity"
	// EmptyBodyHash - Hash of empty body
	EmptyBodyHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
)

// GetCallerIdentityResponse struct
type GetCallerIdentityResponse struct {
	GetCallerIdentityResult GetCallerIdentityResult
}

// GetCallerIdentityResult struct
type GetCallerIdentityResult struct {
	Arn     string `xml:"Arn"`
	UserID  string `xml:"UserId"`
	Account string `xml:"Account"`
}

var (
	region   string
	service  string
	endpoint string
)

func init() {
	region = utils.GetEnvWithDefault("AWS_DEFAULT_REGION", "us-east-1")
	service = "sts"
	endpoint = fmt.Sprintf("https://%s.%s.amazonaws.com/", service, region)
}

// VerifyGetCallerIdentity will verify that query string received is actually a presigned URL
// to sts/GetCallerIdentity.
// Returns:
//  - ARN of the identity when successful
//  - error else
func VerifyGetCallerIdentity(query string, timeout time.Duration) (string, error) {
	var identity GetCallerIdentityResponse

	if strings.ContainsAny(query, "@?/") || strings.HasPrefix(query, "http") {
		return "", fmt.Errorf("invalid query string")
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("unable to create request, %v", err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	req.URL.RawQuery = query

	for key := range req.URL.Query() {
		lowKey := strings.ToLower(key)
		if lowKey == "action" || lowKey == "redirect" || lowKey == "version" {
			return "", fmt.Errorf("action trickery detected")
		}
	}

	req.URL.RawQuery = "Action=GetCallerIdentity&Version=2011-06-15&" + query

	if int64(timeout) <= 0 {
		timeout = 5 * time.Second
	}

	client := &http.Client{
		Timeout: timeout,
	}

	if !strings.HasSuffix(req.URL.Hostname(), ".amazonaws.com") {
		return "", fmt.Errorf("hostname trickery detected, %s", req.URL.Hostname())
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("unable to make request, %v", err)
	}

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return "", fmt.Errorf("got unauthorized, %v", resp.StatusCode)
	}

	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("unable to read response body: %v", err)
	}

	err = xml.Unmarshal(b, &identity)
	if err != nil {
		return "", fmt.Errorf("unable to deserialize response, %v", err)
	}

	if identity.GetCallerIdentityResult.Arn == "" {
		return "", fmt.Errorf("empty result")
	}

	return identity.GetCallerIdentityResult.Arn, nil
}

// PresignGetCallerIdentity will sign a query string
// to retrieve my caller identity by third party.
// Returns:
// - the query string
// - error (when not successful)
func PresignGetCallerIdentity(validity time.Duration) (string, error) {
	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return "", fmt.Errorf("unable to load SDK config, %v", err)
	}

	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return "", fmt.Errorf("unable to retrieve credentials, %v", err)
	}

	if creds.Expired() {
		return "", fmt.Errorf("credentials expired")
	}

	req, err := http.NewRequest("POST", endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("unable to create request, %v", err)
	}

	query := req.URL.Query()

	query.Set("Action", "GetCallerIdentity")
	query.Set("Version", "2011-06-15")

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	if int64(validity) <= 0 {
		validity = 5 * time.Minute
	}

	query.Set("X-Amz-Expires", strconv.FormatInt(int64(validity/time.Second), 10))
	req.URL.RawQuery = query.Encode()

	signer := v4.NewSigner()

	u, _, err := signer.PresignHTTP(ctx, creds, req, EmptyBodyHash, service, region, time.Now())

	if err != nil {
		return "", fmt.Errorf("unable to sign request, %v", err)
	}

	gu, err := url.Parse(u)
	if err != nil {
		return "", fmt.Errorf("unable to parse, %v", err)
	}

	nq := gu.Query()
	nq.Del("Action")
	nq.Del("Version")

	gu.RawQuery = nq.Encode()

	return gu.RawQuery, nil
}
