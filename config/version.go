package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const latestReleaseURL = "https://api.github.com/repos/singnet/snet-daemon/releases/latest"

var versionHTTPClient = &http.Client{Timeout: 10 * time.Second}

// Version configuration can be sent back easily on the response from Daemon
var (
	//latest Tag version
	versionTag string
	//sha1 revision used to build
	sha1Revision string
	//Time when the binary was built
	buildTime string
)

func GetVersionTag() string {
	return versionTag
}

func GetSha1Revision() string {
	return sha1Revision
}

func GetBuildTime() string {
	return buildTime
}

// This function is called to see if the current daemon is on the latest version , if it is not, indicate this to the user
// when the daemon starts.
func CheckVersionOfDaemon() (message string, err error) {
	var latestVersionFromGit string
	message = "Daemon version is " + versionTag
	latestVersionFromGit, err = GetLatestDaemonVersion()
	if len(versionTag) > 0 && err == nil {
		if strings.Compare(latestVersionFromGit, versionTag) != 0 {
			err = fmt.Errorf("there is a newer version of the Daemon %v available. You are currently on %v, please consider upgrading", latestVersionFromGit, versionTag)
		}
	}
	return message, err
}

func GetLatestDaemonVersion() (version string, err error) {
	return getLatestDaemonVersion(versionHTTPClient, latestReleaseURL)
}

func getLatestDaemonVersion(client *http.Client, endpoint string) (string, error) {
	resp, err := client.Get(endpoint)
	if err != nil {
		return "", fmt.Errorf("error getting latest daemon version from github: %+v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github releases API returned status %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading latest daemon version: %w", err)
	}

	var data GitTags
	if err := json.Unmarshal(body, &data); err != nil {
		return "", fmt.Errorf("error decoding latest daemon version: %w", err)
	}
	if data.TagName == "" {
		return "", fmt.Errorf("github releases API response has no tag_name")
	}
	return data.TagName, nil
}

type GitTags struct {
	TagName string `json:"tag_name"`
}
