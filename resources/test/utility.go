package main

import (
	"net"
	"os"
	"os/exec"
	"strings"
)

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
