package lib

import (
	"encoding/json"
	"fmt"
	"github.com/marguerite/util/dirutils"
	"github.com/marguerite/util/fileutils"
	"github.com/marguerite/util/slice"
	"log"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

//Collection A collection of type Font
type Collection []Font

//NewCollection Initialize a new collection of type Font from a font file.
func NewCollection(file string) Collection {
	out, err := exec.Command("/usr/bin/fc-scan", file).Output()
	if err != nil {
		log.Fatal("Can't run 'fc-scan " + file + "'.")
	}

	c := Collection{}
	hint, _ := Hinting(file)
	charset := BuildCharset(file)

	re := regexp.MustCompile(`(?ms)Pattern has(.*?)^\n`)
	match := re.FindAllStringSubmatch(string(out), -1)

	for _, m := range match {
		name := parseFontNames(m[1])
		lang := parseFontLangs(m[1])
		width, weight, slant := parseFontStyle(m[1])
		c = append(c, Font{name, lang, file, hint, width, weight, slant, charset})
	}

	return c
}

//GetFontPaths Get Fonts' path information
//  It can restrict the results by path with "string" or "*regexp.Regexp".
func (c Collection) GetFontPaths(restricts ...interface{}) []string {
	paths := []string{}
	for _, font := range c {
		if path, err := restrictPath(font.Path, restricts...); err == nil {
			paths = append(paths, path)
		}
	}
	return paths
}

//Contains if an element of the Collection has the same name and style with the provided Font
func (c Collection) Contains(f Font) (int, bool) {
	fv := reflect.ValueOf(f)
	for i, j := range c {
		if reflect.DeepEqual(reflect.ValueOf(j).FieldByName("Name").Interface(), fv.FieldByName("Name").Interface()) &&
			reflect.DeepEqual(reflect.ValueOf(j).FieldByName("Width").Interface(), fv.FieldByName("Width").Interface()) &&
			reflect.DeepEqual(reflect.ValueOf(j).FieldByName("Weight").Interface(), fv.FieldByName("Weight").Interface()) &&
			reflect.DeepEqual(reflect.ValueOf(j).FieldByName("Slant").Interface(), fv.FieldByName("Slant").Interface()) {
			return i, true
		}
	}
	return 0, false
}

// Encode encode Collections to json
func (c Collection) Encode() ([]byte, error) {
	b, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		return []byte{}, err
	}
	return b, nil
}

// Decode decode json to Collections
func (c *Collection) Decode(b []byte) error {
	err := json.Unmarshal(b, &c)
	return err
}

//Font Struct contains various font informations
type Font struct {
	Name    []string
	Lang    []string
	Path    string
	Hinting bool
	Width   int
	Weight  int
	Slant   int
	Charset
}

//AppendCharset append charset to an existing Font
func (f *Font) AppendCharset(c Charset) {
	c1 := f.Charset
	slice.Concat(&c1, c)
	f.Charset = c1
}

//Charset Font's Charset
type Charset []string

func (c Charset) Len() int { return len(c) }

func (c Charset) Swap(i, j int) { c[i], c[j] = c[j], c[i] }

func (c Charset) Less(i, j int) bool {
	a, _ := strconv.ParseInt(c[i], 16, 0)
	b, _ := strconv.ParseInt(c[j], 16, 0)
	return a < b
}

func parseFontNames(data string) []string {
	names := []string{}
	reFamily := regexp.MustCompile(`family: (.*)\n`)
	match := reFamily.FindAllStringSubmatch(data, -1)
	for _, m := range match {
		raw := strings.Replace(m[1], "(s)", "", -1)
		candidates := strings.Split(raw, "\"")
		for _, c := range candidates {
			// sometimes the len of candidate is 1, but is useless for a font name anyway
			if len(c) > 1 && !strings.HasSuffix(c, "Regular") && !strings.HasSuffix(c, "Book") {
				names = append(names, c)
			}
		}
	}
	slice.Unique(&names)
	return names
}

func parseFontLangs(data string) []string {
	langs := []string{}
	reLang := regexp.MustCompile(`[^ey]lang: ([^(]+)`)
	match := reLang.FindAllStringSubmatch(string(data), -1)
	for _, m := range match {
		for _, lang := range strings.Split(m[1], "|") {
			langs = append(langs, lang)
		}
	}
	slice.Unique(langs)
	return langs
}

func parseFontStyle(data string) (int, int, int) {
	reStyle := regexp.MustCompile(`(width|weight|slant): (\d+)`)
	match := reStyle.FindAllStringSubmatch(data, -1)
	width := 0
	weight := 0
	slant := 0
	for _, m := range match {
		switch m[1] {
		case "width":
			width, _ = strconv.Atoi(m[2])
		case "weight":
			weight, _ = strconv.Atoi(m[2])
		default:
			slant, _ = strconv.Atoi(m[2])
		}
	}
	return width, weight, slant
}

//LoadFonts Incrementally load global and local fonts.
// It can restrict the results by path with "string" or "*regexp.Regexp".
func LoadFonts(c Collection, restricts ...interface{}) Collection {
	newCollection := c
	// Existing collection
	pathsExisting := c.GetFontPaths(restricts...)
	// Installed Fonts
	fontsInstalled := GetFontsInstalled(restricts...)

	slice.Remove(&fontsInstalled, pathsExisting)

	for _, font := range fontsInstalled {
		slice.Concat(&newCollection, NewCollection(font))
	}

	return newCollection
}

func restrictPath(path string, restricts ...interface{}) (string, error) {
	if len(restricts) == 0 {
		return path, nil
	}

	_, ok := restricts[0].(*regexp.Regexp)
	if reflect.ValueOf(restricts[0]).Kind() != reflect.String && !ok {
		return "", fmt.Errorf("Restrict term must be of type 'string' or '*regexp.Regexp'.")
	}

	base := filepath.Base(path)

	for _, restrict := range restricts {
		if ok {
			if restrict.(*regexp.Regexp).MatchString(base) {
				return path, nil
			}
		} else {
			if strings.Contains(base, restrict.(string)) {
				return path, nil
			}
		}
	}
	return "", fmt.Errorf("No matched path found.")
}

//GetFontsInstalled Get all font files installed on your system
// It can restrict the results by path with "string" or "*regexp.Regexp".
func GetFontsInstalled(restricts ...interface{}) []string {
	local := filepath.Join(GetEnv("HOME"), ".fonts")
	candidates := []string{}

	for _, dir := range []string{local, "/usr/share/fonts"} {
		fonts, _ := dirutils.Ls(dir)
		for _, font := range fonts {
			if fileutils.HasPrefixOrSuffix(filepath.Base(font), "font", ".", ".dir") == 0 {
				if path, err := restrictPath(font, restricts...); err == nil {
					candidates = append(candidates, font)
				}
			}
		}
	}
	return candidates
}
