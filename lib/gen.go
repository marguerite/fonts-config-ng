package lib

import (
	"github.com/marguerite/util/slice"
	"strings"
)

// GenericFamily find generic name through font name
func GenericFamily(fontName string) string {
	if strings.Contains(fontName, " Symbols") {
		return "symbol"
	}
	if strings.Contains(fontName, " Mono") || strings.Contains(fontName, " HW") {
		return "monospace"
	}
	if strings.HasSuffix(fontName, "Emoji") {
		return "emoji"
	}
	if strings.Contains(fontName, " Serif") {
		return "serif"
	}
	return "sans-serif"
}

// GetUnstyledFontName pick unstyled font names
func GetUnstyledFontName(f Font) []string {
	names := f.Name
	s, _ := slice.ShortestString(names)
	slice.Remove(&names, s)
	// trim "Noto Sans Display UI"
	if strings.HasSuffix(s, "UI") {
		s = strings.TrimRight(s, " UI")
	}
	out := []string{s}
	for _, n := range names {
		if !strings.Contains(n, s) {
			out = append(out, n)
		}
	}

	return out
}

// GenerateDefaultFamily return a default family fontconfig block
func GenerateDefaultFamily(fontName string) string {
	return "\t<alias>\n\t\t<family>" + fontName + "</family>\n\t\t<default>\n\t\t\t<family>" +
		GenericFamily(fontName) + "</family>\n\t\t</default>\n\t</alias>\n\n"
}

func generateFontTypeByHinting(fontName string, hinting bool) string {
	txt := "\t<match target=\"font\">\n\t\t<test name=\"family\">\n\t\t\t<string>" + fontName + "</string>\n\t\t</test>\n"
	txt += "\t\t<edit name=\"font_type\" mode=\"assign\">\n\t\t\t<string>"
	if hinting {
		txt += "TT Instructed Font"
	} else {
		txt += "NON TT Instructed Font"
	}
	txt += "</string>\n\t\t</edit>\n\t</match>\n\n"
	return txt
}

// GenerateFontTypeByHinting generate font_type block based on hinting
func GenerateFontTypeByHinting(f Font) string {
	if len(f.Name) > 1 {
		txt := ""
		for _, v := range f.Name {
			txt += generateFontTypeByHinting(v, f.Hinting)
		}
		return txt
	}
	return generateFontTypeByHinting(f.Name[0], f.Hinting)
}

// GenerateFamilyPreferListForLang generate family preference list of fonts for a generic font name
// and a specific language
func GenerateFamilyPreferListForLang(generic, lang string, fonts []string) string {
	txt := "\t<match>\n\t\t<test name=\"family\">\n\t\t\t<string>" + generic + "</string>\n\t\t</test>\n"
	txt += "\t\t<test name=\"lang\">\n\t\t\t<string>" + lang + "</string>\n\t\t</test>\n"
	txt += "\t\t<edit name=\"family\" mode=\"prepend\">\n"
	for _, f := range fonts {
		txt += "\t\t\t<string>" + f + "</string>\n"
	}
	txt += "\t\t</edit>\n\t</match>\n\n"
	return txt
}

// CharsetToFontConfig convert Charset to fontconfig conf
func CharsetToFontConfig(c Charset) string {
	str := "\t\t\t\t<charset>\n"
	for _, v := range c {
		if strings.Contains(v, "..") {
			str += "\t\t\t\t\t<range>\n"
			for _, s := range strings.Split(v, "..") {
				str += "\t\t\t\t\t\t<int>0x" + s + "</int>\n"
			}
			str += "\t\t\t\t\t</range>\n"
		} else {
			str += "\t\t\t\t\t<int>0x" + v + "</int>\n"
		}
	}
	str += "\t\t\t\t</charset>\n"
	return str
}
