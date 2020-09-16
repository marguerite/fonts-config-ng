package command

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// Environ safely get an environment variable
func Environ(env string) (string, error) {
	val, ok := os.LookupEnv(env)
	if !ok {
		return "", fmt.Errorf("%s not set.", env)
	}
	if len(val) == 0 {
		return val, fmt.Errorf("%s is empty.", env)
	}
	return val, nil
}

// Search if an executable exists
func Search(cmd string) (string, error) {
	f, err := os.Stat(cmd)
	if err != nil {
		if os.IsNotExist(err) {
			// look in $Path
			f1, err1 := exec.LookPath(cmd)
			if err1 != nil {
				return "", fmt.Errorf("%s doesn't exist in current directory or $PATH.", cmd)
			}
			return f1, nil
		}
		return "", fmt.Errorf("Another unhandled non-IsNotExist PathError occurs %s", err.Error())
	}
	if f.IsDir() {
		return "", fmt.Errorf("%s is a directory", cmd)
	}
	if isExecutable(f) {
		return cmd, nil
	}
	return "", fmt.Errorf("%s is a file but has no exec permission", cmd)
}

// Run run command with options, returns output, ExitStatus and error
func Run(cmd string, opts ...string) (string, int, error) {
	out, err := exec.Command(cmd, opts...).Output()

	fmt.Printf("Executing: %s %s\n", cmd, strings.Join(opts, " "))

	if err != nil {
		if _, ok := err.(*exec.Error); ok {
			return string(out), -1, err
		}

		if msg, ok := err.(*exec.ExitError); ok {
			if waitStatus, ok := msg.Sys().(syscall.WaitStatus); ok {
				return string(out), waitStatus.ExitStatus(), err
			}
		}

		return string(out), -1, err
	}

	return string(out), 0, nil
}
