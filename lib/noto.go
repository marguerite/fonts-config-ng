package lib

import (
	"strings"

	ft "github.com/marguerite/fonts-config-ng/font"
	"github.com/marguerite/go-stdlib/slice"
)

// GenNotoConfig generate fontconfig for Noto Fonts
func GenNotoConfig(c ft.Collection, userMode bool) {
	c = c.FindByName("Noto")
	family := genNotoDefaultFamily(c, userMode)
	fpl := genNotoConfig(c, userMode)
	faPos := GetFcConfig("notoDefault", userMode)
	fplPos := GetFcConfig("notoPrefer", userMode)
	overwriteOrRemoveFile(faPos, []byte(family))
	overwriteOrRemoveFile(fplPos, []byte(fpl))
}

func genNotoDefaultFamily(c ft.Collection, userMode bool) string {
	str := genFcPreamble(userMode, "<!-- Default families for Noto Fonts installed on your system.-->")
	// font names across different font.Name may be equal.
	m := make(map[string]struct{})

	for _, font := range c {
		for _, name := range font.Name {
			if _, ok := m[name]; !ok {
				m[name] = struct{}{}
				str += genDefaultFamily(name)
			}
		}
	}

	str += FcSuffix

	return str
}

func genNotoConfig(c ft.Collection, userMode bool) string {
	nonLangFonts := []string{"Noto Sans", "Noto Sans Display",
		"Noto Sans Mono", "Noto Sans Symbols", "Noto Sans Symbols2",
		"Noto Serif", "Noto Serif Display",
		"Noto Mono", "Noto Emoji", "Noto Color Emoji"}

	var str string

	for _, v := range []string{"sans-serif", "serif", "monospace"} {
		m := make(map[string][]string)
		for _, font := range c {
			if b, err := slice.Contains(font.Name, nonLangFonts); !b && err == nil {
				if getGenericFamily(font.Name[0]) != v {
					continue
				}
				for _, lang := range font.Lang {
					if strings.HasPrefix(lang, "zh") || lang == "ja" || lang == "ko" {
						continue
					}
					val, ok := m[lang]
					if ok {
						if b1, err1 := slice.Contains(val, font.Name[0]); !b1 && err1 == nil {
							m[lang] = append(val, font.Name[0])
						}
					} else {
						m[lang] = []string{font.Name[0]}
					}
				}
			}
		}

		for k, v1 := range m {
			str += "\t<match>\n\t\t<test name=\"family\">\n\t\t\t<string>" + v + "</string>\n\t\t</test>\n" +
				"\t\t<test name=\"lang\">\n\t\t\t<string>" + k + "</string>\n\t\t</test>\n" +
				"\t\t<edit name=\"family\" mode=\"prepend\">\n"
			for _, v2 := range v1 {
				str += "\t\t\t<string>" + v2 + "</string>\n"
			}
			str += "\t\t</edit>\n\t</match>\n\n"
		}
	}

	return genFcPreamble(userMode, "<!-- Language specific family preference list for Noto Fonts installed on your system.-->") +
		str +
		FcSuffix
}

// genDefaultFamily generate default family fontconfig block for font name
func genDefaultFamily(name string) string {
	str := "\t<alias>\n\t\t<family>" + name + "</family>\n\t\t<default>\n\t\t\t<family>"
	str += getGenericFamily(name)
	str += "</family>\n\t\t</default>\n\t</alias>\n\n"
	return str
}

// getGenericFamily get generic name through font name
func getGenericFamily(name string) string {
	if strings.Contains(name, " Symbols") {
		return "symbol"
	}
	if strings.Contains(name, " Mono") || strings.Contains(name, " HW") {
		return "monospace"
	}
	if strings.HasSuffix(name, "Emoji") {
		return "emoji"
	}
	if strings.Contains(name, " Serif") {
		return "serif"
	}
	return "sans-serif"
}
