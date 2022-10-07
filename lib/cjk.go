package lib

import (
	"strings"

	ft "github.com/marguerite/fonts-config-ng/font"
	"github.com/marguerite/go-stdlib/slice"
)

//GenCJKConfig generate cjk specific fontconfig configuration like
// special matrix adjustment for "Noto Sans/Serif", dual-width Asian fonts and etc.
func GenCJKConfig(c ft.Collection, userMode bool) {
	conf := GetFcConfig("cjk", userMode)
	text := genFcPreamble(userMode, "")
	text += fixDualAsianFonts(c)
	text += genNotoCJK()
	text += FcSuffix
	overwriteOrRemoveFile(conf, []byte(text))
}

//isSpacingDual find spacing=dual/mono/charcell
func isSpacingDual(font ft.Font) int {
	if font.Spacing > 90 && !font.Outline {
		return 1
	}
	if font.Spacing == 90 {
		return 0
	}
	return -1
}

//isCJKFont find if a font supports CJK
func isCJKFont(font ft.Font) bool {
	supportedLangs := []string{"zh", "ja", "ko", "zh-cn", "zh-tw", "zh-hk", "zh-mo", "zh-sg"}
	if ok, err := slice.Contains(font.Lang, supportedLangs); ok && err == nil {
		return true
	}
	return false
}

//fixDualAsianFonts fix rendering of dual-width Asian fonts (spacing=dual)
func fixDualAsianFonts(c ft.Collection) string {
	comment := "<!-- The dual-width Asian fonts (spacing=dual) are not rendered correctly," +
		"apparently FreeType forces all widths to match.\n" +
		"Trying to disable the width forcing code by setting globaladvance=false alone doesn't help.\n" +
		"As a brute force workaround, also set spacing=proportional, i.e. handle them as proportional fonts. -->\n" +
		"<!-- There is a similar problem with dual width bitmap fonts which don't have spacing=dual but mono or charcell.-->\n\n"
	text := ""

	for _, font := range c {
		if isSpacingDual(font) >= 0 && isCJKFont(font) {
			text += genDualAisanConfig(font)
		}
	}

	if len(text) > 0 {
		return comment + text
	}
	return text
}

func ppd(generic, lang string) string {
	if lang != "ja" {
		return ""
	}
	m := map[string][]string{"Sans": []string{"IPAPGothic", "IPAexGothic", "M+ 1c", "M+ 1p", "VL PGothic"}, "Serif": []string{"IPAPMincho", "IPAexMincho"}, "monospace": []string{"IPAGothic", "M+ 1m", "VL Gothic"}}
	var str string
	for _, v := range m[generic] {
		str += "\t\t\t<string>" + v + "</string>\n"
	}
	return str
}

func apd(generic, lang string) string {
	switch lang {
	case "ja":
		m := map[string]string{"Sans": "IPAGothic", "Serif": "IPAMincho"}
		if val, ok := m[generic]; ok {
			return "\t\t\t<string>" + val + "</string>\n"
		}
		return ""
	case "ko":
		m := map[string]string{"Sans": "NanumGothic", "Serif": "NanumMyeongjo", "monospace": "NanumGothicCoding"}
		return "\t\t\t<string>" + m[generic] + "</string>\n"
	case "zh-tw", "zh-hk", "zh-mo":
		if generic == "Serif" {
			return "\t\t\t<string>CMEXSong</string>\n"
		}
		return ""
	default:
		return ""
	}
}

func genNotoCJK() string {
	order := map[string][]string{"zh-cn": []string{"SC", "HK", "TW", "JP", "KR"},
		"zh-tw": []string{"TC", "HK", "SC", "JP", "KR"},
		"zh-hk": []string{"HK", "TC", "SC", "JP", "KR"},
		"zh-mo": []string{"HK", "SC", "TC", "JP", "KR"},
		"zh-sg": []string{"SC", "HK", "TW", "JP", "KR"},
		"ja":    []string{"JP", "KR", "HK", "TW", "SC"},
		"ko":    []string{"KR", "JP", "HK", "TW", "SC"}}

	var str string

	for _, v := range []string{"sans-serif", "serif"} {
		for k, v1 := range order {
			str += "\t<match>\n\t\t<test name=\"family\">\n\t\t\t<string>" + v + "</string>\n\t\t</test>\n"
			v3 := v
			if v3 == "sans-serif" {
				v3 = v3[:4]
			}
			v3 = strings.Title(v3)
			str += "\t\t<test name=\"lang\">\n\t\t\t<string>" + k + "</string>\n\t\t</test>\n" +
				"\t\t<edit name=\"family\" mode=\"prepend\">\n" +
				ppd(v, k)
			if v == "sans-serif" {
				str += "\t\t\t<string>Noto " + v3 + "</string>\n"
			}
			for _, v2 := range v1 {
				str += "\t\t\t<string>Noto " + v3 + " " + v2 + "</string>\n"
			}
			if k == "zh-mo" {
				str += "\t\t\t<string>Noto " + v3 + " CJK HK</string>\n"
			} else {
				str += "\t\t\t<string>Noto " + v3 + " CJK " + v1[0] + "</string>\n"
			}
			str += apd(v, k)
			str += "\t\t</edit>\n\t</match>\n\n"
		}
	}

	for k, v := range order {
		str += "\t<match>\n\t\t<test name=\"family\">\n\t\t\t<string>monospace</string>\n\t\t</test>\n" +
			"\t\t<test name=\"lang\">\n\t\t\t<string>" + k + "</string>\n\t\t</test>\n" +
			"\t\t<edit name=\"family\" mode=\"prepend\">\n" +
			ppd("monospace", k) +
			"\t\t\t<string>Noto Sans CJK "
		if k == "zh-mo" {
			str += "HK"
		} else {
			str += v[0]
		}
		str += "</string>\n" + apd("monospace", k)
		str += "\t\t</edit>\n\t</match>\n\n"
	}

	return str
}
