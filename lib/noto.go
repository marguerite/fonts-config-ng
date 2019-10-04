package lib

import (
	"github.com/marguerite/util/slice"
	"reflect"
	"strings"
)

type NotoFPLs []NotoFPL

func (fpl NotoFPLs) GenLatinFPL() string {
	str := ""
	for _, font := range fpl {
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

type NotoFPL struct {
	Lang       string
	NameLang   string
	EditMethod string
	Sans       []string
	Serif      []string
	Monospace  []string
}

func NewNotoFPL(lang, method string) NotoFPL {
	return NotoFPL{lang, lang, method, []string{}, []string{}, []string{}}
}

func (fpl *NotoFPL) AppendByFontType(font string) {
	fv := reflect.ValueOf(fpl).Elem()
	generic := getGenericFamily(font)
	if generic == "sans-serif" {
		generic = "sans"
	}
	v := fv.FieldByName(strings.Title(generic))
	if v.IsValid() && v.CanSet() {
		v.Set(reflect.Append(v, reflect.ValueOf(font)))
	}
}

//GenNotoConfig generate fontconfig for Noto Fonts
func GenNotoConfig(fonts Collection, userMode bool) {
	defaultFamilies, fpl := genNotoConfig(fonts, userMode)
	defaultPos := GetConfigLocation("notoDefault", userMode)
	fplPos := GetConfigLocation("notoPrefer", userMode)
	overwriteOrRemoveFile(defaultPos, []byte(defaultFamilies), 0644)
	overwriteOrRemoveFile(fplPos, []byte(fpl), 0644)
}

func genNotoConfig(fonts Collection, userMode bool) (string, string) {
	fonts = fonts.FindByPath("Noto")
	fpl := NotoFPLs{}

	defaultFamilies := genConfigPreamble(userMode, "Default families for Noto Fonts installed on your system.")

	for _, font := range fonts {
		defaultFamilies += genDefaultFamilyNoto(font)
	}

	defaultFamilies += "</fontconfig>\n"

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
				f := NewNotoFPL(lang, "none")
				for _, name := range font.UnstyledName() {
					f.AppendByFontType(name)
				}
				fpl = append(fpl, f)
			}
		}
	}

	langFPL := genConfigPreamble(userMode, "Language specific family preference list for Noto Fonts.") +
		fpl.GenLatinFPL() +
		"</fontconfig>\n"

	return defaultFamilies, langFPL
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

//genDefaultFamilyNoto generate default family fontconfig block for Noto Fonts
func genDefaultFamilyNoto(font Font) string {
	str := ""
	for _, name := range font.Name {
		str += "\t<alias>\n\t\t<family>" + name + "</family>\n\t\t<default>\n\t\t\t<family>"
		str += getGenericFamily(name)
		str += "</family>\n\t\t</default>\n\t</alias>\n\n"
	}
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
