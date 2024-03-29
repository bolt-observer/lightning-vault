package utils

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"
)

// CloudProvider enum.
type CloudProvider int

// CloudProvider enum values.
const (
	UnknownProvider CloudProvider = iota
	AWS
	GCP
)

// URLCloudPair struct.
type URLCloudPair struct {
	URL      string
	Provider CloudProvider
	Header   http.Header
}

// DetermineProvider tries to determine the cloud provider or uses CLOUD_PROVIDER environment variable
func DetermineProvider() CloudProvider {
	// Override the selection
	switch strings.ToLower(os.Getenv("CLOUD_PROVIDER")) {
	case "aws":
		return AWS
	case "gcp":
		return GCP
	}

	client := http.Client{
		Timeout: time.Second * 3,
	}

	pairs := []URLCloudPair{
		{URL: "http://169.254.169.254/latest/dynamic/instance-identity/document", Provider: AWS, Header: make(http.Header)},
		{URL: "http://metadata.google.internal/computeMetadata/v1", Provider: GCP, Header: http.Header{
			"Metadata-Flavor": {"Google"},
		}},
	}

	for _, pair := range pairs {
		req, err := http.NewRequest("GET", pair.URL, nil)
		if err != nil {
			glog.Warningf("Error creating request: %v", err)
			continue
		}

		req.Header.Set("User-Agent", "lightning-vault")
		for k, v := range pair.Header {
			req.Header.Set(k, v[0])
		}

		resp, err := client.Do(req)
		if err != nil {
			continue
		}

		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusPermanentRedirect || resp.StatusCode == http.StatusTemporaryRedirect {
			return pair.Provider
		}
	}

	return UnknownProvider
}

// GetGCPProjectID - get GCP project id from workstation
func GetGCPProjectID() (string, error) {
	id := os.Getenv("GCP_PROJECT_ID")
	if id != "" {
		return id, nil
	}

	client := http.Client{
		Timeout: time.Second * 3,
	}

	req, err := http.NewRequest("GET", "http://metadata.google.internal/computeMetadata/v1/project/project-id", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "lightning-vault")
	req.Header.Set("Metadata-Flavor", "Google")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
