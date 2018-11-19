package ipfsutils

import (
	"github.com/ipfs/go-ipfs-api"

	log "github.com/sirupsen/logrus"
	"io/ioutil"

	"regexp"
)

func GetIpfsFile(hash string) string {
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

	jsondata := string(blob)
	re := regexp.MustCompile("\\n")
	jsondata = re.ReplaceAllString(jsondata, " ")
	cid.Close()
	return jsondata
}

func GetIpfsShell() *shell.Shell {
	//Read from Configuration file
	sh := shell.NewShell("http://localhost:5002/")
	return sh
}

func GetmpeAddress() string {
	return metaData.MpeAddress
}
