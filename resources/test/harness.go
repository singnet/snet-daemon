package test

import (
	"encoding/json"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

type signedJob struct {
	address   string
	signature string
}

type harness struct {
	t              testing.TB
	blockchainPath string
	truffleEnv     []string
	buildStatePath string
}

type jobFile struct {
	Job string
}

type jobInvocationFile struct {
	Signature string
}

func (h harness) createSignedJob() signedJob {
	runCommand(h.blockchainPath, h.truffleEnv, "npm", "run", "create-job").Wait()
	runCommand(h.blockchainPath, h.truffleEnv, "npm", "run", "fund-job").Wait()
	runCommand(h.blockchainPath, h.truffleEnv, "npm", "run", "sign-job").Wait()

	f, err := os.Open(path.Join(h.buildStatePath, "JobAddress.json"))
	require.NoError(h.t, err)

	jFile := jobFile{}
	dec := json.NewDecoder(f)
	require.NoError(h.t, dec.Decode(&jFile))

	f, err = os.Open(path.Join(h.buildStatePath, "JobInvocation.json"))
	require.NoError(h.t, err)

	dec = json.NewDecoder(f)
	jIFile := jobInvocationFile{}
	require.NoError(h.t, dec.Decode(&jIFile))

	return signedJob{
		address:   jFile.Job,
		signature: jIFile.Signature,
	}
}
