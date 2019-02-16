package dirutils

import (
	"fmt"
	"os"
	"path/filepath"
)

// Debug a trigger for debug output
const Debug int = 256

/*NonExistTargetError is used to indicate the target a symlink points
  to actually does not exist on the filesystem.
*/
type NonExistTargetError struct {
	Desc string
	Err  error
}

func (e NonExistTargetError) Error() string {
	return e.Desc
}

/*ReadSymlink follows the path of the symlink recursively and finds out
  the target it finally points to.
*/
func ReadSymlink(path string) (string, error) {
	link, err := os.Readlink(path)
	if err != nil {
		return path, err
	}
	if !filepath.IsAbs(link) {
		link = filepath.Join(filepath.Dir(path), link)
	}
	info, err := os.Stat(link)
	if err != nil {
		return link, NonExistTargetError{path + " points to an non-existent target " + link, err}
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return ReadSymlink(link)
	}
	return link, nil
}

/*Ls accepts a directory and the kind of file beneath to be listed,
  returns the list of file and the possible error.

	Kind supports: dir, symlink, defaults to file.
*/
func Ls(d, kind string) ([]string, error) {
	var files []string
	e := filepath.Walk(string(d), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				fmt.Printf("WARNING: no permission to visit %s, skipped.\n", path)
				return nil
			}
			return err
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
					if _, ok := err.(NonExistTargetError); ok {
						// the symlink points to an non-existent target, ignore
						fmt.Printf("WARNING: %s points to an non-existent target %s.\n", path, link)
						return nil
					}
					return err
				}
				f, err := os.Stat(link)
				if err != nil {
					return err
				}
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
					if _, ok := err.(NonExistTargetError); ok {
						// the symlink points to an non-existent target, ignore
						fmt.Printf("WARNING: %s points to an non-existent target %s.\n", path, link)
						return nil
					}
					return err
				}
				f, err := os.Stat(link)
				if err != nil {
					return err
				}
				if f.Mode().IsRegular() {
					files = append(files, link)
				}
			}
		}
		return nil
	})
	return files, e
}

// MakePath create directories for path
func MkdirP(path string, verbosity int) error {
	p := filepath.Dir(path)
	if verbosity >= Debug {
		fmt.Printf("--- creating directory: %s\n", p)
	}
	if _, err := os.Stat(p); os.IsNotExist(err) {
		err := os.MkdirAll(p, os.ModeDir)
		if err != nil {
			if verbosity >= Debug {
				fmt.Println("can not create.")
			}
			return err
		}
		if verbosity >= Debug {
			fmt.Println("created.")
		}
	} else {
		if verbosity >= Debug {
			fmt.Println("exists.")
		}
		return nil
	}
	return nil
}
