package lib

import (
	"fmt"
	"github.com/marguerite/util/command"
	"os"
	"regexp"
	"strings"
)

// FcCache run fc-cache command on the running system
func FcCache(verbosity int) {
	if cmd, ok := command.Search("/usr/bin/fc-cache", verbosity); ok {
		cmdOpts := []string{}
		if verbosity >= VerbosityVerbose {
			cmdOpts = append(cmdOpts, "--verbose")
		}
		debug(verbosity, VerbosityVerbose, "Creating fontconfig cache files.\n")

		_, status, _ := command.Run(cmd, cmdOpts, verbosity)

		debug(verbosity, VerbosityDebug, fmt.Sprintf("exit status of fc-cache: %d\n", status))
	}
}

// FpRehash run xset fp rehash on the running system
func FpRehash(verbosity int) {
	if cmd, ok := command.Search("/usr/bin/xset", verbosity); ok {
		re := regexp.MustCompile(`^:\d.*$`)
		if re.MatchString(os.Getenv("DISPLAY")) {
			cmdOpts := []string{"fp", "rehash"}
			debug(verbosity, VerbosityVerbose, "Rereading the font databases in the current font path ...\n")
			debug(verbosity, VerbosityDebug, "--- running xset fp rehash\n")

			out, _, _ := command.Run(cmd, cmdOpts, verbosity)
			debug(verbosity, VerbosityDebug, string(out)+"\n")
		} else {
			debug(verbosity, VerbosityVerbose, "It is not a local display, do not reread X font databases for now.\n")
			debug(verbosity, VerbosityDebug, "--- NOTE: do not run 'xset fp rehash', no local display detected.\n")
		}
	}
}

// ReloadXfsConfig reload Xorg Font Server on the running system
func ReloadXfsConfig(verbosity int) {
	if cmd, ok := command.Search("/usr/bin/ps", verbosity); ok {
		cmdOpts := []string{"-C", "xfs", "-o", "pid="}
		pid, _, _ := command.Run(cmd, cmdOpts, verbosity)
		pid = strings.TrimSpace(pid)
		if len(pid) != 0 {
			debug(verbosity, VerbosityVerbose, fmt.Sprintf("Reloading config file of X Font Server %s ...\n", pid))
			command.Run("/usr/bin/pkill", []string{"-USR1", pid}, verbosity)
		} else {
			debug(verbosity, VerbosityDebug, "X Font Server not used.\n")
		}
	} else {
		debug(verbosity, VerbosityVerbose, "--- WARNING: ps command is missing, couldn't search for X Font Server pids.")
	}
}
