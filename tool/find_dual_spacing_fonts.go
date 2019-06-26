package main

import (
	"fmt"
	"github.com/marguerite/util/slice"
	"github.com/openSUSE/fonts-config/lib"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// isCJKLang if lang supports CJK and what CJK it supports
func langCJK(lang string) []string {
	cjk := []string{"ja", "ko", "zh-cn", "zh-sg", "zh-tw", "zh-mo", "zh-hk", "zh"}
	out := []string{}
	for _, i := range strings.Split(lang, "|") {
		if b, _ := slice.Contains(cjk, i); b {
			out = append(out, i)
		}
	}
	return out
}

func fontSpacing(spacing, outline string) int {
	s, _ := strconv.Atoi(spacing)
	o, _ := strconv.ParseBool(outline)
	if s > 0 {
		if s > 90 && !o {
			return 1
		}
		return 0
	}
	return -1
}

func main() {
	localFonts := lib.ReadFontFilesFromDir(filepath.Join(lib.GetEnv("HOME"), ".fonts"), false)
	fonts := lib.ReadFontFilesFromDir("/usr/share/fonts/truetype", false)
	slice.Concat(&fonts, localFonts)

	re := regexp.MustCompile(`(?s)Pattern.*?family: (.*?\n).*?spacing: (\d+).*?outline: (\w+).*?lang: (.*?)\n.*?\n\n`)

	for _, v := range fonts {
		out, _ := exec.Command("fc-scan", v).Output()
		for _, r := range re.FindAllStringSubmatch(string(out), -1) {
			fmt.Println(r)
			spacing := fontSpacing(r[2], r[3])
			if spacing >= 0 && len(langCJK(r[4])) > 0 {
				replacer := strings.NewReplacer("\"", "", "(s)", "", "(w)", "")
				family := replacer.Replace(r[1])
				fmt.Println(family)
			}
		}
	}
}
