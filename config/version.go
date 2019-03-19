package config
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