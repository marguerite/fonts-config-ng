package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

const (
	// Debug echo debug level information to output
	Debug int = 256
	// Verbose echo verbose level information to output
	Verbose int = 1
	// Quiet echo nothing to output
	Quiet int = 0
	// FcSuffix suffix for every fontconfig configuration file
	FcSuffix string = "</fontconfig>\n"
)

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
		"render":      {"10-rendering-options.conf", "rendering-options.conf"},
		"fpl":         {"58-family-prefer-local.conf", "family-prefer.conf"},
		"blacklist":   {"81-emoji-blacklist-glyphs.conf", "emoji-blacklist-glyphs.conf"},
		"notoDefault": {"49-family-default-noto.conf", "family-default-noto.conf"},
		"notoPrefer":  {"59-family-prefer-lang-specific-noto.conf", "family-prefer-lang-specific-noto.conf"},
		"cjk":         {"59-family-prefer-lang-specific-cjk.conf", "family-prefer-lang-specific-cjk.conf"},
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
