package lib

func GenTTType(fonts Collection, userMode bool) {
	tt, nonTT := genTTType(fonts, userMode)
	ttFile := GetConfigLocation("tt", userMode)
	nonTTFile := GetConfigLocation("nonTT", userMode)

	overwriteOrRemoveFile(ttFile, []byte(tt), 0644)
	overwriteOrRemoveFile(nonTTFile, []byte(nonTT), 0644)
}

func genTTType(fonts Collection, userMode bool) (string, string) {
	tt := genConfigPreamble(userMode, "<!-- TT instructed fonts installed on your system. Maybe CFF/PostScript based or Truetype based. -->")
	nonTT := genConfigPreamble(userMode, "<!-- NON TT instructed fonts installed on your system.-->")

	// font names across different font.Name may be equal.
	m := make(map[string]struct{})

	for _, font := range fonts {
		for _, name := range font.Name {
			if _, ok := m[name]; !ok {
				m[name] = struct{}{}
				if font.Hinting {
					tt += genFontTypeByHinting(name, true)
				} else {
					nonTT += genFontTypeByHinting(name, false)
				}
			}
		}
	}

	tt += "</fontconfig>\n"
	nonTT += "</fontconfig>\n"

	return tt, nonTT
}
