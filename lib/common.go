package lib

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// Debug echo debug level information to output
const Debug int = 256

// Verbose echo verbose level information to output
const Verbose int = 1

// Quiet echo nothing to output
const Quiet int = 0

// FcSuffix suffix for every fontconfig configuration file
const FcSuffix string = "</fontconfig>\n"

// Dbg if dbgLevel >= limit, return the dbgOut. dbgOut can be plain string or func to format debug information by yourself
func Dbg(verbosity int, level int, dbgOut interface{}, parms ...interface{}) {
	if verbosity >= level {
		kd := reflect.TypeOf(dbgOut).Kind()
		if kd == reflect.Func {
			args := []reflect.Value{}
			for _, parm := range parms {
				args = append(args, reflect.ValueOf(parm))
			}
			out := reflect.ValueOf(dbgOut).Call(args)
			// may have multiple returns
			for _, o := range out {
				fmt.Println(o)
			}
		}
		if kd == reflect.String {
			fmt.Println(dbgOut)
		}
	}
}

// GetFcConfig return fontconfig file locations
func GetFcConfig(c string, userMode bool) string {
	m := map[string][]string{
		"render":      []string{"10-rendering-options.conf", "rendering-options.conf"},
		"fpl":         []string{"58-family-prefer-local.conf", "family-prefer.conf"},
		"blacklist":   []string{"81-emoji-blacklist-glyphs.conf", "emoji-blacklist-glyphs.conf"},
		"tt":          []string{"10-group-tt-hinted-fonts.conf", "tt-hinted-fonts.conf"},
		"nonTT":       []string{"10-group-tt-non-hinted-fonts.conf", "tt-non-hinted-fonts.conf"},
		"notoDefault": []string{"49-family-default-noto.conf", "family-default-noto.conf"},
		"notoPrefer":  []string{"59-family-prefer-lang-specific-noto.conf", "family-prefer-lang-specific-noto.conf"},
		"cjk":         []string{"59-family-prefer-lang-specific-cjk.conf", "family-prefer-lang-specific-cjk.conf"},
	}

	if userMode {
		return filepath.Join(os.Getenv("HOME"), ".config/fontconfig", m[c][1])
	}

	prefix := "/etc/sysconfig"
	if strings.HasSuffix(m[c][0], ".conf") {
		prefix = "/etc/fonts/conf.d"
	}
	return filepath.Join(prefix, m[c][0])
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

//overwriteOrRemoveFile Overwrite file with new content or completely remove the file.
func overwriteOrRemoveFile(path string, content []byte) error {
	os.Remove(path)
	if len(content) == 0 {
		return nil
	}
	f, err := os.Create(path)
	defer f.Close()
	if err != nil {
		return err
	}
	n, err := f.Write(content)
	if n != len(content) {
		return fmt.Errorf("not fully written")
	}
	if err != nil {
		return err
	}
	return nil
}
