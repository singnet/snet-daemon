package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/singnet/snet-daemon/config"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	"SNET_POLL_SLEEP_SECS=1",
	"SNET_SERVICE_TYPE=grpc",
	"SMET_WIRE_ENCODING=json",
}

func TestEndToEnd(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err, "Unable to determine working directory")

	dbPath := config.GetString(config.DbPathKey)
	blockchainPath := filepath.Join(cwd, "..", "blockchain")
	buildPath := filepath.Join(blockchainPath, "build")
	buildStatePath := filepath.Join(buildPath, "state")
	nodePath := filepath.Join(blockchainPath, "node_modules")

	runCommand("", nil, "rm", "-rf", dbPath).Wait()
	runCommand("", nil, "rm", "-rf", buildPath).Wait()

	require.NoError(t, runCommand("", nil, "go", "build", "-o",
		filepath.Join(buildPath, "snetd"),
		filepath.Join(cwd, "..", "..", "snetd", "snetd.go"),
	).Wait(), "Unable to build snetd")

	runCommand(blockchainPath, nil, "npm", "run", "create-mnemonic").Wait()
	rawMnemonic, err := ioutil.ReadFile(buildStatePath + "/Mnemonic.json")
	if err != nil {
		log.WithError(err).Panic()
	}
	mFile := &mnemonicFile{}
	err = json.Unmarshal(rawMnemonic, mFile)
	require.NoError(t, err)

	ganacheCmd := runCommand(nodePath, nil, "./.bin/ganache-cli", "-m", mFile.Mnemonic)
	defer ganacheCmd.Process.Wait()
	defer ganacheCmd.Process.Signal(syscall.SIGTERM)

	runCommand(blockchainPath, nil, "npm", "run", "migrate").Wait()
	runCommand(blockchainPath, nil, "npm", "run", "create-agent").Wait()

	rawAgent, err := ioutil.ReadFile(buildStatePath + "/AgentAddress.json")
	require.NoError(t, err)

	aFile := &agentFile{}
	err = json.Unmarshal(rawAgent, aFile)
	require.NoError(t, err)

	testConfiguration = append(testConfiguration, fmt.Sprintf("SNET_AGENT_CONTRACT_ADDRESS=%+v", aFile.Agent))
	testConfiguration = append(testConfiguration, fmt.Sprintf("SNET_HDWALLET_MNEMONIC=%+v", mFile.Mnemonic))
	snetdCmd := runCommand("", testConfiguration, filepath.Join(buildPath, "snetd"))
	defer snetdCmd.Wait()
	defer snetdCmd.Process.Signal(syscall.SIGTERM)
	time.Sleep(2 * time.Second)

	runCommand(blockchainPath, nil, "npm", "run", "create-job").Wait()
	runCommand(blockchainPath, nil, "npm", "run", "fund-job").Wait()
	runCommand(blockchainPath, nil, "npm", "run", "sign-job").Wait()
	rawJob, err := ioutil.ReadFile(buildStatePath + "/JobAddress.json")
	require.NoError(t, err)

	jFile := &jobFile{}
	err = json.Unmarshal(rawJob, jFile)
	require.NoError(t, err)

	rawJobInvocation, err := ioutil.ReadFile(buildStatePath + "/JobInvocation.json")
	require.NoError(t, err)

	jIFile := &jobInvocationFile{}
	err = json.Unmarshal(rawJobInvocation, jIFile)
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	httpReq, err := http.NewRequest("POST", "http://127.0.0.1:5000/FakeService/FakeMethod",
		bytes.NewBuffer([]byte("\x00\x00\x00\x00\x13"+`{"hello":"goodbye"}`)))
	require.NoError(t, err)

	httpReq.Header.Set("content-type", "application/grpc-web+json")
	httpReq.Header.Set("snet-job-address", jFile.Job)
	httpReq.Header.Set("snet-job-signature", jIFile.Signature)

	httpResp, err := http.DefaultClient.Do(httpReq)
	require.NoError(t, err)

	grpcMessage := httpResp.Header.Get("Grpc-Message")
	assert.Empty(t, grpcMessage, "Got non-empty Grpc-Message on response")

	grpcStatus := httpResp.Header.Get("Grpc-Status")
	assert.Empty(t, grpcStatus, "Got non-empty Grpc-Status code on response")

	httpRespBytes, err := ioutil.ReadAll(httpResp.Body)
	fmt.Print(string(httpRespBytes))
	assert.NotEmpty(t, httpRespBytes, "Expected response body from daemon")

	time.Sleep(2 * time.Second)
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
