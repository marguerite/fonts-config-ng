package lib

import (
	"github.com/marguerite/util/slice"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// langContainsCJK if lang supports CJK and what CJK it supports
func langContainsCJK(lang string) []string {
	cjk := []string{"ja", "ko", "zh-cn", "zh-sg", "zh-tw", "zh-mo", "zh-hk", "zh"}
	out := []string{}
	for _, i := range strings.Split(lang, "|") {
		if b, _ := slice.Contains(cjk, i); b {
			out = append(out, i)
		}
	}
	return out
}

// dualSpacing find spacing=dual/mono/charcell
func dualSpacing(spacing, outline string) int {
	s, _ := strconv.Atoi(spacing)
	o, _ := strconv.ParseBool(outline)
	if s > 90 && !o {
		return 1
	}
	if s == 90 {
		return 0
	}
	return -1
}

func fixDualSpacing(userMode bool) string {
	fonts := ReadFontFiles()
	comment := "<!-- The dual-width Asian fonts (spacing=dual) are not rendered correctly," +
		"apparently FreeType forces all widths to match.\n" +
		"Trying to disable the width forcing code by setting globaladvance=false alone doesn't help.\n" +
		"As a brute force workaround, also set spacing=proportional, i.e. handle them as proportional fonts. -->\n" +
		"<!-- There is a similar problem with dual width bitmap fonts which don't have spacing=dual but mono or charcell.-->\n"
	text := ""

	re := regexp.MustCompile(`(?s)Pattern.*?\n\n`)
	re1 := regexp.MustCompile(`(?s)family: (.*?)\n.*?spacing: (\d+).*?outline: (\w+).*?lang: (.*?)\n`)
	replacer := strings.NewReplacer("(s)", "", "(w)", "")
	wg := sync.WaitGroup{}
	wg.Add(len(fonts))
	mux := sync.Mutex{}

	for _, v := range fonts {
		go func(font string, t *string) {
			defer wg.Done()
			str := ""
			out, _ := exec.Command("fc-scan", font).Output()
			for _, r := range re.FindAllStringSubmatch(string(out), -1) {
				m := re1.FindStringSubmatch(r[0])
				if len(m) == 0 {
					continue
				}
				spacing := dualSpacing(m[2], m[3])
				if spacing >= 0 && len(langContainsCJK(replacer.Replace(m[4]))) > 0 {
					family := replacer.Replace(m[1])
					families := []string{}
					// "\"Dorid Sans Japanese\"" and "\"DotumChe\" \"GulimChe\""
					for _, i := range strings.Split(family, "\"") {
						// no font name is just 1 length but whitespace is
						if len(i) > 1 {
							families = append(families, i)
						}
					}
					for _, j := range families {
						str += "\t<match target=\"font\">\n\t\t<test name=\"family\" compare=\"contains\">\n"
						str += "\t\t\t<string>"
						str += j
						str += "</string>\n\t\t</test>\n"
						str += "\t\t<edit name=\"spacing\" mode=\"append\">\n\t\t\t<const>proportional</const>\n\t\t</edit>\n"
						str += "\t\t<edit name=\"globaladvance\" mode=\"append\">\n\t\t\t<bool>false</bool>\n\t\t</edit>\n\t</match>\n"
					}
				}
			}
			mux.Lock()
			*t += str
			mux.Unlock()
		}(v, &text)
	}

	wg.Wait()

	if len(text) == 0 {
		return ""
	}
	return genConfigPreamble(userMode, comment) + text + "</fontconfig>\n"
}

// FixDualSpacing fix dual-width Asian fonts
func FixDualSpacing(userMode bool) {
	text := fixDualSpacing(userMode)
	dualConfig := GenConfigLocation("dual", userMode)
	err := persist(dualConfig, []byte(text), 0644)
	if err != nil {
		log.Fatalf("Can not write %s: %s\n", dualConfig, err.Error())
	}
}
