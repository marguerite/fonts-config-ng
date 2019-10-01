package main

import (
	"github.com/marguerite/util/fileutils"
	"github.com/openSUSE/fonts-config/lib"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type fplByMethod struct {
	Prefer map[string][]string
	Append map[string][]string
}

func generateMatrix(font string, matrix []float64, langs []string) string {
	out := ""

	if len(matrix) != 4 {
		log.Fatalf("Invalid matrix: %v", matrix)
	}

	for _, lang := range langs {
		s := "\t<match target=\"font\">\n\t\t<test name=\"family\">\n\t\t\t<string>" + font + "</string>\n\t\t</test>\n"
		s += "\t\t<test name=\"namelang\">\n\t\t\t<string>" + lang + "</string>\n\t\t</test>\n"
		s += "\t\t<edit name=\"matrix\" mode=\"assign\">\n\t\t\t<times>\n\t\t\t\t<name>matrix</name>\n\t\t\t\t<matrix>\n"
		s += "\t\t\t\t\t<double>" + strconv.FormatFloat(matrix[0], 'f', -1, 64) + "</double>\n"
		s += "\t\t\t\t\t<double>" + strconv.FormatFloat(matrix[1], 'f', -1, 64) + "</double>\n"
		s += "\t\t\t\t\t<double>" + strconv.FormatFloat(matrix[2], 'f', -1, 64) + "</double>\n"
		s += "\t\t\t\t\t<double>" + strconv.FormatFloat(matrix[3], 'f', -1, 64) + "</double>\n"
		s += "\t\t\t\t</matrix>\n\t\t\t</times>\n\t\t</edit>\n\t</match>\n\n"
		out += s
	}
	return out
}

func generateWeight(font string, weights [][]int, langs []string) string {
	out := ""

	for _, w := range weights {
		if len(w) < 3 {
			log.Fatalf("invalid weight item: %v", w)
		}
	}

	for _, lang := range langs {
		for _, w := range weights {
			s := "\t<match target=\"font\">\n\t\t<test name=\"family\">\n\t\t\t<string>" + font + "</string>\n"
			s += "\t\t</test>\n\t\t<test name=\"namelang\">\n\t\t\t<string>" + lang + "</string>\n\t\t</test>\n"

			if w[0] != 0 {
				s += "\t\t<test name=\"weight\" compare=\"more_eq\">\n\t\t\t<int>" + strconv.FormatInt(int64(w[0]), 10) + "</int>\n\t\t</test>\n"
			}

			if w[1] != 0 {
				s += "\t\t<test name=\"weight\" compare=\"less_eq\">\n\t\t\t<int>" + strconv.FormatInt(int64(w[1]), 10) + "</int>\n\t\t</test>\n"
			}

			s += "\t\t<edit name=\"weight\" mode=\"assign\">\n\t\t\t<int>" + strconv.FormatInt(int64(w[2]), 10) + "</int>\n\t\t</edit>\n\t</match>\n\n"
			out += s
		}
	}
	return out
}

func generateWidth(font string, widths []int, langs []string) string {
	out := ""

	if len(widths) != 2 {
		log.Fatalf("invalid weight item: %v", widths)
	}

	for _, lang := range langs {
		s := "\t<match target=\"font\">\n\t\t<test name=\"family\">\n\t\t\t<string>" + font + "</string>\n\t\t</test>\n"
		s += "\t\t<test name=\"namelang\">\n\t\t\t<string>" + lang + "</string>\n\t\t</test>\n"
		s += "\t\t<test name=\"width\" compare=\"more_eq\">\n\t\t\t<int>" + strconv.FormatInt(int64(widths[0]), 10) + "</int>\n\t\t</test>\n"
		s += "\t\t<test name=\"width\" compare=\"less_eq\">\n\t\t\t<int>" + strconv.FormatInt(int64(widths[1]), 10) + "</int>\n\t\t</test>\n"
		s += "\t\t<edit name=\"width\" mode=\"assign\">\n\t\t\t<int>" + strconv.FormatInt(int64(widths[0]), 10) + "</int>\n\t\t</edit>\n\t</match>\n\n"
		out += s
	}
	return out
}

func generatePrefer(font string, family map[string][]string, langSpecific map[string]fplByMethod, langs []string) string {
	out := ""
	langMap := map[string]string{
		"zh-cn": "cn", "zh-sg": "cn",
		"zh-tw": "tw", "zh-hk": "tw", "zh-mo": "tw",
		"ja": "jp", "ko": "kr"}

	for _, lang := range langs {
		editLang := langMap[lang]
		s := "\t<match>\n\t\t<test name=\"family\">\n\t\t\t<string>" + font + "</string>\n\t\t</test>\n"
		s += "\t\t<test name=\"lang\">\n\t\t\t<string>" + lang + "</string>\n\t\t</test>\n"
		s += "\t\t<edit name=\"family\" mode=\"prepend\">\n"

		for m, n := range langSpecific {
			if m == editLang && len(n.Prefer) > 0 {
				for x, y := range n.Prefer {
					if x == font {
						for _, k := range y {
							s += "\t\t\t<string>" + k + "</string>\n"
						}
					}
				}
			}
		}

		if font == "sans-serif" {
			s += "\t\t\t<string>Noto Sans</string>\n"
		}

		if font == "serif" {
			s += "\t\t\t<string>Noto Serif</string>\n"
		}

		s += "\t\t\t<string>" + family[editLang][0] + "</string>\n"

		if font != "monospace" {
			for i, v := range family {
				if i != editLang {
					s += "\t\t\t<string>" + v[0] + "</string>\n"
				}
			}

			s += "\t\t\t<string>" + family[editLang][1] + "</string>\n"
		}

		for m, n := range langSpecific {
			if m == editLang && len(n.Append) > 0 {
				for x, y := range n.Append {
					if x == font {
						for _, k := range y {
							s += "\t\t\t<string>" + k + "</string>\n"
						}
					}
				}
			}
		}

		s += "\t\t</edit>\n\t</match>\n\n"
		out += s

	}
	return out
}

func buildSourceHanFontsList() []string {
	var fonts []string
	family := []string{" Sans", " Serif", " Sans HW"}
	variants := []string{" CN", " TW", " JP", " KR", "", " J", " K", " SC", " TC"}

	for _, f := range family {
		for _, v := range variants {
			font := "Source Han" + f + v
			fonts = append(fonts, font)
		}
	}
	return fonts
}

func remainFamily(family map[string][]string, lang string) []string {
	var remains []string
	for i, v := range family {
		if i != lang {
			remains = append(remains, v[0])
		}
	}
	return remains
}

func generateSourceHanAlias(fonts []string, sans, serif, mono map[string][]string) string {
	out := ""
	regionSuffix := []string{"CN", "TW", "JP", "KR"}
	otcSuffix := []string{"J", "K", "SC", "TC"}
	langMap := map[string]string{"CN": "cn", "SC": "cn", "TW": "tw", "TC": "tw",
		"JP": "jp", "J": "jp", "KR": "kr", "K": "kr"}

	for _, f := range fonts {
		fa := strings.Split(f, " ")
		if fileutils.HasPrefixOrSuffix(fa[len(fa)-1], []string{"Sans", "Serif", "HW"}) {
			fa = append(fa, "J")
		}
		lang := fa[len(fa)-1]
		variant := fa[len(fa)-2]
		editLang := langMap[lang]
		familyMap := map[string]string{"Sans": sans[editLang][0],
			"Serif": serif[editLang][0],
			"HW":    mono[editLang][0]}
		remainMap := map[string][]string{"Sans": remainFamily(sans, editLang),
			"Serif": remainFamily(serif, editLang),
			"HW":    make([]string, 0)}
		str := "\t<alias>\n\t\t<family>" + f + "</family>\n"

		if fileutils.HasPrefixOrSuffix(lang, regionSuffix) {
			str += "\t\t<accept>\n\t\t\t<family>" + familyMap[variant] + "</family>\n\t\t</accept>\n"
		}

		if fileutils.HasPrefixOrSuffix(lang, otcSuffix) {
			if variant == "HW" {
				str += "\t\t<accept>\n"
			} else {
				str += "\t\t<prefer>\n"
			}
			str += "\t\t\t<family>" + familyMap[variant] + "</family>\n"
			for _, r := range remainMap[variant] {
				str += "\t\t\t<family>" + r + "</family>\n"
			}
			if variant == "HW" {
				str += "\t\t</accept>\n"
			} else {
				str += "\t\t</prefer>\n"
			}
		}
		out += str + "\t</alias>\n\n"
	}
	return out
}

func buildNotoCJKAndSourceHanList(sans, serif, mono map[string][]string) []string {
	fonts := buildSourceHanFontsList()
	for _, l := range []map[string][]string{sans, serif, mono} {
		for _, v := range l {
			for _, k := range v {
				fonts = append(fonts, k)
			}
		}
	}
	return fonts
}

func generateHinting(fonts []string, hint string) string {
	out := ""
	for _, v := range fonts {
		s := "\t<match target=\"font\">\n\t\t<test name=\"family\">\n\t\t\t<string>" + v + "</string>\n\t\t</test>\n"
		s += "\t\t<edit name=\"hintstyle\" mode=\"assign\">\n\t\t\t<const>" + hint + "</const>\n\t\t</edit>\n\t</match>\n\n"
		out += s
	}
	return out
}

func main() {
	langs := []string{"zh-cn", "zh-sg", "zh-tw", "zh-hk", "zh-mo", "ja", "ko"}
	namelangs := []string{"zh-CN", "zh-SG", "zh-TW", "zh-HK", "zh-MO", "ja", "ko"}
	matrix := []float64{0.67, 0, 0, 0.67}
	weights := [][]int{{0, 40, 0}, {50, 99, 50}, {99, 179, 80}, {180, 0, 180}}
	widths := []int{63, 100}
	sans := map[string][]string{
		"cn": {"Noto Sans SC", "Noto Sans CJK SC"},
		"tw": {"Noto Sans TC", "Noto Sans CJK TC"},
		"jp": {"Noto Sans JP", "Noto Sans CJK JP"},
		"kr": {"Noto Sans KR", "Noto Sans CJK KR"},
	}
	serif := map[string][]string{
		"cn": {"Noto Serif SC", "Noto Serif CJK SC"},
		"tw": {"Noto Serif TC", "Noto Serif CJK TC"},
		"jp": {"Noto Serif JP", "Noto Serif CJK JP"},
		"kr": {"Noto Serif KR", "Noto Serif CJK KR"},
	}
	mono := map[string][]string{
		"cn": {"Noto Sans Mono CJK SC"},
		"tw": {"Noto Sans Mono CJK TC"},
		"jp": {"Noto Sans Mono CJK JP"},
		"kr": {"Noto Sans Mono CJK KR"},
	}
	langSpecific := map[string]fplByMethod{
		"tw": {Append: map[string][]string{"serif": {"CMEXSong"}}},
		"kr": {Append: map[string][]string{"sans-serif": {"NanumGothic"}, "serif": {"NanumMyeongjo"}, "monospace": {"NanumGothicCoding"}}},
		"jp": {Prefer: map[string][]string{"sans-serif": {"IPAPGothic", "IPAexGothic", "M+ 1c", "M+ 1p", "VL PGothic"}, "serif": {"IPAPMincho", "IPAexMincho"}, "monospace": {"IPAGothic", "M+ 1m", "VL Gothic"}}, Append: map[string][]string{"sans-serif": {"IPAGothic"}, "serif": {"IPAMincho"}}},
	}

	list := buildNotoCJKAndSourceHanList(sans, serif, mono)

	cjk := "<?xml version=\"1.0\"?>\n<!DOCTYPE fontconfig SYSTEM \"fonts.dtd\">\n<fontconfig>\n<!-- Generated by /usr/lib/fonts-config/generate_cjk_config -->\n"
	defaultFamilyCJK := cjk
	ttGroupCJK := cjk

	cjk += "<!--- Currently we use Region Specific Subset OpenType/CFF (Subset OTF)\n"
	cjk += "\tflavor of Google's Noto Sans/Serif CJK fonts, but previously we\n"
	cjk += "\tused Super OpenType/CFF Collection (Super OTC), and other distributions\n"
	cjk += "\tmay use Language specific OpenType/CFF (OTF) flavor. So\n"
	cjk += "\tNoto Sans/Serif CJK SC/TC/JP/KR are also common font names.-->\n\n"
	cjk += "<!--- fontconfig doesn't support the OpenType locl GSUB feature,\n"
	cjk += "\tso only the default glyph variant (JP) can be used in the\n"
	cjk += "\tSuper OTC and OTF flavors. We gave them very low priority\n"
	cjk += "\ton openSUSE even if they were installed manually.-->\n\n"
	cjk += "<!--- 1. Prepend 'Noto Sans/Serif' before CJK because the Latin part is from\n"
	cjk += "\t\t'Adobe Source Sans/Serif Pro' which is 2/3 smaller than Noto.-->\n"

	cjk += generateMatrix("Noto Sans", matrix, namelangs)
	cjk += generateMatrix("Noto Serif", matrix, namelangs)
	cjk += generateWeight("Noto Sans", weights, namelangs)
	cjk += generateWeight("Noto Serif", weights, namelangs)
	cjk += generateWidth("Noto Sans", widths, namelangs)
	cjk += generateWidth("Noto Serif", widths, namelangs)
	cjk += generatePrefer("sans-serif", sans, langSpecific, langs)
	cjk += generatePrefer("serif", serif, langSpecific, langs)

	cjk += "<!--- 2. Don't prepend for Mono because its Latin part 'Adobe Source Code Pro'\n"
	cjk += "\t\tis openSUSE's choice for monospace font.\n"
	cjk += "\t3. 'Noto Sans Mono CJK XX' is real font in openSUSE.-->\n"
	cjk += generatePrefer("monospace", mono, langSpecific, langs)

	cjk += "<!--- 4. Alias 'Adobe Source Sans/Serif/HW' because they are the same as Noto.\n"
	cjk += "\t\tIf installed manually they can still be used.-->\n"
	cjk += generateSourceHanAlias(buildSourceHanFontsList(), sans, serif, mono)

	cjk += generateHinting(list, "hintfull")

	for _, f := range list {
		defaultFamilyCJK += lib.GenerateDefaultFamily(f)
		ttGroupCJK += lib.GenerateFontTypeByHinting(lib.Font{[]string{f}, []string{}, true})
	}

	cjk += "</fontconfig>"
	defaultFamilyCJK += "</fontconfig>"
	ttGroupCJK += "</fontconfig>"

	err := ioutil.WriteFile("59-family-prefer-lang-specific-cjk.conf", []byte(cjk), 0644)
	lib.ErrChk(err)
	err = ioutil.WriteFile("10-group-tt-hinted-cjk.conf", []byte(ttGroupCJK), 0644)
	lib.ErrChk(err)
	err = ioutil.WriteFile("49-family-default-cjk.conf", []byte(defaultFamilyCJK), 0644)
	lib.ErrChk(err)
}
