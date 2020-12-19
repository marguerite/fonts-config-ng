package lib

import (
	"strings"

	ft "github.com/marguerite/fonts-config-ng/font"
	"github.com/marguerite/go-stdlib/slice"
)

//GenCJKConfig generate cjk specific fontconfig configuration like
// special matrix adjustment for "Noto Sans/Serif", dual-width Asian fonts and etc.
func GenCJKConfig(availFonts ft.Collection, userMode bool) {
	conf := GetFcConfig("cjk", userMode)
	text := genFcPreamble(userMode, "")
	text += fixDualAsianFonts(availFonts, userMode)
	text += tweakNotoSansSerif(availFonts)
	text += aliasSourceHan(availFonts)
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

func tweakNotoSansSerif(availFonts ft.Collection) string {
	nameLangs := []string{"zh-CN", "zh-SG", "zh-TW", "zh-HK", "zh-MO", "ja", "ko"}
	matrix := []float64{0.90, 0, 0, 1}
	weights := [][]int{{0, 40, 0}, {50, 99, 50}, {99, 179, 80}, {180, 0, 180}}
	widths := []int{63, 100}
	comment := "<!--- Adjust Noto Sans/Serif for CJK\n" +
		"\tThe Latin part of Noto Sans/Serif SC is Adobe Source Sans/Serif Pro,\n" +
		"\tGoogle suggests to prepend Noto Sans/Serif to cover Latin glyphs for CJK,\n" +
		"\tbut Adobe Source Sans/Serif Pro is 2/3 smaller than Noto Sans/Serif.\n" +
		"\twe shrink Noto Sans/Serif with matrix, and adjust its weight/width correspondingly. -->\n\n"
	text := ""

	for _, i := range []string{"Noto Sans", "Noto Serif"} {
		text += genCJKMatrixConfig(i, matrix, nameLangs, availFonts)
		text += genCJKWeightConfig(i, weights, nameLangs, availFonts)
		text += genCJKWidthConfig(i, widths, nameLangs, availFonts)
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

func aliasSourceHan(availFonts ft.Collection) string {
	comment := "<!--- Alias 'Adobe Source Han Sans/Serif/Sans HW' since its CJK part is the same as Noto Sans/Serif.\n" +
		"\t1. We don't need to prepend Source Sans/Serif Pro, since the Latin part has already been.\n" +
		"\t2. If installed manually they can still be used.-->\n\n"
	text := ""

	genericSuffix := []string{"Sans", "Serif", "Sans HW"}
	regionSuffix := []string{" CN", " TW", " JP", " KR"}
	otcSuffix := []string{"", " J", " K", " SC", " TC"}

	for _, g := range genericSuffix {
		for _, r := range regionSuffix {
			text += genSourceHanAliasConfig(g, r, false, availFonts)
		}
		for _, o := range otcSuffix {
			text += genSourceHanAliasConfig(g, o, true, availFonts)
		}
	}
	if len(text) > 0 {
		return comment + text
	}
	return text
}

func genSourceHanAliasConfig(generic, suffix string, otc bool, availFonts ft.Collection) string {
	fontName := "Source Han " + generic + suffix
	hw := strings.Contains(fontName, " HW")
	sufMap := map[string]string{" CN": " SC", " TW": " TC", " J": " JP", " K": " KR"}
	sufs := []string{" JP", " KR", " SC", " TC"}
	notoGeneric := generic
	if notoGeneric == "Sans HW" {
		notoGeneric = "Sans Mono CJK"
	}

	if len(suffix) == 0 {
		suffix = " J"
	}

	notoSuffix := suffix
	if val, ok := sufMap[suffix]; ok {
		notoSuffix = val
	}

	notoName := "Noto " + notoGeneric + notoSuffix

	if len(availFonts.FindByName(notoName)) <= 0 {
		return ""
	}

	str := "\t<alias>\n\t\t<family>" + fontName + "</family>\n"

	if !otc || hw {
		str += "\t\t<accept>\n\t\t\t<family>" + notoName + "</family>\n\t\t</accept>\n"
	} else {
		remain := sufs
		slice.Remove(&remain, notoSuffix)
		str += "\t\t<prefer>\n\t\t\t<family>" + notoName + "</family>\n"
		for _, i := range remain {
			str += "\t\t\t<family>Noto " + notoGeneric + i + "</family>\n"
		}
		str += "\t\t</prefer>\n"
	}

	str += "\t</alias>\n\n"

	return str
}
