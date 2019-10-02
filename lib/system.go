package lib

import (
	"fmt"
	"github.com/marguerite/util/command"
	"regexp"
	"strings"
)

// FcCache run fc-cache command on the running system
func FcCache(verbosity int) {
	if cmd, err := command.Search("/usr/bin/fc-cache"); err == nil {
		debug(verbosity, VerbosityVerbose, "Creating fontconfig cache files.\n")

		opts := ""

		if verbosity >= VerbosityVerbose {
			opts = "--verbose"
		}

		_, status, _ := command.Run(cmd, opts)

		debug(verbosity, VerbosityDebug, fmt.Sprintf("Exit status of fc-cache: %d\n", status))
	}
}

// FpRehash run xset fp rehash on the running system
func FpRehash(verbosity int) {
	if cmd, err := command.Search("/usr/bin/xset"); err == nil {
		re := regexp.MustCompile(`^:\d.*$`)
		if re.MatchString(GetEnv("DISPLAY")) {
			debug(verbosity, VerbosityVerbose, "Rereading the font databases in the current font path ...\n")
			debug(verbosity, VerbosityDebug, "Running xset fp rehash\n")

			out, _, _ := command.Run(cmd, "fp", "rehash")
			debug(verbosity, VerbosityDebug, string(out)+"\n")
		} else {
			debug(verbosity, VerbosityVerbose, "It is not a local display, do not reread X font databases for now.\n")
			debug(verbosity, VerbosityDebug, "NOTE: do not run 'xset fp rehash', no local display detected.\n")
		}
	}
}

// ReloadXfsConfig reload Xorg Font Server on the running system
func ReloadXfsConfig(verbosity int) {
	if cmd, err := command.Search("/usr/bin/ps"); err == nil {
		pid, _, _ := command.Run(cmd, "-C", "xfs", "-o", "pid=")
		pid = strings.TrimSpace(pid)
		if len(pid) != 0 {
			debug(verbosity, VerbosityVerbose, fmt.Sprintf("Reloading config file of X Font Server %s ...\n", pid))
			command.Run("/usr/bin/pkill", "-USR1", pid)
		} else {
			debug(verbosity, VerbosityDebug, "X Font Server not used.\n")
		}
	} else {
		debug(verbosity, VerbosityVerbose, "WARNING: ps command is missing, couldn't search for X Font Server pids.")
	}
}
