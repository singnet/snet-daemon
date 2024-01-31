package ipfsutils

import (
	"archive/tar"
	"context"
	"fmt"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/kubo/client/rpc"
	"github.com/singnet/snet-daemon/config"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strings"
	"time"
)

// ReadFilesCompressed - read all files which have been compressed, there can be more than one file
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

func GetIpfsFile(hash string) (content string) {

	log.WithField("hash", hash).Debug("Hash Used to retrieve from IPFS")

	ipfsClient := GetIPFSClient()

	cID, err := cid.Parse(hash)
	if err != nil {
		log.WithError(err).WithField("hashFromMetaData", hash).Panic("error parsing the ipfs hash")
	}

	req := ipfsClient.Request("cat", cID.String())
	if err != nil {
		log.WithError(err).WithField("hashFromMetaData", hash).Panic("error executing the cat command in ipfs")
		return
	}
	resp, err := req.Send(context.Background())
	defer resp.Close()
	if err != nil {
		log.WithError(err).WithField("hashFromMetaData", hash).Panic("error executing the cat command in ipfs")
		return
	}
	if resp.Error != nil {
		log.WithError(err).WithField("hashFromMetaData", hash).Panic("error executing the cat command in ipfs")
		return
	}
	fileContent, err := io.ReadAll(resp.Output)
	if err != nil {
		log.WithError(err).WithField("hashFromMetaData", hash).Panicf("error: in Reading the meta data file %s", err)
		return
	}

	log.WithField("hash", hash).WithField("blob", string(fileContent)).Debug("Blob of IPFS file with hash")

	//sum, err := cID.Prefix().Sum(fileContent)
	//if err != nil {
	//	log.WithError(err).Panicf("error in generating the hash for the meta data read from IPFS : %v", err)
	//}

	// Create a cid manually to check cid
	_, c, err := cid.CidFromBytes(append(cID.Bytes(), fileContent...))
	if err != nil {
		log.WithError(err).WithField("hashFromMetaData", hash).Panic("error generating ipfs hash")
		return
	}

	// To test if two cid's are equivalent, be sure to use the 'Equals' method:
	if !c.Equals(cID) {
		log.WithError(err).WithField("hashFromIPFSContent", c.String()).Panicf("IPFS hash verification failed. Generated hash doesnt match with expected hash %s", hash)
	}

	return string(fileContent)
}

func GetIPFSClient() *rpc.HttpApi {
	httpClient := http.Client{
		Timeout: time.Duration(config.GetInt(config.IpfsTimeout)) * time.Second,
	}
	ifpsClient, err := rpc.NewURLApiWithClient(config.GetString(config.IpfsEndPoint), &httpClient)
	if err != nil {
		log.WithError(err).Panicf("Connection failed to IPFS: %s", config.GetString(config.IpfsEndPoint))
	}
	return ifpsClient
}
