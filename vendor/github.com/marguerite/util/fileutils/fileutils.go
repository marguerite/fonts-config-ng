package fileutils

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
)

// Debug a trigger for debug output
const Debug int = 256

func Touch(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		f, err := os.Create(path)
		if err != nil {
			if os.IsPermission(err) {
				fmt.Printf("WARNING: no permission to create %s, skipped...\n", path)
				return nil
			}
			return err
		}
		defer f.Close()
	}
	return nil
}

// Remove remove a file
func Remove(f string) error {
	if _, err := os.Stat(f); os.IsNotExist(err) {
		return fmt.Errorf("Error 1: %s doesn't exist.\n", f)
	}
	err := os.Remove(f)
	if err != nil {
		return fmt.Errorf("Error 2: can not delete %s, patherr: %s.\n", f, err.Error())
	}
	return nil
}

func Copy(src, dst string) error {
	stat, err := os.Stat(src)
	if err != nil {
		return err
	}

	// source is a directory
	if stat.IsDir() {
		return fmt.Errorf("%s is a directory", src)
	}

	// source is a symlink
	if stat.Mode()&os.ModeSymlink == os.ModeSymlink {
		fmt.Printf("%s is a symlink, following the original one", src)
		org, err := os.Readlink(src)
		if err != nil {
			return err
		}
		src = org
	}

	if info, err := os.Stat(dst); !os.IsNotExist(err) {
		// dst is a directory
		if info.IsDir() {
			basename := filepath.Base(src)
			dst = filepath.Join(dst, basename)
		} else {
			err = os.Remove(dst)
			if err != nil {
				return err
			}
		}
	}

	in, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(dst, in, stat.Mode())
	if err != nil {
		return err
	}
	return nil
}

// HasPrefixSuffixInGroup if a string's prefix/suffix matches one in group
// b trigger's prefix match
func HasPrefixSuffixInGroup(s string, group []string, b bool) bool {
	prefix := "(?i)"
	suffix := ""
	if b {
		prefix += "^"
	} else {
		suffix += "$"
	}

	for _, v := range group {
		re := regexp.MustCompile(prefix + regexp.QuoteMeta(v) + suffix)
		if re.MatchString(s) {
			return true
		}
	}
	return false
}
