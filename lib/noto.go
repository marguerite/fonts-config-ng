package lib

import (
	"fmt"
	"github.com/marguerite/util/slice"
	"reflect"
	"strings"
)

type NotoLPLs []NotoLPL

func (lpl *NotoLPLs) AppendFont(lang, font string) {
	found := false
	for i, v := range *lpl {
		if v.Lang == lang {
			found = true
			(*lpl)[i].AppendFont(font)
		}
	}
	if !found {
		v := NewNotoLPL(lang, "none")
		v.AppendFont(font)
		*lpl = append(*lpl, v)
	}
}

func (lpl NotoLPLs) GenLPL() string {
	str := ""
	for _, font := range lpl {
		if isCJK(font.Lang) {
			continue
		}
		if len(font.Sans) > 0 {
			str += genFPLForLang("sans-serif", font.Lang, font.Sans)
		}
		if len(font.Serif) > 0 {
			str += genFPLForLang("serif", font.Lang, font.Serif)
		}
		if len(font.Monospace) > 0 {
			str += genFPLForLang("monospace", font.Lang, font.Monospace)
		}
	}
	return str
}

//NotoLPL Noto's Language Preference List
type NotoLPL struct {
	Lang       string
	NameLang   string
	EditMethod string
	Sans       []string
	Serif      []string
	Monospace  []string
}

func NewNotoLPL(lang, method string) NotoLPL {
	return NotoLPL{lang, lang, method, []string{}, []string{}, []string{}}
}

func (lpl *NotoLPL) AppendFont(font string) {
	fv := reflect.ValueOf(lpl).Elem()
	generic := getGenericFamily(font)
	if generic == "sans-serif" {
		generic = "sans"
	}
	v := fv.FieldByName(strings.Title(generic))
	if v.IsValid() {
		if v.Len() == 0 {
			v.Set(reflect.Append(v, reflect.ValueOf(font)))
		} else {
			if b, err := slice.Contains(v.Interface(), font); !b && err == nil {
				v.Set(reflect.Append(v, reflect.ValueOf(font)))
			}
		}
	}
}

//GenNotoConfig generate fontconfig for Noto Fonts
func GenNotoConfig(fonts Collection, userMode bool) {
	fonts = fonts.FindByPath("Noto")
	family := genNotoDefaultFamily(fonts, userMode)
	lpl := genNotoConfig(fonts, userMode)
	faPos := GetConfigLocation("notoDefault", userMode)
	lplPos := GetConfigLocation("notoPrefer", userMode)
	overwriteOrRemoveFile(faPos, []byte(family), 0644)
	overwriteOrRemoveFile(lplPos, []byte(lpl), 0644)
}

func genNotoDefaultFamily(fonts Collection, userMode bool) string {
	str := genConfigPreamble(userMode, "<!--Default families for Noto Fonts installed on your system.-->")
	// font names across different font.Name may be equal.
	m := make(map[string]struct{})

	for _, font := range fonts {
		for _, name := range font.Name {
			if _, ok := m[name]; !ok {
				m[name] = struct{}{}
				str += genDefaultFamily(name)
			}
		}
	}

	str += "</fontconfig>\n"

	return str
}

func genNotoConfig(fonts Collection, userMode bool) string {
	lpl := NotoLPLs{}

	nonLangFonts := []string{"Noto Sans", "Noto Sans Disp", "Noto Sans Display",
		"Noto Sans Mono", "Noto Sans Symbols", "Noto Sans Symbols2",
		"Noto Serif", "Noto Serif Disp", "Noto Serif Display",
		"Noto Mono"}

	for _, font := range fonts {
		if b, err := slice.Contains(font.Name, nonLangFonts); !b && err == nil && len(font.Lang) > 0 {
			for _, lang := range font.Lang {
				if lang == "und-zsye" {
					continue
				}

				for _, name := range font.UnstyledName() {
					lpl.AppendFont(lang, name)
				}
			}
		}
	}

	str := genConfigPreamble(userMode, "<!--Language specific family preference list for Noto Fonts.-->") +
		lpl.GenLPL() +
		"</fontconfig>\n"

	return str
}

// genFPLForLang generate family preference list of fonts for a generic font name
// and a specific language
func genFPLForLang(generic, lang string, fonts []string) string {
	str := "\t<match>\n\t\t<test name=\"family\">\n\t\t\t<string>" + generic + "</string>\n\t\t</test>\n"
	str += "\t\t<test name=\"lang\">\n\t\t\t<string>" + lang + "</string>\n\t\t</test>\n"
	str += "\t\t<edit name=\"family\" mode=\"prepend\">\n"
	for _, f := range fonts {
		str += "\t\t\t<string>" + f + "</string>\n"
	}
	str += "\t\t</edit>\n\t</match>\n\n"
	return str
}

//genDefaultFamily generate default family fontconfig block for font name
func genDefaultFamily(name string) string {
	str := "\t<alias>\n\t\t<family>" + name + "</family>\n\t\t<default>\n\t\t\t<family>"
	str += getGenericFamily(name)
	str += "</family>\n\t\t</default>\n\t</alias>\n\n"
	return str
}

//getGenericFamily get generic name through font name
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

func isCJK(lang string) bool {
	if strings.HasPrefix(lang, "zh-") || lang == "ja" || lang == "ko" {
		return true
	}
	return false
}
