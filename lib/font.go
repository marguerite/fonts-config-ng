package lib

import (
	"fmt"
	"os/exec"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/marguerite/util/fileutils"
	"github.com/marguerite/util/slice"
)

//Collection A collection of type Font
type Collection []Font

// NewCollection Initialize a new collection of Font from system installed fonts queried by fc-cat.
func NewCollection() Collection {
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
				NewFont(&font, i)
			}

		}
		if len(font.File) != 0 {
			fonts = append(fonts, font)
		}
	}

	return fonts
}

//FindByName Find Fonts by name restricts
func (c Collection) FindByName(restricts ...interface{}) Collection {
	newC := Collection{}
	for _, font := range c {
		if _, err := getMatchedFontName(font.Name, restricts...); err == nil {
			newC = append(newC, font)
		}
	}
	return newC
}

func (c *Collection) AppendCharsetOrFont(f Font) {
	if i, ok := c.Find(f); ok {
		(*c)[i].AppendCharset(f.Charset)
	} else {
		*c = append(*c, f)
	}
}

//Find whether font collection contains a specific font and return its index
func (c Collection) Find(f Font) (int, bool) {
	for i, j := range c {
		if j.File == j.File {
			return i, true
		}
	}
	return 0, false
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
	Charset
}

// NewFont generate a new Font structure from input string
func NewFont(font *Font, in string) {
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
				font.Charset = NewCharset(strings.TrimSpace(arr[1]))
			}
		}
	} else {
		font.File = in
	}
}

//SetName set font name
func (f *Font) SetName(name []string) {
	f.Name = name
}

//SetStyle set font style
func (f *Font) SetStyle(width, weight, slant int) {
	f.Width = width
	f.Weight = weight
	f.Slant = slant
}

//SetCharset set font charsets
func (f *Font) SetCharset(charset Charset) {
	f.Charset = charset
}

//AppendCharset append charset to an existing Font
func (f *Font) AppendCharset(c Charset) {
	c1 := f.Charset
	slice.Concat(&c1, c)
	f.Charset = c1
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
