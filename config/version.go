package config

import (
	"fmt"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/storage/memory"
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



func CheckVersionOfDaemon() (message string,err error) {
	//if the version tag has been set
	message  = ""
	if len(versionTag) > 0 {
		//compare this with the latest version tag from gits
		{
			if repo, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
				URL: "https://github.com/singnet/snet-daemon",
			}); err == nil {
				if latest,err := GetLatestTagFromRepository(repo); err == nil {
					latest = strings.Replace(latest, "refs/tags/", "", -1)
					if strings.Compare(latest,versionTag) != 0 {
						message = fmt.Sprintf("PLEASE NOTE, the latest version if Daemon is %s, You are on %s",latest,versionTag)
					}
				}
			}
		}
	}
	return message,err
}

func GetLatestTagFromRepository(repository *git.Repository) (string, error) {
	tagRefs, err := repository.Tags()
	if err != nil {
		return "", err
	}
	var latestTagCommit *object.Commit
	var latestTagName string
	err = tagRefs.ForEach(func(tagRef *plumbing.Reference) error {
		revision := plumbing.Revision(tagRef.Name().String())
		tagCommitHash, err := repository.ResolveRevision(revision)
		if err != nil {
			return err
		}
		commit, err := repository.CommitObject(*tagCommitHash)
		if err != nil {
			return err
		}

		if latestTagCommit == nil {
			latestTagCommit = commit
			latestTagName = tagRef.Name().String()
		}

		if commit.Committer.When.After(latestTagCommit.Committer.When) {
			latestTagCommit = commit
			latestTagName = tagRef.Name().String()
		}

		return nil
	})
	if err != nil {
		return "", err
	}
	return latestTagName, nil
}