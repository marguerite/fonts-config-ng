package dirutils

import (
	"fmt"
	"github.com/marguerite/util/slice"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// ErrNonExistTarget ErrNonExistTarget is used to indicate the target a symlink points to actually does not exist on the filesystem.
type ErrNonExistTarget struct {
	Path string
	Link string
}

func (e ErrNonExistTarget) Error() string {
	return e.Path + "points to an non-existent target " + e.Link
}

func pathSeparator() string {
	if runtime.GOOS == "windows" {
		return "\\"
	}
	return "/"
}

// ReadSymlink follows the path of the symlink recursively and finds out the target it finally points to.
func ReadSymlink(path string) (string, error) {
	link, err := os.Readlink(path)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(link) {
		link = filepath.Join(filepath.Dir(path), link)
	}
	f, err := os.Stat(link)
	if err != nil {
		return "", ErrNonExistTarget{path, link}
	}
	if f.Mode()&os.ModeSymlink != 0 {
		return ReadSymlink(link)
	}
	return link, nil
}

// parse **/*, * and {,}
func ParseWildcard(s string) (string, []*regexp.Regexp) {
	r := []*regexp.Regexp{}
	fn := func(p string) *regexp.Regexp {
		p = strings.Replace(p, "**", "*", -1)
		p = strings.Replace(p, "*"+pathSeparator()+"*", "*", -1)
		p = strings.Replace(p, "*", ".*", -1)
		p = strings.Replace(p, "\\", "\\\\", -1)
		return regexp.MustCompile("^" + p + "$")
	}

	fn1 := func(p string) string {
		if strings.HasSuffix(p, pathSeparator()) {
			return p
		}
		return filepath.Dir(p)
	}

	// /usr/lib/xxx.{a, la}
	re := regexp.MustCompile(`([^{]+){([^}]+)}(.*)?`)
	m := re.FindStringSubmatch(s)
	if len(m) != 0 {
		for _, v := range strings.Split(m[2], ",") {
			ns := m[1] + strings.TrimSpace(v)
			if len(m[3]) != 0 {
				ns += m[3]
			}
			r = append(r, fn(ns))
		}
		return fn1(m[1]), r
	}
	if strings.Contains(s, "*") {
		r = append(r, fn(s))
		return fn1(strings.Split(s, "*")[0]), r
	}
	return s, r
}

func ls(d string, kind string) ([]string, error) {
	d1, _ := filepath.Abs(d)
	dir, r := ParseWildcard(d1)

	f, err := os.Stat(dir)
	if err != nil {
		return []string{}, err
	}
	if f.Mode().IsRegular() {
		return []string{dir}, nil
	}

	files := []string{}
	e := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				fmt.Println("WARNING: no permission to visit " + path + ", skipped")
				return nil
			}
			return err
		}

		// filter
		ok := false
		if len(r) != 0 {
			for _, re := range r {
				if re.MatchString(path) {
					ok = true
				}
			}
		} else {
			ok = true
		}

		if !ok {
			return nil
		}

		switch kind {
		case "dir":
			if info.IsDir() {
				files = append(files, path)
			}
			// the symlinks to directories
			if info.Mode()&os.ModeSymlink != 0 {
				link, err := ReadSymlink(path)
				if err != nil {
					if _, ok := err.(ErrNonExistTarget); ok {
						// the symlink points to an non-existent target, ignore
						fmt.Println("WARNING: " + path + " points to an non-existent target " + link)
						return nil
					}
					return err
				}
				f, _ := os.Stat(link)
				if f.IsDir() {
					files = append(files, path)
				}
			}
		case "symlink":
			if info.Mode()&os.ModeSymlink != 0 {
				files = append(files, path)
			}
		default:
			if info.Mode().IsRegular() {
				files = append(files, path)
			}
			// the symlinks to actual files
			if info.Mode()&os.ModeSymlink != 0 {
				link, err := ReadSymlink(path)
				if err != nil {
					if _, ok := err.(ErrNonExistTarget); ok {
						// the symlink points to an non-existent target, ignore
						fmt.Println("WARNING: " + path + " points to an non-existent target " + link)
						return nil
					}
					return err
				}
				f, _ := os.Stat(link)
				if f.Mode().IsRegular() {
					files = append(files, link)
				}
			}
		}
		return nil
	})
	return files, e
}

// Ls Takes a directory and the kind of file to be listed, returns the list of file and the possible error. Kind supports: dir, symlink, defaults to file.
func Ls(d string, kinds ...string) ([]string, error) {

	if len(kinds) == 0 {
		return ls(d, "")
	}

	if len(kinds) == 1 {
		return ls(d, kinds[0])
	}

	f := []string{}
	for _, kind := range kinds {
		i, err := ls(d, kind)
		if err != nil {
			// f is incomplete
			return f, err
		}
		slice.Concat(&f, i)
	}
	return f, nil
}

// MkdirP create directories for path
func MkdirP(path string) error {
	fmt.Println("Creating directory: " + path)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, os.ModeDir)
		if err != nil {
			fmt.Println("Can not create directory " + path)
			return err
		}
		fmt.Println(path + " created")
	} else {
		fmt.Println(path + " exists already")
		return nil
	}
	return nil
}

func parsePattern(re []*regexp.Regexp, pattern interface{}) []*regexp.Regexp {
	switch v := pattern.(type) {
	case *regexp.Regexp:
		re = append(re, v)
	case []*regexp.Regexp:
		for _, i := range v {
			re = append(re, i)
		}
	case string:
		re = append(re, regexp.MustCompile(v))
	case []string:
		for _, i := range v {
			re = append(re, regexp.MustCompile(i))
		}
	default:
		fmt.Println("Unsupported pattern type. Supported: *regexp.Regexp, []*regexp.Regexp, string, []string.")
		os.Exit(1)
	}
	return re
}

// Glob return files in `dir` directory that matches `pattern`. can pass `ex` to exclude file from the matches. ex's expanded regex number can be zero (no exclusion), one (test against every expanded regex in pattern), or equals to the number of expanded regex in pattern(one exclude regex refers to one match regex). expanded regex number, eg: [".*\\.yaml","opencc\\/.*"] is one slice param, but the expanded regex number will be two.
func Glob(dir string, pattern interface{}, ex ...interface{}) ([]string, error) {
	return glob(dir, pattern, Ls, ex...)
}

// fn: used to pass a test function or a real function that involves file operations.
func glob(dir string, pattern interface{}, fn func(string, ...string) ([]string, error), ex ...interface{}) ([]string, error) {
	files, err := fn(dir)
	if err != nil {
		return []string{}, err
	}

	re := []*regexp.Regexp{}
	re = parsePattern(re, pattern)
	re1 := []*regexp.Regexp{}
	if len(ex) > 0 {
		// ex's type is []interface{}, need to assert to actual type first
		for _, v := range ex {
			re1 = parsePattern(re1, v)
		}
	}

	if len(re) != len(re1) && len(re1) > 1 {
		fmt.Println("We just support exclude regex whose number matches zero, one or the number of regex in pattern.")
		os.Exit(1)
	}

	m := []string{}
	for _, f := range files {
		for i, r := range re {
			if r.MatchString(f) {
				if len(re1) == 0 {
					m = append(m, f)
				} else {
					if len(re1) == 1 {
						if !re1[0].MatchString(f) && len(re1[0].String()) != 0 {
							m = append(m, f)
						}
					} else {
						if !re1[i].MatchString(f) && len(re1[i].String()) != 0 {
							m = append(m, f)
						}
					}
				}

			}
		}
	}
	return m, nil
}
