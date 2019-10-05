package lib

import (
	"fmt"
	"github.com/marguerite/util/fileutils"
	"github.com/marguerite/util/slice"
	"reflect"
	"strings"
)

type NotoLFPLs []NotoLFPL

func (lfpl *NotoLFPLs) AppendFont(lang, font string) {
	found := false
	for i, v := range *lfpl {
		if v.Lang == lang {
			found = true
			(*lfpl)[i].AppendFont(font)
		}
	}
	if !found {
		v := NewNotoLFPL(lang, "none")
		v.AppendFont(font)
		*lfpl = append(*lfpl, v)
	}
}

//FindByLang Find lang item by lang in lfpl.
func (lfpl NotoLFPLs) FindByLang(lang []string) NotoLFPLs {
	n := NotoLFPLs{}
	for _, v := range lfpl {
		if b, err := slice.Contains(lang, v.Lang); b && err == nil {
			n = append(n, v)
		}
	}
	return n
}

func (lfpl NotoLFPLs) GenLFPL() string {
	str := ""
	for _, v := range lfpl {
		if v.CJK {

		} else {
			str += genLatinFPL(v)
		}
	}
	return str
}

type NotoLFPL struct {
	Lang       string
	NameLang   string
	CJK        bool
	EditMethod string
	Sans       []string
	Serif      []string
	Monospace  []string
}

func NewNotoLFPL(lang, method string) NotoLFPL {
	nameLang := lang
	cjk := false
	if fileutils.HasPrefixOrSuffix(lang, "zh-", "ja", "ko") != 0 {
		cjk = true
	}
	return NotoLFPL{lang, nameLang, cjk, method, []string{}, []string{}, []string{}}
}

func (lfpl *NotoLFPL) AppendFont(font string) {
	fv := reflect.ValueOf(lfpl).Elem()
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
	lfpl := genNotoConfig(fonts, userMode)
	faPos := GetConfigLocation("notoDefault", userMode)
	lfplPos := GetConfigLocation("notoPrefer", userMode)
	overwriteOrRemoveFile(faPos, []byte(family), 0644)
	overwriteOrRemoveFile(lfplPos, []byte(lfpl), 0644)
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
	lfpl := NotoLFPLs{}

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
					lfpl.AppendFont(lang, name)
				}
			}
		}
	}

	fmt.Println(lfpl)

	str := genConfigPreamble(userMode, "<!--Language specific family preference list for Noto Fonts.-->") +
		lfpl.GenLFPL() +
		"</fontconfig>\n"

	return str
}

// genFPLForLang generate family preference list of fonts for a generic font name
// and a specific language
func genLatinFPL(lfpl NotoLFPL) string {
	str := ""
	for _, generic := range []string{"sans-serif", "serif", "monospace"} {
		mark := strings.Title(generic)
		if mark == "Sans-Serif" {
			mark = "Sans"
		}
		v := reflect.ValueOf(lfpl).FieldByName(mark)
		if v.Len() > 0 {
			str += "\t<match>\n\t\t<test name=\"family\">\n\t\t\t<string>" + generic + "</string>\n\t\t</test>\n"
			str += "\t\t<test name=\"lang\">\n\t\t\t<string>" + lfpl.Lang + "</string>\n\t\t</test>\n"
			str += "\t\t<edit name=\"family\" mode=\"prepend\">\n"
			for i := 0; i < v.Len(); i++ {
				str += "\t\t\t<string>" + v.Index(i).String() + "</string>\n"
			}
			str += "\t\t</edit>\n\t</match>\n\n"
		}
	}

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
