package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

//Version configuration can be sent back easily on the response from Daemon
var (
	//latest Tag version
	versionTag string
	//sha1 revision used to build
	sha1Revision   string
	//Time when the binary was built
	buildTime   string

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


//This function is called to see if the current daemon is on the latest version , if it is not, indicate this to the user
//when the daemon starts.
func CheckVersionOfDaemon() (message string,err error) {
	latestVersionFromGit,err := GetLatestDaemonVersion()
	if len(versionTag) > 0 && err == nil  {
		if strings.Compare(latestVersionFromGit,versionTag) != 0 {
			message = fmt.Sprintf("There is a newer version of the Daemon %v available. You are currently on %v, please consider upgrading.",latestVersionFromGit,versionTag)
		}
	}
	return message,err
}


func GetLatestDaemonVersion() (version string,err error) {
	resp, err := http.Get("https://api.github.com/repos/singnet/snet-daemon/releases/latest")
	if err != nil {
		return "",err
	}

	if resp.StatusCode == http.StatusOK {
		if body, err := ioutil.ReadAll(resp.Body) ; err == nil {
			var data GitTags
			if err = json.Unmarshal(body, &data); err == nil {
				version = data.Tag_name
			}
		}
	}
	defer resp.Body.Close()

	return "",err
}

type GitTags struct {
	Tag_name string `json:"tag_name"`
}