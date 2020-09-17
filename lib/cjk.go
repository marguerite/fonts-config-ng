package lib

//GenCJKConfig generate cjk specific fontconfig configuration like
// special matrix adjustment for "Noto Sans/Serif", dual-width Asian fonts and etc.
func GenCJKConfig(fonts Collection, userMode bool) {
	conf := GetConfigLocation("cjk", userMode)
	text := fixDualAsianFonts(fonts, userMode)
	text += tweakNotoSansSerif(userMode)
	text += FontConfigSuffix
	overwriteOrRemoveFile(conf, []byte(text), 0644)
}

//fixDualAsianFonts fix rendering of dual-width Asian fonts (spacing=dual)
func fixDualAsianFonts(fonts Collection, userMode bool) string {
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
	return genConfigPreamble(userMode, comment) + text
}

func tweakNotoSansSerif(userMode bool) string {
	nameLangs := []string{"zh-CN", "zh-SG", "zh-TW", "zh-HK", "zh-MO", "ja", "ko"}
	matrix := []float64{0.67, 0, 0, 0.67}

	comment := "<!--- Currently we use Region Specific Subset OpenType/CFF (Subset OTF)\n" +
		"\tflavor of Google's Noto Sans/Serif CJK fonts, but previously we\n" +
		"\tused Super OpenType/CFF Collection (Super OTC), and other distributions\n" +
		"\tmay use Language specific OpenType/CFF (OTF) flavor. So\n" +
		"\tNoto Sans/Serif CJK SC/TC/JP/KR are also common font names.-->\n\n" +
		"<!--- fontconfig doesn't support the OpenType locl GSUB feature,\n" +
		"\tso only the default glyph variant (JP) can be used in the\n" +
		"\tSuper OTC and OTF flavors. We gave them very low priority\n" +
		"\ton openSUSE even if they were installed manually.-->\n\n" +
		"<!--- 1. Prepend 'Noto Sans/Serif' before CJK because the Latin part is from\n" +
		"\t\t'Adobe Source Sans/Serif Pro' which is 2/3 smaller than Noto.-->\n"

	text := comment
	text += genMatrix("Noto Sans", matrix, nameLangs)
	text += genMatrix("Noto Serif", matrix, nameLangs)

	return text
}
