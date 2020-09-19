package lib

import (
	"reflect"
	"strings"

	"github.com/marguerite/util/slice"
)

type LFPLs []LFPL

func (lfpl *LFPLs) AddFont(lang, font, generic, list string) {
	found := false
	for i, v := range *lfpl {
		if v.Lang == lang {
			found = true
			(*lfpl)[i].AddFont(font, generic, list)
		}
	}
	if !found {
		v := NewLFPL(lang)
		v.AddFont(font, generic, list)
		*lfpl = append(*lfpl, v)
	}
}

//GenLFPLsConfig turn Language grouped Family Preference List to Fontconfig Configuration
func (lfpl LFPLs) GenLFPLsConfig() string {
	config := ""
	for _, v := range lfpl {
		// we need a place to insert CJK comments for once.
		if v.Lang == "ja" {
			config += "<!--- Currently we use Region Specific Subset OpenType/CFF (Subset OTF)\n" +
				"\tflavor of Google's Noto Sans/Serif CJK fonts, but previously we\n" +
				"\tused Super OpenType/CFF Collection (Super OTC), and other distributions\n" +
				"\tmay use Language specific OpenType/CFF (OTF) flavor. So\n" +
				"\tNoto Sans/Serif CJK SC/TC/JP/KR are also common font names.-->\n\n" +
				"<!--- fontconfig doesn't support the OpenType locl GSUB feature,\n" +
				"\tso only the default glyph variant (JP) can be used in the\n" +
				"\tSuper OTC and OTF flavors. We gave them very low priority\n" +
				"\ton openSUSE even if they were installed manually.-->\n\n" +
				"<!--- 1. Prepend 'Noto Sans/Serif' before CJK because the Latin part is from\n" +
				"\t'Adobe Source Sans/Serif Pro'.-->\n" +
				"<!--- 2. Don't prepend for Mono because its Latin part 'Adobe Source Code Pro'\n" +
				"\tis openSUSE's choice for monospace font.\n" +
				"<!--- 3. 'Noto Sans Mono CJK XX' is real font in openSUSE.-->\n\n"
		}
		config += notoGenConfigForSpecificGenericFontAndLang(v)
	}
	return config
}

//LFPL Language grouped family preference list
type LFPL struct {
	Lang      string
	Sans      PAD
	Serif     PAD
	Monospace PAD
}

//NewLFPL initialie a new language grouped family preference list
func NewLFPL(lang string) LFPL {
	return LFPL{lang, PAD{}, PAD{}, PAD{}}
}

func (lfpl *LFPL) AddFont(font, generic, list string) {
	fv := reflect.ValueOf(lfpl).Elem()
	v := fv.FieldByName(generic)

	if v.IsValid() {
		if v.NumField() == 0 {
			v.Set(reflect.ValueOf(NewPAD(font, lfpl.Lang)))
		} else {
			v1 := v.FieldByName(list)
			m := map[string][]string{"JP": {"ja"}, "KR": {"ko"},
				"SC": {"zh-cn", "zh-sg"},
				"TC": {"zh-tw", "zh-hk", "zh-mo"}}

			// "Noto Sans JP" -> "JP"
			s, ok := m[font[len(font)-2:]]

			if ok {
				// "Noto Sans JP" and language is "ja"
				if b, err := slice.Contains(s, lfpl.Lang); b && err == nil {
					if b1, err1 := slice.Contains(v1.Interface(), font); !b1 && err1 == nil {
						// Prepend
						s1 := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(font)), v1.Len()+1, v1.Cap()+1)
						s1.Index(0).Set(reflect.ValueOf(font))
						for i := 0; i < v1.Len(); i++ {
							s1.Index(i + 1).Set(v1.Index(i))
						}
						v1.Set(s1)
					}
				} else {
					// Normal Add
					if b1, err1 := slice.Contains(v1.Interface(), font); !b1 && err1 == nil {
						v1.Set(reflect.Append(v1, reflect.ValueOf(font)))
					}
				}
			} else {
				// Latin Fonts, Normal Add
				if b, err := slice.Contains(v1.Interface(), font); !b && err == nil {
					v1.Set(reflect.Append(v1, reflect.ValueOf(font)))
				}
			}
		}
	}
}

//PAD (P)repend/(A)ppend/(D)efault Family Preference List
type PAD struct {
	Prepend CandidateList
	Append  CandidateList
	Default CandidateList
}

