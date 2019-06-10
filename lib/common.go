package lib

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

// VerbosityDebug an interger to control verbose output
const VerbosityDebug int = 256

// VerbosityVerbose an interger to control verbose output
const VerbosityVerbose int = 1

// VerbosityQuiet an interger to control verbose output
const VerbosityQuiet int = 0

func debug(verbosity int, level int, text string) {
	if verbosity >= level {
		fmt.Printf(text)
	}
}

func GetEnv(env string) string {
	val, ok := os.LookupEnv(env)
	if !ok {
		log.Fatalf("Environment Variable %s not set.", env)
	}
	if len(val) == 0 {
		log.Fatalf("Environment Variable %s is empty.", env)
	}
	return val
}

// ErrChk panic at error
func ErrChk(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

// FileOp return an io.ReadWriter for futher r/w operations
func FileOp(f string) io.ReadWriter {
	op, e := os.Open(f)
	if e != nil {
		log.Fatalf("Cannot open %s: %s\n", f, e.Error())
	}
	return op
}

// SysconfigLoc get fonts-config sysconfig location
func SysconfigLoc(userMode bool) string {
	if userMode {
		return filepath.Join(GetEnv("HOME"), ".config/fontconfig/fonts-config")
	}
	return "/etc/sysconfig/fonts-config"
}
