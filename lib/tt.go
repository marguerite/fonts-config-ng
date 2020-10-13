package lib

import (
	"io/ioutil"
	"reflect"
	"sync"

	"github.com/golang/freetype"
	"github.com/marguerite/util/fileutils"
	ft "github.com/openSUSE/fonts-config/font"
)

//GenTTType group fonts based on their hinting instructions' existences
func GenTTType(c ft.Collection, userMode bool) {
	tt, nonTT := genTTType(c, userMode)
	ttFile := GetFcConfig("tt", userMode)
	nonTTFile := GetFcConfig("nonTT", userMode)

	overwriteOrRemoveFile(ttFile, []byte(tt))
	overwriteOrRemoveFile(nonTTFile, []byte(nonTT))
}

// isHintedFont checks if a font has hinting instructions, for ".ttf" font, it stores
// builtin hinting instructions in "cvt", "fpgm", "prep" table. fpgm table is the
// most important table because it's the actual bytecode intepreter virtual machine
// https://developer.apple.com/fonts/TrueType-Reference-Manual/RM06/Chap6fpgm.html
// https://developer.apple.com/fonts/TrueType-Reference-Manual/RM03/Chap3.html
// for ".otf" fonts, the hinting intelligence is in the rasterizer that Adobe
// contributed to fontconfig.
// https://blog.typekit.com/2010/12/02/the-benefits-of-opentypecff-over-truetype/
func isHintedFont(font ft.Font) bool {
	if fileutils.HasPrefixOrSuffix(font.File, ".ttf", ".ttc") != 0 {
		return ttfHasFpgm(font.File)
	}
	if fileutils.HasPrefixOrSuffix(font.File, ".otf", ".otc") != 0 {
		return true
	}
	return false
}

// ttfHasFpgm if .ttf font has fpgm table and content
func ttfHasFpgm(path string) bool {
	b, e := ioutil.ReadFile(path)
	if e != nil {
		return false
	}
	font, e := freetype.ParseFont(b)
	if e != nil {
		return false
	}
	fpgm := reflect.Indirect(reflect.ValueOf(font)).FieldByName("fpgm")
	if fpgm.Len() > 0 {
		return true
	}
	return false
}

func genTTType(c ft.Collection, userMode bool) (string, string) {
	tt := genFcPreamble(userMode, "<!-- TT instructed fonts installed on your system. Maybe CFF/PostScript based or Truetype based. -->")
	nontt := genFcPreamble(userMode, "<!-- NON TT instructed fonts installed on your system.-->")

	wg := sync.WaitGroup{}
	wg.Add(len(c))
	mux := sync.Mutex{}

	for _, font := range c {
		go func(f ft.Font, tt, nontt *string) {
			defer wg.Done()

			hint := isHintedFont(f)

			// font.Name across different Fonts may contain equal values.
			names := f.Name
			if len(f.Name) > 1 {
				names = f.Name[1:]
			}

			for _, n := range names {
				if hint {
					mux.Lock()
					*tt += genFontTypeByHinting(n, true)
					mux.Unlock()
					continue
				}
				mux.Lock()
				*nontt += genFontTypeByHinting(n, false)
				mux.Unlock()
			}
		}(font, &tt, &nontt)
	}

	wg.Wait()

	tt += FcSuffix
	nontt += FcSuffix

	return tt, nontt
}
