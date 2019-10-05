package lib

import (
	"strconv"
	"strings"
)

// genConfigPreamble generate fontconfig preamble
func genConfigPreamble(userMode bool, comment string) string {
	config := "<?xml version=\"1.0\"?>\n<!DOCTYPE fontconfig SYSTEM \"fonts.dtd\">\n\n<!-- DO NOT EDIT; this is a generated file -->\n<!-- modify "
	config += GetConfigLocation("fc", false)
	config += " && run /usr/bin/fonts-config "
	if userMode {
		config += "-\\-user "
	}
	config += "instead. -->\n"
	config += comment
	config += "\n<fontconfig>\n"
	return config
}

//genFontTypeByHinting generate fontconfig font_type block based on tt hinting.
func genFontTypeByHinting(name string, hinting bool) string {
	str := "\t<match target=\"font\">\n\t\t<test name=\"family\">\n\t\t\t<string>" + name + "</string>\n\t\t</test>\n"
	str += "\t\t<edit name=\"font_type\" mode=\"assign\">\n\t\t\t<string>"
	if hinting {
		str += "TT Instructed Font"
	} else {
		str += "NON TT Instructed Font"
	}
	str += "</string>\n\t\t</edit>\n\t</match>\n\n"
	return str
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
