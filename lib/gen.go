package lib

import (
	"strconv"

	ft "github.com/marguerite/fonts-config-ng/font"
)

// genFcPreamble generate fontconfig preamble
func genFcPreamble(userMode bool, comment string) string {
	config := "<?xml version=\"1.0\"?>\n<!DOCTYPE fontconfig SYSTEM \"fonts.dtd\">\n\n<!-- DO NOT EDIT; this is a generated file -->\n<!-- modify "
	config += "/etc/sysconfig/fonts-config && run /usr/bin/fonts-config "
	if userMode {
		config += "-\\-user "
	}
	config += "instead. -->\n"
	config += comment
	config += "\n<fontconfig>\n\n"
	return config
}

func genBlacklistConfig(b Blacklist) string {
	config := "\t<match target=\"scan\">\n\t\t<test name=\"family\">\n\t\t\t<string>" + b.Name + "</string>\n\t\t</test>\n"
	config += "\t\t<edit name=\"charset\" mode=\"assign\">\n\t\t\t<minus>\n\t\t\t\t<name>charset</name>\n"
	for _, v := range b.Charset {
		config += "\t\t\t\t<charset>\n"
		if v.Min != v.Max {
			config += "\t\t\t\t\t<range>\n"
			config += "\t\t\t\t\t\t<int>0x" + strconv.FormatUint(v.Min, 16) + "</int>\n"
			config += "\t\t\t\t\t\t<int>0x" + strconv.FormatUint(v.Max, 16) + "</int>\n"
			config += "\t\t\t\t\t</range>\n"
		} else {
			config += "\t\t\t\t\t<int>0x" + strconv.FormatUint(v.Min, 16) + "</int>\n"
		}
		config += "\t\t\t\t</charset>\n"
	}
	config += "\t\t\t</minus>\n\t\t</edit>\n\t</match>\n\n"
	return config
}

func genDualAisanConfig(font ft.Font) string {
	config := ""
	for _, name := range font.Name {
		config += "\t<match target=\"font\">\n\t\t<test name=\"family\" compare=\"contains\">\n\t\t\t<string>"
		config += name
		config += "</string>\n\t\t</test>\n"
		config += "\t\t<edit name=\"spacing\" mode=\"append\">\n\t\t\t<const>proportional</const>\n\t\t</edit>\n"
		config += "\t\t<edit name=\"globaladvance\" mode=\"append\">\n\t\t\t<bool>false</bool>\n\t\t</edit>\n\t</match>\n\n"
	}
	return config
}
