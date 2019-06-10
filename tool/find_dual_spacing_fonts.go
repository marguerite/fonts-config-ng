package main

import (
	"fmt"
	"github.com/marguerite/util/slice"
	"github.com/openSUSE/fonts-config/lib"
	"os/exec"
	"path/filepath"
	"regexp"
)

func main() {
	localFonts := lib.ReadFontFilesFromDir(filepath.Join(lib.GetEnv("HOME"), ".fonts"), false)
	fonts := lib.ReadFontFilesFromDir("/usr/share/fonts/truetype", false)
	slice.Concat(&fonts, localFonts)

	re := regexp.MustCompile(`(?m)^Pattern.*?^\n`)

	for _, v := range fonts {
		out, _ := exec.Command("fc-scan", v).Output()
		for _, r := range re.FindAllStringSubmatch(string(out), -1) {
			fmt.Println(r)
		}
	}
}
