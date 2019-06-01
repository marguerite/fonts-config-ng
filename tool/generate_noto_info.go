package main

import (
	"flag"
	"fmt"
	"github.com/marguerite/util/dirutils"
	"github.com/marguerite/util/fileutils"
	"github.com/openSUSE/fonts-config/lib"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// unpackRPM unpack RPM to the dst directory
func unpackRPM(rpm, dst string) {
	c1 := exec.Command("/usr/bin/rpm2cpio", rpm)
	c2 := exec.Command("/usr/bin/cpio", "-idm", "-D", dst)
	reader, writer := io.Pipe()
	c1.Stdout = writer
	c2.Stdin = reader
	c1.Start()
	c2.Start()
	c1.Wait()
	writer.Close()
	c2.Wait()
	reader.Close()
}

func main() {
	var pkgDir string
	flag.StringVar(&pkgDir, "pkgdir", filepath.Join(os.Getenv("HOME"), "binaries"), "the pkgdir contains rpm packages for fonts.")
	flag.Parse()

	collection := lib.Collection{}
	notoFonts := []string{}
	re := regexp.MustCompile(`([^\/]+)-fonts.*$`)
	wg := &sync.WaitGroup{}
	mux := &sync.Mutex{}
	files, _ := dirutils.Ls(pkgDir, "file")

	for _, v := range files {
		if strings.HasPrefix(filepath.Base(v), "noto-") {
			notoFonts = append(notoFonts, v)
		}
	}

	for _, v := range notoFonts {
		wg.Add(1)
		go func(rpm string) {
			rpmFile := filepath.Base(rpm)
			privateDir := filepath.Join(pkgDir, re.FindStringSubmatch(rpm)[1])
			dst := filepath.Join(privateDir, rpmFile)
			err := os.MkdirAll(privateDir, 0755)
			if err != nil {
				fmt.Printf("can't create dir for %s, %s skipped.\n", privateDir, rpm)
				return
			}

			err = fileutils.Copy(rpm, dst)
			if err != nil {
				fmt.Printf("can't copy %s to %s, skipped.\n", rpm, dst)
			}

			unpackRPM(dst, privateDir)

			fontFiles, _ := filepath.Glob(filepath.Join(privateDir, "usr/share/fonts/truetype/*"))

			for _, ttf := range fontFiles {
				n := lib.ParseFontInfoFromFile(ttf)
				mux.Lock()
				collection = append(collection, n)
				mux.Unlock()
			}

			_ = os.RemoveAll(privateDir)
			wg.Done()
		}(v)
	}

	wg.Wait()

	js, err := collection.Encode()
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile("noto.json", js, 0644)
	if err != nil {
		panic(err)
	}
}
