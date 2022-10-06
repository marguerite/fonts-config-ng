package lib

import (
	ft "github.com/marguerite/fonts-config-ng/font"
	"github.com/marguerite/go-stdlib/slice"
)

//GenCJKConfig generate cjk specific fontconfig configuration like
// special matrix adjustment for "Noto Sans/Serif", dual-width Asian fonts and etc.
func GenCJKConfig(availFonts ft.Collection, userMode bool) {
	conf := GetFcConfig("cjk", userMode)
	text := genFcPreamble(userMode, "")
	text += fixDualAsianFonts(availFonts, userMode)
	text += aliasNotoCJKOTC(availFonts)
	text += FcSuffix
	overwriteOrRemoveFile(conf, []byte(text))
}

//isSpacingDual find spacing=dual/mono/charcell
func isSpacingDual(font ft.Font) int {
	if font.Spacing > 90 && !font.Outline {
		return 1
	}
	if font.Spacing == 90 {
		return 0
	}
	return -1
}

//isCJKFont find if a font supports CJK
func isCJKFont(font ft.Font) bool {
	supportedLangs := []string{"zh", "ja", "ko", "zh-cn", "zh-tw", "zh-hk", "zh-mo", "zh-sg"}
	if ok, err := slice.Contains(font.Lang, supportedLangs); ok && err == nil {
		return true
	}
	return false
}

//fixDualAsianFonts fix rendering of dual-width Asian fonts (spacing=dual)
func fixDualAsianFonts(availFonts ft.Collection, userMode bool) string {
	comment := "<!-- The dual-width Asian fonts (spacing=dual) are not rendered correctly," +
		"apparently FreeType forces all widths to match.\n" +
		"Trying to disable the width forcing code by setting globaladvance=false alone doesn't help.\n" +
		"As a brute force workaround, also set spacing=proportional, i.e. handle them as proportional fonts. -->\n" +
		"<!-- There is a similar problem with dual width bitmap fonts which don't have spacing=dual but mono or charcell.-->\n\n"
	text := ""

	for _, font := range availFonts {
		if isSpacingDual(font) >= 0 && isCJKFont(font) {
			text += genDualAisanConfig(font)
		}
	}

	if len(text) > 0 {
		return comment + text
	}
	return text
}

func aliasNotoCJKOTC(availFonts ft.Collection) string {
	comment := "<!-- Alias 'Noto Sans/Serif CJK SC/TC/JP/KR' since they may not installed. -->\n\n"
	text := ""
	otcSuffix := []string{" JP", " KR", " SC", " TC"}
	for _, g := range []string{"Sans", "Serif"} {
		for _, o := range otcSuffix {
			name := "Noto " + g + o
			if len(availFonts.FindByName(name)) <= 0 {
				continue
			}
			text += "\t<alias>\n\t\t<family>Noto " + g + " CJK" + o + "</family>\n\t\t<prefer>\n"
			text += "\t\t\t<family>" + name + "</family>\n"
			remain := otcSuffix
			slice.Remove(&remain, o)
			for _, r := range remain {
				text += "\t\t\t<family>Noto " + g + r + "</family>\n"
			}
			text += "\t\t</prefer>\n\t</alias>\n\n"
		}
	}
	if len(text) > 0 {
		return comment + text
	}
	return text
}
