package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/singnet/snet-daemon/config"
	log "github.com/sirupsen/logrus"
)

type mnemonicFile struct {
	Mnemonic string
}

type agentFile struct {
	Agent string
}

type jobFile struct {
	Job string
}

type jobInvocationFile struct {
	Signature string
}

var testConfiguration = []string{
	"SNET_BLOCKCHAIN_ENABLED=true",
	"SNET_DAEMON_LISTENING_PORT=5000",
	"SNET_DAEMON_TYPE=grpc",
	"SNET_DB_PATH=snetd.db",
	"SNET_ETHEREUM_JSON_RPC_ENDPOINT=http://127.0.0.1:8545",
	"SNET_HDWALLET_INDEX=0",
	"SNET_LOG_LEVEL=5",
	"SNET_PASSTHROUGH_ENABLED=false",
	"SNET_POLL_SLEEP_SECS=5",
	"SNET_SERVICE_TYPE=grpc",
	"SMET_WIRE_ENCODING=json",
}

func main() {
	dbPath := config.GetString(config.DbPathKey)
	buildPath := "resources/blockchain/build"
	buildStatePath := "resources/blockchain/build/state"
	blockchainPath := "resources/blockchain"
	nodePath := "resources/blockchain/node_modules"

	runCommand("", nil, "rm", "-rf", dbPath).Wait()
	runCommand("", nil, "rm", "-rf", buildPath).Wait()

	runCommand("", nil, "go", "build", "-o", "resources/blockchain/build/snetd",
		"snetd/snetd.go").Wait()

	runCommand(blockchainPath, nil, "npm", "run", "create-mnemonic").Wait()
	rawMnemonic, err := ioutil.ReadFile(buildStatePath + "/Mnemonic.json")
	if err != nil {
		log.WithError(err).Panic()
	}
	mFile := &mnemonicFile{}
	err = json.Unmarshal(rawMnemonic, mFile)
	if err != nil {
		log.WithError(err).Panic()
	}
	ganacheCmd := runCommand(nodePath, nil, "./.bin/ganache-cli", "-m", mFile.Mnemonic)
	defer ganacheCmd.Process.Wait()
	defer ganacheCmd.Process.Signal(syscall.SIGTERM)
	runCommand(blockchainPath, nil, "npm", "run", "migrate").Wait()
	runCommand(blockchainPath, nil, "npm", "run", "create-agent").Wait()
	rawAgent, err := ioutil.ReadFile(buildStatePath + "/AgentAddress.json")
	if err != nil {
		log.WithError(err).Error()
		return
	}
	aFile := &agentFile{}
	err = json.Unmarshal(rawAgent, aFile)
	if err != nil {
		log.WithError(err).Error()
		return
	}
	testConfiguration = append(testConfiguration, fmt.Sprintf("SNET_AGENT_CONTRACT_ADDRESS=%+v", aFile.Agent))
	testConfiguration = append(testConfiguration, fmt.Sprintf("SNET_HDWALLET_MNEMONIC=%+v", mFile.Mnemonic))
	snetdCmd := runCommand("", testConfiguration, "resources/blockchain/build/snetd")
	defer snetdCmd.Wait()
	defer snetdCmd.Process.Signal(syscall.SIGTERM)
	time.Sleep(time.Second * 7)
	runCommand(blockchainPath, nil, "npm", "run", "create-job").Wait()
	runCommand(blockchainPath, nil, "npm", "run", "fund-job").Wait()
	runCommand(blockchainPath, nil, "npm", "run", "sign-job").Wait()
	rawJob, err := ioutil.ReadFile(buildStatePath + "/JobAddress.json")
	if err != nil {
		log.WithError(err).Error()
		return
	}
	jFile := &jobFile{}
	err = json.Unmarshal(rawJob, jFile)
	if err != nil {
		log.WithError(err).Error()
		return
	}
	rawJobInvocation, err := ioutil.ReadFile(buildStatePath + "/JobInvocation.json")
	if err != nil {
		log.WithError(err).Error()
		return
	}
	jIFile := &jobInvocationFile{}
	err = json.Unmarshal(rawJobInvocation, jIFile)
	if err != nil {
		log.WithError(err).Error()
		return
	}
	time.Sleep(time.Second * 7)
	httpReq, err := http.NewRequest("POST", "http://127.0.0.1:5000/FakeService/FakeMethod",
		bytes.NewBuffer([]byte(`\x00\x00\x00\x00\x13{"hello":"goodbye"}`)))
	if err != nil {
		log.WithError(err).Error()
		return
	}
	httpReq.Header.Set("content-type", "application/grpc-web+json")
	httpReq.Header.Set("snet-job-address", jFile.Job)
	httpReq.Header.Set("snet-job-signature", jIFile.Signature)
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		log.WithError(err).Error()
		return
	}
	httpRespBytes, err := ioutil.ReadAll(httpResp.Body)
	fmt.Print(string(httpRespBytes))
	time.Sleep(time.Second * 7)
}

func runCommand(dir string, env []string, name string, arg ...string) *exec.Cmd {
	cmd := exec.Command(name, arg...)
	if dir != "" {
		cmd.Dir = dir
	}
	if env != nil && len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Start()
	return cmd
}
