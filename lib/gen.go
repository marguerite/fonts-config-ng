package lib

import (
	"strconv"

	ft "github.com/marguerite/fonts-config-ng/font"
)

// genFcPreamble generate fontconfig preamble
func genFcPreamble(userMode bool, comment string) string {
	cfg := "<?xml version=\"1.0\"?>\n<!DOCTYPE fontconfig SYSTEM \"fonts.dtd\">\n\n<!-- DO NOT EDIT; this is a generated file -->\n<!-- modify "
	cfg += "/etc/sysconfig/fonts-config && run /usr/bin/fonts-config "
	if userMode {
		cfg += "-\\-user "
	}
	cfg += "instead. -->\n"
	cfg += comment
	cfg += "\n<fontconfig>\n\n"
	return cfg
}

func genBlacklistConfig(b Blacklist) string {
	cfg := "\t<match target=\"scan\">\n\t\t<test name=\"family\">\n\t\t\t<string>" + b.Name + "</string>\n\t\t</test>\n"
	cfg += "\t\t<edit name=\"charset\" mode=\"assign_replace\">\n\t\t\t<minus>\n\t\t\t\t<name>charset</name>\n\t\t\t\t<charset>\n"
	for _, v := range b.Charset {
		if v.Min != v.Max {
			cfg += "\t\t\t\t\t<range>\n"
			cfg += "\t\t\t\t\t\t<int>0x" + strconv.FormatUint(v.Min, 16) + "</int>\n"
			cfg += "\t\t\t\t\t\t<int>0x" + strconv.FormatUint(v.Max, 16) + "</int>\n"
			cfg += "\t\t\t\t\t</range>\n"
		} else {
			cfg += "\t\t\t\t\t<int>0x" + strconv.FormatUint(v.Min, 16) + "</int>\n"
		}
	}
	cfg += "\t\t\t\t</charset>\n\t\t\t</minus>\n\t\t</edit>\n\t</match>\n\n"
	return cfg
}

func genDualAisanConfig(font ft.Font) (cfg string) {
	for _, name := range font.Name {
		cfg += "\t<match target=\"font\">\n\t\t<test name=\"family\" compare=\"contains\">\n\t\t\t<string>"
		cfg += name
		cfg += "</string>\n\t\t</test>\n"
		cfg += "\t\t<edit name=\"spacing\" mode=\"append\">\n\t\t\t<const>proportional</const>\n\t\t</edit>\n"
		cfg += "\t\t<edit name=\"globaladvance\" mode=\"append\">\n\t\t\t<bool>false</bool>\n\t\t</edit>\n\t</match>\n\n"
	}
	return cfg
}
