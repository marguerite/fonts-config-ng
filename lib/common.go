package lib

import (
	"bytes"
	"github.com/marguerite/util/dirutils"
	"github.com/marguerite/util/fileutils"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
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

// Location system locations
type Location struct {
	System string
	User   string
}

// GenConfigLocation return config file locations
func GenConfigLocation(c string, userMode bool) string {
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

//ReadFontFiles Reads global and local fonts by default, can add restricts in format of string or *regexp.Regexp,
//will return the matched ones only.
func ReadFontFiles(restricts ...interface{}) []string {
	_, ok := restricts[0].(*regexp.Regexp)
	if reflect.ValueOf(restricts[0]).Kind() != reflect.String && !ok {
		log.Fatal("Restricts must be string or *regexp.Regexp")
	}

	candidates := []string{}

	local := filepath.Join(GetEnv("HOME"), ".fonts")
	for _, dir := range []string{local, "/usr/share/fonts"} {
		fonts, _ := dirutils.Ls(dir)
		for _, font := range fonts {
			base := filepath.Base(font)
			if fileutils.HasPrefixOrSuffix(base, []string{"font", ".", ".dir"}) != 0 {
				continue
			}
			in := true
			if len(restricts) > 0 {
				for _, restrict := range restricts {
					if ok {
						re, _ := restrict.(*regexp.Regexp)
						if !re.MatchString(base) {
							in = false
						}
					} else {
						str, _ := restrict.(string)
						if !strings.Contains(base, str) {
							in = false
						}
					}
				}
			}
			if in {
				candidates = append(candidates, font)
			}
		}
	}

	return candidates
}
