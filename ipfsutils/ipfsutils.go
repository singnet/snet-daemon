package ipfsutils

import (
	"github.com/ipfs/go-ipfs-api"
	"github.com/singnet/snet-daemon/config"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"strings"
	"time"
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
		log.WithError(err).WithField("hashFromMetaData", hash).Panicf("error: in Reading the meta data file %s", err)

	}
	log.WithField("hash", hash).WithField("blob", string(blob)).Debug("Blob of IPFS file with hash")

	jsondata := string(blob)

	//validating the file read from IPFS
	newHash, err := sh.Add(strings.NewReader(jsondata), shell.OnlyHash(true))
	if err != nil {
		log.WithError(err).Panicf("error in generating the hash for the meta data read from IPFS : %v", err)
	}
	if newHash != hash {
		log.WithError(err).WithField("hashFromIPFSContent", newHash).
			Panicf("IPFS hash verification failed. Generated hash doesnt match with expected hash %s", hash)
	}

	cid.Close()
	return jsondata
}

func GetIpfsShell() *shell.Shell {
	sh := shell.NewShell(config.GetString(config.IpfsEndPoint))
	// sets the timeout for accessing the ipfs content
	sh.SetTimeout(time.Duration(config.GetInt(config.IpfsTimeout)) * time.Second)
	return sh
}
