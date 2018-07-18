package main

import (
	"os"
	"os/exec"
)

func main() {
	runCommand("resources/blockchain", nil, "npm", "install").Wait()
}

func runCommand(dir string, env []string, name string, arg ...string) *exec.Cmd {
	cmd := exec.Command(name, arg...)
	if dir != "" {
		cmd.Dir = dir
	}
	if env != nil && len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	cmd.Stderr = os.Stdout
	cmd.Stdout = os.Stdout
	cmd.Start()
	return cmd
}
