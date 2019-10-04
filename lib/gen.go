package lib

import (
	"strconv"
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

func genBlacklistConfig(f Font) string {
	conf := "\t<match target=\"scan\">\n\t\t<test name=\"family\">\n\t\t\t<string>" + f.Name[0] + "</string>\n\t\t</test>\n"
	if !(f.Width == 0 && f.Weight == 0 && f.Slant == 0) {
		if f.Width != 100 {
			conf += "\t\t<test name=\"width\">\n\t\t\t<int>" + strconv.Itoa(f.Width) + "</int>\n\t\t</test>\n"
		}
		if f.Weight != 80 {
			conf += "\t\t<test name=\"weight\">\n\t\t\t<int>" + strconv.Itoa(f.Weight) + "</int>\n\t\t</test>\n"
		}
		if f.Slant != 0 {
			conf += "\t\t<test name=\"slant\">\n\t\t\t<int>" + strconv.Itoa(f.Slant) + "</int>\n\t\t</test>\n"
		}
	}
	conf += "\t\t<edit name=\"charset\" mode=\"assign\">\n\t\t\t<minus>\n\t\t\t\t<name>charset</name>\n"
	conf += genCharsetConfig(f.Charset)
	conf += "\t\t\t</minus>\n\t\t</edit>\n\t</match>\n\n"
	return conf
}

// genCharsetConfig convert Charset to fontconfig conf
func genCharsetConfig(c Charset) string {
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

func genDualConfig(f Font) string {
	str := ""
	for _, name := range f.Name {
		str += "\t<match target=\"font\">\n\t\t<test name=\"family\" compare=\"contains\">\n"
		str += "\t\t\t<string>"
		str += name
		str += "</string>\n\t\t</test>\n"
		str += "\t\t<edit name=\"spacing\" mode=\"append\">\n\t\t\t<const>proportional</const>\n\t\t</edit>\n"
		str += "\t\t<edit name=\"globaladvance\" mode=\"append\">\n\t\t\t<bool>false</bool>\n\t\t</edit>\n\t</match>\n"
	}
	return str
}