//NewPAD initialize a new PAD
func NewPAD(font, lang string) PAD {
	l := CandidateList{}
	l.Add(font, lang)

	return PAD{CandidateList{}, CandidateList{}, l}
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

//Installed leave the installed font in CandidateList only
func (l *CandidateList) Installed(c Collection) {
	for _, v := range *l {
		if len(c.FindByName(v)) == 0 {
			slice.Remove(l, v)
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
	str := genFcPreamble(userMode, "<!-- Default families for Noto Fonts installed on your system.-->")
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

	str += FcSuffix

	return str
}

func genNotoConfig(fonts Collection, userMode bool) string {
	lfpl := LFPLs{}

	nonLangFonts := []string{"Noto Sans", "Noto Sans Disp", "Noto Sans Display",
		"Noto Sans Mono", "Noto Sans Symbols", "Noto Sans Symbols2",
		"Noto Serif", "Noto Serif Disp", "Noto Serif Display",
		"Noto Mono", "Noto Emoji", "Noto Color Emoji"}

	for _, font := range fonts {
		if b, err := slice.Contains(font.Name, nonLangFonts); !b && err == nil && len(font.Lang) > 0 {
			for _, lang := range font.Lang {
				for _, name := range font.UnstyledName() {
					lfpl.AddFont(lang, name, strings.Title(getGenericFamily(name)), "Default")
				}
			}
		}
	}
	completeCJK(&lfpl, fonts)

	return genFcPreamble(userMode, "<!-- Language specific family preference list for Noto Fonts installed on your system.-->") +
		lfpl.GenLFPLsConfig() +
		FcSuffix
}

// notoGenConfigForSpecificGenericFontAndLang generate family preference list of fonts for a generic font name
// and a specific language
func notoGenConfigForSpecificGenericFontAndLang(lfpl LFPL) string {
	str := ""
	for _, generic := range []string{"sans-serif", "serif", "monospace"} {
		mark := generic
		if mark == "sans-serif" {
			mark = "sans"
		}
		mark = strings.Title(mark)
		v := reflect.ValueOf(lfpl).FieldByName(mark) //FPL
		s := "\t<match>\n\t\t<test name=\"family\">\n\t\t\t<string>" + generic + "</string>\n\t\t</test>\n"
		s += "\t\t<test name=\"lang\">\n\t\t\t<string>" + lfpl.Lang + "</string>\n\t\t</test>\n"
		s += "\t\t<edit name=\"family\" mode=\"prepend\">\n"
		s1 := ""
		for _, method := range []string{"Prepend", "Default", "Append"} {
			v1 := v.FieldByName(method)
			if v1.Len() > 0 {
				for i := 0; i < v1.Len(); i++ {
					s1 += "\t\t\t<string>" + v1.Index(i).String() + "</string>\n"
				}
			}
		}
		if len(s1) > 0 {
			s += s1
			s += "\t\t</edit>\n\t</match>\n\n"
			str += s
		}
	}

	return str
}

//genDefaultFamily generate default family fontconfig block for font name
func genDefaultFamily(name string) string {
	str := "\t<alias>\n\t\t<family>" + name + "</family>\n\t\t<default>\n\t\t\t<family>"
	name = getGenericFamily(name)
	if name == "sans" {
		name = "sans-serif"
	}
	str += name
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
	return "sans"
}

func completeCJK(lfpl *LFPLs, c Collection) {
	for i, v := range *lfpl {
		switch v.Lang {
		case "zh-cn", "zh-sg":
			if len(v.Sans.Default) > 0 {
				ppd := v.Sans.Prepend
				ppd = append(ppd, "Noto Sans")
				ppd.Installed(c)
				if len(ppd) > 0 {
					(*lfpl)[i].Sans.Prepend = ppd
				}
				variant := genAllVariantsAlternative(v.Sans.Default[0], c)
				if len(variant) > 0 {
					apd := v.Sans.Append
					apd = append(apd, variant)
					apd.Installed(c)
					if len(apd) > 0 {
						(*lfpl)[i].Sans.Append = apd
					}
				}
			}
			if len(v.Serif.Default) > 0 {
				ppd := v.Serif.Prepend
				ppd = append(ppd, "Noto Serif")
				ppd.Installed(c)
				if len(ppd) > 0 {
					(*lfpl)[i].Serif.Prepend = ppd
				}
				variant := genAllVariantsAlternative(v.Serif.Default[0], c)
				if len(variant) > 0 {
					apd := v.Serif.Append
					apd = append(apd, variant)
					apd.Installed(c)
					if len(apd) > 0 {
						(*lfpl)[i].Serif.Append = apd
					}
				}
			}
		case "zh-tw", "zh-hk", "zh-mo":
			if len(v.Sans.Default) > 0 {
				ppd := v.Sans.Prepend
				ppd = append(ppd, "Noto Sans")
				ppd.Installed(c)
				if len(ppd) > 0 {
					(*lfpl)[i].Sans.Prepend = ppd
				}
				variant := genAllVariantsAlternative(v.Sans.Default[0], c)
				if len(variant) > 0 {
					apd := v.Sans.Append
					apd = append(apd, variant)
					apd.Installed(c)
					if len(apd) > 0 {
						(*lfpl)[i].Sans.Append = apd
					}
				}
			}
			if len(v.Serif.Default) > 0 {
				ppd := v.Serif.Prepend
				ppd = append(ppd, "Noto Serif")
				ppd.Installed(c)
				if len(ppd) > 0 {
					(*lfpl)[i].Serif.Prepend = ppd
				}
				variant := genAllVariantsAlternative(v.Serif.Default[0], c)
				apd := v.Serif.Append
				if len(variant) > 0 {
					apd = append(apd, variant)
				}
				apd = append(apd, "CMEXSong")
				apd.Installed(c)
				if len(apd) > 0 {
					(*lfpl)[i].Serif.Append = apd
				}
			}
		case "ko":
			if len(v.Sans.Default) > 0 {
				ppd := v.Sans.Prepend
				ppd = append(ppd, "Noto Sans")
				ppd.Installed(c)
				if len(ppd) > 0 {
					(*lfpl)[i].Sans.Prepend = ppd
				}
				apd := v.Sans.Append
				variant := genAllVariantsAlternative(v.Sans.Default[0], c)
				if len(variant) > 0 {
					apd = append(apd, variant)
				}
				apd = append(apd, "NanumGothic")
				apd.Installed(c)
				if len(apd) > 0 {
					(*lfpl)[i].Sans.Append = apd
				}
			}
			if len(v.Serif.Default) > 0 {
				ppd := v.Serif.Prepend
				ppd = append(ppd, "Noto Serif")
				ppd.Installed(c)
				if len(ppd) > 0 {
					(*lfpl)[i].Serif.Prepend = ppd
				}
				apd := v.Serif.Append
				variant := genAllVariantsAlternative(v.Serif.Default[0], c)
				if len(variant) > 0 {
					apd = append(apd, variant)
				}
				apd = append(apd, "NanumMyeongjo")
				apd.Installed(c)
				if len(apd) > 0 {
					(*lfpl)[i].Serif.Append = apd
				}
			}
			if len(v.Monospace.Default) > 0 {
				apd := v.Monospace.Append
				apd = append(apd, "NanumGothicCoding")
				apd.Installed(c)
				if len(apd) > 0 {
					(*lfpl)[i].Monospace.Append = apd
				}
			}
		case "ja":
			if len(v.Sans.Default) > 0 {
				ppd := v.Sans.Prepend
				slice.Concat(&ppd, CandidateList{"IPAPGothic", "IPAexGothic", "M+ 1c", "M+ 1p", "VL PGothic", "Noto Sans"})
				ppd.Installed(c)
				if len(ppd) > 0 {
					(*lfpl)[i].Sans.Prepend = ppd
				}
				apd := v.Sans.Append
				variant := genAllVariantsAlternative(v.Sans.Default[0], c)
				if len(variant) > 0 {
					apd = append(apd, variant)
				}
				apd = append(apd, "IPAGothic")
				apd.Installed(c)
				if len(apd) > 0 {
					(*lfpl)[i].Sans.Append = apd
				}
			}
			if len(v.Serif.Default) > 0 {
				ppd := v.Serif.Prepend
				slice.Concat(&ppd, CandidateList{"IPAPMincho", "IPAexMincho", "Noto Serif"})
				ppd.Installed(c)
				if len(ppd) > 0 {
					(*lfpl)[i].Serif.Prepend = ppd
				}
				apd := v.Serif.Append
				variant := genAllVariantsAlternative(v.Serif.Default[0], c)
				if len(variant) > 0 {
					apd = append(apd, variant)
				}
				apd = append(apd, "IPAMincho")
				apd.Installed(c)
				if len(apd) > 0 {
					(*lfpl)[i].Serif.Append = apd
				}
			}
			if len(v.Monospace.Default) > 0 {
				ppd := v.Monospace.Prepend
				slice.Concat(&ppd, CandidateList{"IPAGothic", "M+ 1m", "VL Gothic"})
				ppd.Installed(c)
				if len(ppd) > 0 {
					(*lfpl)[i].Monospace.Prepend = ppd
				}
			}
		}
	}
}

func genAllVariantsAlternative(font string, c Collection) string {
	f := strings.Split(font, " ")
	name := strings.Join(f[:2], " ") + " CJK " + f[len(f)-1]
	if len(c.FindByName(name)) > 0 {
		return name
	}
	return ""
}

func genCJKPrependML(generic, lang string, c Collection) CandidateList {
	m := CandidateList{}
	if generic == "Sans" || generic == "Serif" {
		m = append(m, "Noto "+generic)
	}
	ja := map[string][]string{"Sans": {"IPAPGothic", "IPAexGothic", "M+ 1c", "M+ 1p", "VL PGothic"},
		"Serif":     {"IPAPMincho", "IPAexMincho"},
		"Monospace": {"IPAGothic", "M+ 1m", "VL Gothic"}}
	if lang == "ja" {
		slice.Concat(&m, ja[generic])
	}
	m.Installed(c)
	return m
}

func genCJKAppendML(generic, lang string, c Collection) CandidateList {
	m := CandidateList{}
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
	m.Installed(c)
	return m
}
