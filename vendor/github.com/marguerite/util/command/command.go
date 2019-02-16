package command

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// Debug a trigger for debug output
const Debug int = 256

// Search search if an executable exists
func Search(cmd string, verbosity int) (string, bool) {
	if _, err := os.Stat(cmd); os.IsNotExist(err) {
		if verbosity >= Debug {
			fmt.Printf("--- WARNING: no executable from %s found\n", cmd)
		}
		return cmd, false
	}
	return cmd, true
}

func cmdOptionToString(opts []string) string {
	str := ""
	for _, s := range opts {
		str += s + " "
	}
	return str
}

// Run run command with options, returns output, ExitStatus and error
func Run(cmd string, opts []string, verbosity int) (string, int, error) {
	out, err := exec.Command(cmd, opts...).Output()
	status := 0

	if verbosity >= Debug {
		fmt.Printf("--- executing: %s %s\n", cmd, cmdOptionToString(opts))
	}

	if err != nil {
		if msg, ok := err.(*exec.Error); ok {
			return string(out), -1, fmt.Errorf(msg.Error())
		}

		if msg, ok := err.(*exec.ExitError); ok {
			if waitStatus, ok := msg.Sys().(syscall.WaitStatus); ok {
				return string(out), waitStatus.ExitStatus(), err
			}
		}

		return string(out), -1, err
	}

	return string(out), status, err
}
