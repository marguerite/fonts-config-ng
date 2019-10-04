package lib

import (
	"log"
)

func fixDualSpacing(fonts Collection, userMode bool) string {
	comment := "<!-- The dual-width Asian fonts (spacing=dual) are not rendered correctly," +
		"apparently FreeType forces all widths to match.\n" +
		"Trying to disable the width forcing code by setting globaladvance=false alone doesn't help.\n" +
		"As a brute force workaround, also set spacing=proportional, i.e. handle them as proportional fonts. -->\n" +
		"<!-- There is a similar problem with dual width bitmap fonts which don't have spacing=dual but mono or charcell.-->\n"
	text := ""

	for _, font := range fonts {
		if font.Dual >= 0 && font.CJK[0] != "none" {
			text += genDualConfig(font)
		}
	}

	if len(text) == 0 {
		return ""
	}
	return genConfigPreamble(userMode, comment) + text + "</fontconfig>\n"
}

// FixDualSpacing fix dual-width Asian fonts
func FixDualSpacing(fonts Collection, userMode bool) {
	text := fixDualSpacing(fonts, userMode)
	dualConfig := GetConfigLocation("dual", userMode)
	err := persist(dualConfig, []byte(text), 0644)
	if err != nil {
		log.Fatalf("Can not write %s: %s\n", dualConfig, err.Error())
	}
}
