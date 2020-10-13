package font

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/marguerite/util/fileutils"
	"github.com/marguerite/util/slice"
	fccharset "github.com/openSUSE/fonts-config/fc-charset"
)

//Collection A collection of type Font
type Collection []Font

// NewCollection Initialize a new collection of Font from system installed fonts queried by fc-cat.
func NewCollection() Collection {
	paths := getFontPaths()
	out, err := exec.Command("/usr/bin/fc-cat").Output()
	if err != nil {
		panic(err)
	}

	fonts := Collection{}

	for _, line := range strings.Split(string(out), "\n") {
		font := Font{}
		for _, i := range strings.Split(line, "\"") {
			if len(i) == 0 {
				continue
			}
			if _, err := strconv.Atoi(strings.TrimSpace(i)); err == nil {
				continue
			}
			// reject directory and font format usually not used for display
			if fileutils.HasPrefixOrSuffix(i, ".dir", ".pcf.gz", ".pfa", ".pfb", ".afm", ".otb") == 0 {
				// Multi thread here
				NewFont(&font, i, paths)
			}

		}
		if len(font.File) != 0 {
			fonts = append(fonts, font)
		}
	}

	return fonts
}

// FindByName Find Fonts by font name string or font name regexp pattern
func (c Collection) FindByName(restricts ...interface{}) Collection {
	newC := Collection{}
	for _, font := range c {
		if _, err := getMatchedFontName(font.Name, restricts...); err == nil {
			newC = append(newC, font)
		}
	}
	return newC
}

// FilterNameList given a list of font names, leave those in the collection in list
// usually used to avoid useless fontconfig rules or trash in FC_DEBUG
func (c Collection) FilterNameList(list *[]string) {
	for _, name := range *list {
		if len(c.FindByName(name)) == 0 {
			slice.Remove(list, name)
		}
	}
}

//Font font struct with informations we need
type Font struct {
	File    string
	Name    []string
	Lang    []string
	Width   int
	Weight  int
	Slant   int
	Spacing int
	Outline bool
	fccharset.Charset
}

// NewFont generate a new Font structure from input string
func NewFont(font *Font, in string, paths map[string]string) {
	// parse Filename or other information
	if strings.Contains(in, ":") {
		for idx, i := range strings.Split(in, ":") {
			if idx == 0 {
				font.Name = strings.Split(strings.TrimSpace(i), ",")
				continue
			}
			arr := strings.Split(i, "=")
			if ok, err := slice.Contains([]string{"width", "weight", "slant", "spacing"}, arr[0]); ok && err == nil {
				// convert to value to int
				val, err := strconv.ParseInt(arr[1], 10, 64)
				if err != nil {
					continue
				}

				v := reflect.Indirect(reflect.ValueOf(font))
				v.FieldByName(strings.Title(arr[0])).SetInt(val)
			}
			if arr[0] == "outline" {
				val, err := strconv.ParseBool(i)
				if err != nil {
					continue
				}
				font.Outline = val
			}
			if arr[0] == "lang" {
				font.Lang = strings.Split(strings.TrimSpace(arr[1]), "|")
			}
			if arr[0] == "charset" {
				font.Charset = fccharset.NewCharset(strings.TrimSpace(arr[1]))
			}
		}
	} else {
		font.File = paths[in]
	}
}

// IsEmoji whether a font is a emoji font
func (f Font) IsEmoji() bool {
	if ok, err := slice.Contains(f.Lang, "und-zsye"); ok && err == nil {
		return true
	}
	return false
}

func getMatchedFontName(names []string, restricts ...interface{}) ([]string, error) {
	if len(restricts) == 0 {
		return names, nil
	}

	_, ok := restricts[0].(*regexp.Regexp)
	if reflect.ValueOf(restricts[0]).Kind() != reflect.String && !ok {
		return []string{}, fmt.Errorf("restrict term must be of type 'string' or '*regexp.Regexp'")
	}

	for _, name := range names {
		for _, restrict := range restricts {
			if ok {
				if restrict.(*regexp.Regexp).MatchString(name) {
					return names, nil
				}
			} else {
				if strings.Contains(name, restrict.(string)) {
					return names, nil
				}
			}
		}
	}
	return []string{}, fmt.Errorf("no matched name found")
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
					fonts[filepath.Base(font)] = font
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
