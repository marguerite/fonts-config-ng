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
	Lang      string
	NameLang  string
	CJK       bool
	Sans      FPL
	Serif     FPL
	Monospace FPL
}

func NewNotoLFPL(lang, method string) NotoLFPL {
	nameLang := lang
	cjk := false
	if fileutils.HasPrefixOrSuffix(lang, "zh-", "ja", "ko") != 0 {
		cjk = true
	}
	return NotoLFPL{lang, nameLang, cjk, method, FPL{}, FPL{}, FPL{}}
}

func (lfpl *NotoLFPL) AddFont(font string, c Collection) {
	generic := getGenericFamily(font)
	if generic == "sans-serif" {
		generic = "sans"
	}
	generic = strings.Title(generic)

	fv := reflect.ValueOf(lfpl).Elem()
	v := fv.FieldByName(generic)

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

type FPL struct {
	Prepend ModificationList
	Append  ModificationList
	Default CandidateList
}

func NewFPL(font, lang string, ppd, apd ModificationList, c Collection) FPL {
	defa := List{}
	defa.Add(font, lang)
	if fileutils.HasPrefixOrSuffix(lang, "zh-", "ja", "ko") != 0 {
		if variant := genAllVariantsAlternative(font, c); len(variant) > 0 {
			apd.Prepend(variant)
		}
	}
	return FPL{ppd, apd, defa}
}

func (f *FPL) Add(font string) {
	f.Default.Add(font, f.Lang)
}

func genAllVariantsAlternative(font string, c Collection) string {
	f := strings.Split(font, " ")
	name := strings.Join(f[:2], " ") + " CJK " + f[len(f)-1]
	if len(c.FindByName(name)) > 0 {
		return name
	}
	return ""
}

type ModificationList []string

//Installed leave the installed font in ModificationList only
func (m *ModificationList) Installed(c Collection) {
	for _, v := range *m {
		if len(c.FindByName(v)) == 0 {
			slice.Remove(m, v)
		}
	}
}

func (m *ModificationList) Prepend(font string) {
	if b, err := slice.Contains(*m, font); !b && err != nil {
		*m = append([]string{font}, *m...)
	}
}

func genCJKPrependML(generic, lang string, c Collection) ModificationList {
	m := ModificationList{}
	if generic == "Sans" || generic == "Serif" {
		m = append(m, "Noto "+generic)
	}
	ja := map[string][]string{"Sans": {"IPAPGothic", "IPAexGothic", "M+ 1c", "M+ 1p", "VL PGothic"},
		"Serif":     {"IPAPMincho", "IPAexMincho"},
		"Monospace": {"IPAGothic", "M+ 1m", "VL Gothic"}}
	if lang == "ja" {
		slice.Concat(&m, ja[generic])
	}
	return m.Installed(c)
}

func genCJKAppendML(generic, lang string, c Collection) ModificationList {
	m := ModificationList{}
	ko := map[string]string{"Sans": "NanumGothic", "Serif": "NanumMyeongjo", "Monospace": "NanumGothicCoding"}
	ja := map[string]string{"Sans": "IPAGothic", "Serif": "IPAMincho"}
	switch lang {
	case "zh-tw", "zh-hk", "zh-mo":
		m = append(m, "CMEXSong")
	case "ja":
		if _, ok := ja[generic]; ok {
			m = append(m, ja[generic])
		}
	case "ko":
		m = append(m, ko[generic])
	}
	return m.Installed(c)
}

//CandidateList Font Candidate List
type CandidateList []string

//Add Add or Prepend to List
func (l *CandidateList) Add(font, lang string) {
	m := map[string][]string{"JP": {"ja"}, "KR": {"ko"},
		"SC": {"zh-cn", "zh-sg"},
		"TC": {"zh-tw", "zh-hk", "zh-mo"}}

	// "Noto Sans JP" -> "JP"
	s, ok := m[font[len(font)-2:]]

	if ok {
		// "Noto Sans JP" and language is "ja"
		if b, err := slice.Contains(s, lang); b && err == nil {
			if b1, err1 := slice.Contains(*l, font); !b1 && err1 == nil {
				// Prepend
				*l = append([]string{font}, *l...)
			}
		} else {
			// Normal Add
			if b1, err1 := slice.Contains(*l, font); !b1 && err1 == nil {
				*l = append(*l, font)
			}
		}
	} else {
		// Latin Fonts, Normal Add
		if b, err := slice.Contains(*l, font); !b && err == nil {
			*l = append(*l, font)
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
