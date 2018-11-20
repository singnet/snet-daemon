package ipfsutils

import (
	"github.com/ipfs/go-ipfs-api"
	"github.com/singnet/snet-daemon/config"

	log "github.com/sirupsen/logrus"
	"io/ioutil"

	"regexp"
)

func GetIpfsFile(hash string) string {

	reg, err := regexp.Compile("ipfs://")
	if err != nil {
		log.Fatal(err)
	}
	hash = reg.ReplaceAllString(hash, "")

	reg, err = regexp.Compile("[^a-zA-Z0-9=]+")
	if err != nil {
		log.Fatal(err)
	}
	hash = reg.ReplaceAllString(hash, "")

	jsondata := string(hash)

	re := regexp.MustCompile("\\n")
	sh := GetIpfsShell()
	cid, err := sh.Cat(hash)
	if err != nil {
		log.WithError(err).Panic("error executing the cat command in ipfs")
	}

	blob, err := ioutil.ReadAll(cid)
	if err != nil {
		log.WithError(err).Panic("error: in Reading the meta data file %s", err)

	}
	log.Debug(string(blob))

	jsondata = string(blob)
	re = regexp.MustCompile("\\n")
	jsondata = re.ReplaceAllString(jsondata, " ")
	cid.Close()
	return jsondata
}

func GetIpfsShell() *shell.Shell {
	//Read from Configuration file
	sh := shell.NewShell(config.GetString(config.IpfsEndPoint))
	return sh
}
