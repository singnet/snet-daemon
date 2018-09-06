package main

import (
	"os"
	"os/exec"
	"fmt"
	"log"
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
	err := cmd.Start()
	if err != nil {
		fmt.Errorf("command execution '%s' execution in dir '%s' fails.\n", name, dir)
		log.Fatal(err)
	}

	return cmd
}
