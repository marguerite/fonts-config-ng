package fileutils

import (
	"errors"
	"fmt"
	"github.com/marguerite/util/dirutils"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
)

// Touch a file or directory
func Touch(path string, isDir ...bool) error {
	// process isDir, we don't allow > 1 arguments.
	if len(isDir) > 1 {
		return errors.New("isDir is a symbol indicating whether the path is a target directory. You shall not pass two arguments.")
	}
	ok := false
	if len(isDir) == 1 {
		ok = isDir[0]
	}

	_, err := os.Stat(path)

	if err != nil {
		if os.IsNotExist(err) {
			if ok {
				err := dirutils.MkdirP(path)
				return err
			}

			dir := filepath.Dir(path)
			if dir != "." {
				err := dirutils.MkdirP(dir)
				if err != nil {
					fmt.Println("Can not create containing directory " + dir)
					return err
				}
			}
			f, err := os.Create(path)
			defer f.Close()
			if err != nil {
				if os.IsPermission(err) {
					fmt.Println("WARNING: no permission to create " + path + ", skipped...")
					return nil
				}
				return err
			}
		} else {
			return fmt.Errorf("Another unhandled non-IsNotExist PathError occurs: %s", err.Error())
		}
	}

	return nil
}

func cp(f, dst, orig string) ([]string, error) {
	// f always exists
	fi, _ := os.Stat(f)
	in, err := ioutil.ReadFile(f)
	if err != nil {
		return []string{}, err
	}

	di, err := os.Stat(dst)
	// just copy for non-existent target
	if os.IsNotExist(err) {
		err1 := ioutil.WriteFile(dst, in, fi.Mode())
		if err1 != nil {
			return []string{}, err1
		}
		return []string{dst}, nil
	} else {
		// dst here can only be file or dir because previous ReadSymlink in copy()
		if di.Mode().IsDir() {
			dst = filepath.Join(dst, filepath.Base(f))
			err1 := ioutil.WriteFile(dst, in, fi.Mode())
			if err1 != nil {
				return []string{}, err1
			}
			return []string{dst}, nil
		} else {
			err1 := ioutil.WriteFile(dst, in, fi.Mode())
			if err1 != nil {
				return []string{}, err1
			}
			if len(orig) > 0 {
				err1 = os.RemoveAll(orig)
				if err1 != nil {
					return []string{}, err1
				}
				err1 = os.Symlink(dst, orig)
				if err1 != nil {
					return []string{}, err1
				}
			}
			return []string{dst}, nil
		}
	}
	return []string{}, nil
}

func copy(f string, dst string, re []*regexp.Regexp, fn func(s, d, o string) ([]string, error)) ([]string, error) {
	fi, err := os.Stat(f)
	if err != nil {
		fmt.Println(f + " to copy does not exist, please check again")
		return []string{}, err
	}

	// file is a symlink, copy its original content
	if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
		fmt.Println(f + " is a symlink, copying the original file")
		link, err := dirutils.ReadSymlink(f)
		if err != nil {
			return []string{}, err
		}
		info, _ := os.Stat(link)
		fi = info
		f = link
	}

	orig := ""
	di, err := os.Stat(dst)
	// dst exists and is a symlink
	if !os.IsNotExist(err) && di.Mode()&os.ModeSymlink == os.ModeSymlink {
		fmt.Println(dst + " is a symlink, copy to its original file and symlink back")
		orig = dst
		link, err := dirutils.ReadSymlink(dst)
		if err != nil {
			return []string{}, err
		}
		info, _ := os.Stat(link)
		di = info
		dst = link
	}

	if fi.Mode().IsRegular() {
		files, err1 := fn(f, dst, orig)
		if err1 != nil {
			return []string{}, err1
		}
		return files, nil
	}
	if fi.Mode().IsDir() {
		if !di.Mode().IsDir() {
			return []string{}, fmt.Errorf("Source is a directory, destination should be directory too")
		}

		files, err1 := dirutils.Ls(f)
		if err1 != nil {
			return []string{}, err1
		}

		res := []string{}
		for _, v := range files {
			ok := false
			if len(re) > 0 {
				for _, r := range re {
					if r.MatchString(v) {
						ok = true
						break
					}
				}
			} else {
				ok = true
			}

			if !ok {
				continue
			}

			copyed, err1 := fn(v, dst, orig)
			if err1 != nil {
				return []string{}, err1
			}
			for _, c := range copyed {
				res = append(res, c)
			}
		}
		return res, nil
	}
	return []string{}, fmt.Errorf("Unknown FileMode %v of source", fi)
}

// Copy like Linux's cp command, copy a file/dirctory to another place.
func Copy(src, dst string) error {
	f, r := dirutils.ParseWildcard(src)
	_, err := copy(f, dst, r, cp)
	return err
}

//HasPrefixOrSuffix Check if str has prefix or suffix provided by the variadic slice
// the slice can be []string or [][]string, which means you can group prefixes and suffixes
// >0 means prefix, <0 means suffix, ==0 means no match.
func HasPrefixOrSuffix(str string, ends ...interface{}) int {
	if len(ends) == 0 {
		return 0
	}

	testK := reflect.ValueOf(ends[0]).Kind()

	if testK == reflect.String {
		// ends is a slice of string, just test prefix or suffix
		for _, v := range ends {
			s := reflect.ValueOf(v).String()
			if strings.HasPrefix(str, s) {
				return 1
			}
			if strings.HasSuffix(str, s) {
				return -1
			}
		}
		return 0
	}

	if testK == reflect.Array || testK == reflect.Slice {
		for _, end := range ends {
			v := reflect.ValueOf(end)
			if v.Index(0).Kind() != reflect.String {
				log.Fatal("You must provide a slice of string, or a slice of string slice to check prefix/suffix against the provided string.")
			}
			for i := 0; i < v.Len(); i++ {
				s := v.Index(i).String()
				if strings.HasPrefix(str, s) {
					return 1
				}
				if strings.HasSuffix(str, s) {
					return -1
				}
			}
		}
	}

	return 0
}
