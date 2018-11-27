package ipfsutils

import (
	"github.com/ipfs/go-ipfs-api"
	"github.com/singnet/snet-daemon/config"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
)

func GetIpfsFile(hash string) string {

	log.WithField("hash", hash).Debug("Hash Used to retrieve from IPFS")

	sh := GetIpfsShell()
	cid, err := sh.Cat(hash)
	if err != nil {
		log.WithError(err).WithField("hashFromMetaData", hash).Panic("error executing the cat command in ipfs")
	}

	blob, err := ioutil.ReadAll(cid)
	if err != nil {
		log.WithError(err).WithField("hashFromMetaData", hash).Panic("error: in Reading the meta data file %s", err)

	}
	log.WithField("hash", hash).WithField("blob", string(blob)).Debug("Blob of IPFS file with hash")

	jsondata := string(blob)

	cid.Close()
	return jsondata
}

func GetIpfsShell() *shell.Shell {
	sh := shell.NewShell(config.GetString(config.IpfsEndPoint))
	return sh
}
