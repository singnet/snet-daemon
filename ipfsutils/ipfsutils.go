package ipfsutils

import (
	"archive/tar"
	"fmt"
	"github.com/ipfs/go-ipfs-api"
	"github.com/singnet/snet-daemon/config"
	log "github.com/sirupsen/logrus"
	"io"
	"strings"
	"time"
)

// to read all files which have been compressed, PS there can be more than one file
// We need to start reading the proto files associated with the service.
// proto files are compressed and stored as modelipfsHash
func ReadFilesCompressed(compressedFile string) (protofiles []string, err error) {
	f := strings.NewReader(compressedFile)
	tarReader := tar.NewReader(f)
	protofiles = make([]string, 0)
	for true {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.WithError(err)
			return nil, err
		}
		name := header.Name
		switch header.Typeflag {
		case tar.TypeDir:
			log.WithField("Directory Name", name).Debug("Directory name ")
		case tar.TypeReg:
			log.WithField("file Name:", name).Debug("File name ")
			data := make([]byte, header.Size)
			_, err := tarReader.Read(data)
			if err != nil && err != io.EOF {
				log.WithError(err)
				return nil, err
			}
			protofiles = append(protofiles, string(data))
		default:
			err = fmt.Errorf(fmt.Sprintf("%s : %c %s %s\n",
				"Unknown file Type ",
				header.Typeflag,
				"in file",
				name,
			))
			log.WithError(err)
			return nil, err
		}
	}
	return protofiles, nil
}

func GetIpfsFile(hash string) string {

	log.WithField("hash", hash).Debug("Hash Used to retrieve from IPFS")

	sh := GetIpfsShell()
	cid, err := sh.Cat(hash)
	if err != nil {
		log.WithError(err).WithField("hashFromMetaData", hash).Panic("error executing the cat command in ipfs")
	}

	blob, err := io.ReadAll(cid)
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
