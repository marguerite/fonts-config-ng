package lib

import (
	"io/ioutil"
	"log"
	"os/exec"
	"path/filepath"
	"reflect"
	"sync"

	"github.com/golang/freetype"
	"github.com/marguerite/util/fileutils"
)

//GenTTType group fonts based on their hinting instructions' existences
func GenTTType(fonts Collection, userMode bool) {
	tt, nonTT := genTTType(fonts, userMode)
	ttFile := GetConfigLocation("tt", userMode)
	nonTTFile := GetConfigLocation("nonTT", userMode)

	overwriteOrRemoveFile(ttFile, []byte(tt), 0644)
	overwriteOrRemoveFile(nonTTFile, []byte(nonTT), 0644)
}

// getFontPaths get all system installed font's paths via fc-list
func getFontPaths() map[string]string {
	out, err := exec.Command("/usr/bin/fc-list").Output()
	if err != nil {
		log.Fatal("no fc-list found")
	}

	tmp := []byte{}
	fonts := make(map[string]string)
	first := true

	for _, b := range out {
		if b == ':' {
			if first {
				font := string(tmp)
				if fileutils.HasPrefixOrSuffix(font, ".pcf.gz", ".pfa", ".pfb", ".afm", ".otb") == 0 {
					fonts[filepath.Base(font)] = filepath.Dir(font)
				}
			}
			tmp = []byte{}
			first = false
			continue
		}
		if b == '\n' {
			tmp = []byte{}
			first = true
			continue
		}
		tmp = append(tmp, b)
	}
	return fonts
}

// isHintedFont checks if a font has hinting instructions, for ".ttf" font, it stores
// builtin hinting instructions in "cvt", "fpgm", "prep" table. fpgm table is the
// most important table because it's the actual bytecode intepreter virtual machine
// https://developer.apple.com/fonts/TrueType-Reference-Manual/RM06/Chap6fpgm.html
// https://developer.apple.com/fonts/TrueType-Reference-Manual/RM03/Chap3.html
// for ".otf" fonts, the hinting intelligence is in the rasterizer that Adobe
// contributed to fontconfig.
// https://blog.typekit.com/2010/12/02/the-benefits-of-opentypecff-over-truetype/
func isHintedFont(font Font) bool {
	if fileutils.HasPrefixOrSuffix(font.File, ".ttf", ".ttc") != 0 {
		if val, ok := getFontPaths()[font.File]; ok {
			return ttfHasFpgm(filepath.Join(val, font.File))
		}
		return false
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

func genTTType(fonts Collection, userMode bool) (string, string) {
	tt := genFcPreamble(userMode, "<!-- TT instructed fonts installed on your system. Maybe CFF/PostScript based or Truetype based. -->")
	nontt := genFcPreamble(userMode, "<!-- NON TT instructed fonts installed on your system.-->")

	wg := sync.WaitGroup{}
	wg.Add(len(fonts))
	mux := sync.Mutex{}
	ch := make(chan struct{}, 100) // ch is a chan to avoid "too many open files" when os exec

	for _, font := range fonts {
		go func(f Font, tt, nontt *string) {
			defer wg.Done()
			defer func() { <-ch }() // release chan
			ch <- struct{}{}        // acquire chan

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
