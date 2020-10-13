package lib

import (
	"bytes"
	"os"
	"sync"

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

// isHintedFont check if a font is truetype/cff by reading sfnt version
func isHintedFont(font ft.Font) bool {
	f, _ := os.Open(font.File)
	b := make([]byte, 4)
	f.ReadAt(b, 0)
	f.Close()
	if bytes.Equal(b, []byte{00, 01, 00, 00}) {
		return true
	}
	switch string(b) {
	case "true", "ttcf", "OTTO", "otto":
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
