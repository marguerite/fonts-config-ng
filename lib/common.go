package lib

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// VerbosityDebug an interger to control verbose output
const VerbosityDebug int = 256

// VerbosityVerbose an interger to control verbose output
const VerbosityVerbose int = 1

// VerbosityQuiet an interger to control verbose output
const VerbosityQuiet int = 0

func debug(verbosity int, level int, text string) {
	if verbosity >= level {
		log.Println(text)
	}
}

// GetEnv get system environment variable
func GetEnv(env string) string {
	val, ok := os.LookupEnv(env)
	if !ok {
		log.Fatalf("Environment Variable %s not set.\n", env)
	}
	if len(val) == 0 {
		log.Fatalf("Environment Variable %s is empty.\n", env)
	}
	return val
}

// ErrChk panic at error
func ErrChk(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

//GetCacheLocation return fonts-config cache location
func GetCacheLocation(userMode bool) string {
	if userMode {
		return filepath.Join(GetEnv("HOME"), ".config/fontconfig/fonts-config.json")
	}
	return "/var/tmp/fonts-config.json"
}

// Location system locations
type Location struct {
	System string
	User   string
}

// GetConfigLocation return config file locations
func GetConfigLocation(c string, userMode bool) string {
	m := map[string]Location{
		"fc":        {"fonts-config", "fonts-config"},
		"render":    {"10-rendering-options.conf", "rendering-options.conf"},
		"fpl":       {"58-family-prefer-local.conf", "family-prefer.conf"},
		"dual":      {"20-fix-globaladvance.conf", "fix-globaladvance.conf"},
		"blacklist": {"81-emoji-blacklist-glyphs.conf", "emoji-blacklist-glyphs.conf"},
	}

	if userMode {
		return filepath.Join(GetEnv("HOME"), ".config/fontconfig/"+m[c].User)
	}

	prefix := "/etc/sysconfig"
	if strings.HasSuffix(m[c].System, ".conf") {
		prefix = "/etc/fonts/conf.d"
	}
	return filepath.Join(prefix, m[c].System)
}

// NewReader create an io.Reader from file
func NewReader(f string) *bytes.Buffer {
	dat, err := os.Open(f)
	if err != nil {
		log.Fatalf("can not open %s: \"%s\".\n", f, err.Error())
	}
	defer dat.Close()

	buf, err := ioutil.ReadAll(dat)
	if err != nil {
		log.Fatalf("can not read %s: \"%s\".\n", f, err.Error())
	}

	return bytes.NewBuffer(buf)
}

// persist Overwrite file with new content or completely remove the file.
func persist(file string, text []byte, perm os.FileMode) error {
	if len(text) == 0 {
		err := os.Remove(file)
		return err
	}
	err := ioutil.WriteFile(file, text, perm)
	return err
}
