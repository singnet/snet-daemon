package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

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
	"SNET_LOG_LEVEL=5",
	"SNET_POLL_SLEEP=1s",
	"SNET_WIRE_ENCODING=json",
}

func TestEndToEnd(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err, "Unable to determine working directory")

	blockchainPath := filepath.Join(cwd, "..", "blockchain")
	buildPath := filepath.Join(blockchainPath, "build")
	buildStatePath := filepath.Join(buildPath, "state")
	nodePath := filepath.Join(blockchainPath, "node_modules")

	runCommand("", nil, "rm", "-rf", "snetd.db").Wait()
	runCommand("", nil, "rm", "-rf", buildPath).Wait()

	require.NoError(t, runCommand("", nil, "go", "build", "-o",
		filepath.Join(buildPath, "snetd"),
		filepath.Join(cwd, "..", "..", "snetd", "main.go"),
	).Wait(), "Unable to build snetd")

	runCommand(blockchainPath, nil, "npm", "run", "create-mnemonic").Wait()
	rawMnemonic, err := ioutil.ReadFile(buildStatePath + "/Mnemonic.json")
	if err != nil {
		log.WithError(err).Panic()
	}
	mFile := &mnemonicFile{}
	err = json.Unmarshal(rawMnemonic, mFile)
	require.NoError(t, err)

	ganachePort := pickAvailablePort()
	truffleEnv := []string{
		"DAEMON_GANACHE_PORT=" + ganachePort,
	}

	daemonPort := pickAvailablePort()

	testConfiguration = append(
		testConfiguration,
		"SNET_ETHEREUM_JSON_RPC_ENDPOINT=http://127.0.0.1:"+ganachePort,
		"SNET_DAEMON_LISTENING_PORT="+daemonPort,
	)

	ganacheCmd := runCommand(nodePath, nil, "./.bin/ganache-cli", "-m", mFile.Mnemonic, "--port", ganachePort)
	defer ganacheCmd.Process.Wait()
	defer ganacheCmd.Process.Signal(syscall.SIGTERM)

	runCommand(blockchainPath, truffleEnv, "npm", "run", "migrate").Wait()
	runCommand(blockchainPath, truffleEnv, "npm", "run", "create-agent").Wait()

	rawAgent, err := ioutil.ReadFile(buildStatePath + "/AgentAddress.json")
	require.NoError(t, err)

	aFile := &agentFile{}
	err = json.Unmarshal(rawAgent, aFile)
	require.NoError(t, err)

	testConfiguration = append(testConfiguration, fmt.Sprintf("SNET_AGENT_CONTRACT_ADDRESS=%+v", aFile.Agent))
	testConfiguration = append(testConfiguration, fmt.Sprintf("SNET_HDWALLET_MNEMONIC=%+v", mFile.Mnemonic))
	snetdCmd := runCommand("", testConfiguration, filepath.Join(buildPath, "snetd"))
	defer func() {
		assert.NoError(t, snetdCmd.Wait(), "daemon exited non-zero")
	}()

	defer snetdCmd.Process.Signal(syscall.SIGTERM)

	h := harness{
		t:              t,
		blockchainPath: blockchainPath,
		truffleEnv:     truffleEnv,
		buildStatePath: buildStatePath,
	}
	time.Sleep(time.Second)

	header := func(msg string) {
		fmt.Println()
		fmt.Println("== " + msg + " ==")
		fmt.Println()
	}

	header("Testing native gRPC client")
	testGRPC(t, h, daemonPort)
	time.Sleep(time.Second)

	header("Testing gRPC-web client")
	testGRPCWeb(t, h, daemonPort)
	time.Sleep(time.Second)
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

func pickAvailablePort() string {
	p, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}
	addr := p.Addr().String()
	parts := strings.Split(addr, ":")
	if len(parts) < 2 {
		panic("Can't parse address: " + addr)
	}
	p.Close()

	return parts[len(parts)-1]
}
