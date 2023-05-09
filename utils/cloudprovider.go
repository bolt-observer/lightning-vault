package utils

import (
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
		{URL: "http://169.254.169.254/latest/dynamic/instance-identity/document", Provider: AWS},
		{URL: "http://metadata.google.internal/computeMetadata/v1", Provider: GCP},
	}

	for _, pair := range pairs {
		req, err := http.NewRequest("GET", pair.URL, nil)
		if err != nil {
			glog.Warningf("Error creating request: %v", err)
			continue
		}

		req.Header.Set("User-Agent", "lightning-vault")

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
