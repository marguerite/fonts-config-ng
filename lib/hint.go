package lib

import (
	"github.com/golang/freetype"
	"github.com/marguerite/util/fileutils"
	"io/ioutil"
	"reflect"
)

// isHinted checks if a font has hinting instructions, for ".ttf" fonts, it checks
// if it has builtin hinting instructions by reading its "fpgm" table;
// basically hinting instructions are stored in "cvt", "fpgm", "prep" table.
// fpgm are the key table because its the actual bytecode intepreter virtual machine
// https://developer.apple.com/fonts/TrueType-Reference-Manual/RM06/Chap6fpgm.html
// https://developer.apple.com/fonts/TrueType-Reference-Manual/RM03/Chap3.html
// for ".otf" fonts, the hinting intelligence is in the rasterizer that Adobe
// contributed to fontconfig.
// https://blog.typekit.com/2010/12/02/the-benefits-of-opentypecff-over-truetype/
func isHinted(f string) (bool, error) {
	if fileutils.HasPrefixOrSuffix(f, ".ttf", ".ttc") != 0 {
		return ttfHinted(f)
	}
	if fileutils.HasPrefixOrSuffix(f, ".otf", ".otc", ".pfa", ".pfb") != 0 {
		return true, nil
	}
	return false, nil
}

func ttfHinted(f string) (bool, error) {
	b, e := ioutil.ReadFile(f)
	if e != nil {
		return false, e
	}
	font, e := freetype.ParseFont(b)
	if e != nil {
		return false, e
	}
	p := reflect.ValueOf(font)
	v := reflect.Indirect(p)
	fpgm := v.FieldByName("fpgm")
	if fpgm.Len() > 0 {
		return true, nil
	}
	return false, nil
}
