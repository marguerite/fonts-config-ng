package lib

func GenTTType(fonts Collection, userMode bool) {
	tt, nonTT := genTTType(fonts, userMode)
	ttFile := GetConfigLocation("tt", userMode)
	nonTTFile := GetConfigLocation("nonTT", userMode)

	overwriteOrRemoveFile(ttFile, []byte(tt), 0644)
	overwriteOrRemoveFile(nonTTFile, []byte(nonTT), 0644)
}

func genTTType(fonts Collection, userMode bool) (string, string) {
	tt := genConfigPreamble(userMode, "TT instructed fonts installed on your system.")
	nonTT := genConfigPreamble(userMode, "NON TT instructed fonts installed on your system.")

	for _, font := range fonts {
		if font.Hinting {
			tt += genFontTypeByHinting(font)
		} else {
			nonTT += genFontTypeByHinting(font)
		}
	}

	tt += "</fontconfig>\n"
	nonTT += "</fontconfig>\n"

	return tt, nonTT
}
