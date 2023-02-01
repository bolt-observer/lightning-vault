package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	entities "github.com/bolt-observer/go_common/entities"
	utils "github.com/bolt-observer/go_common/utils"
	"github.com/getsentry/sentry-go"
)

// GetData - obtain data from vault
func GetData(name string, uniqueID string) (*entities.Data, error) {
	var (
		data entities.Data
	)

	url := utils.GetEnv("MACAROON_STORAGE_URL")
	url = strings.TrimRight(url, "/")
	token := utils.GetEnvWithDefault("READ_TOKEN", "")

	timeout := utils.GetEnvWithDefault("TIMEOUT", "10")
	timeoutInt, err := strconv.Atoi(timeout)
	if err != nil {
		sentry.CaptureException(err)
		return nil, fmt.Errorf("invalid timeout %s", timeout)
	}

	client := &http.Client{
		Timeout: time.Second * time.Duration(timeoutInt),
	}

	prettyURL := fmt.Sprintf("%s/get/%s", url, name)
	if uniqueID != "" {
		prettyURL = fmt.Sprintf("%s/get/%s/%s", url, uniqueID, name)
	}

	req, err := http.NewRequest(http.MethodGet, prettyURL, nil)
	if err != nil {
		return nil, fmt.Errorf("http request failed %v", err)
	}

	if token != "" {
		auth := strings.Split(token, UserPassSeparator)
		if len(auth) != 2 {
			return nil, fmt.Errorf("invalid token")
		}
		req.SetBasicAuth(auth[0], auth[1])
	} else {
		presign, err := PresignGetCallerIdentity(5 * time.Minute)
		if err != nil {
			sentry.CaptureException(err)
			return nil, fmt.Errorf("cannot use presign %v", err)
		}

		req.Header.Add(PresignHeader, presign)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request got error %v", err)
	}
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return nil, fmt.Errorf("authentication failed: %v", resp.StatusCode)
	}
	if resp.StatusCode == 400 || resp.StatusCode == 404 {
		return nil, fmt.Errorf("not found: %v", resp.StatusCode)
	}

	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)
	s := string(b)

	decoder := json.NewDecoder(strings.NewReader(s))

	err = decoder.Decode(&data)
	if err != nil {
		return nil, fmt.Errorf("could not decode %v (%s)", err, s)
	}

	return &data, nil
}
