package lib

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	dirutils "github.com/marguerite/util/dir"
	"github.com/marguerite/util/fileutils"
	"github.com/marguerite/util/slice"
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
	hint, _ := isHinted(file)
	charset := NewCharset(file)

	re := regexp.MustCompile(`(?ms)Pattern has(.*?)^\n`)
	match := re.FindAllStringSubmatch(string(out), -1)

	for _, m := range match {
		name := parseFontNames(m[1])
		lang := parseFontLangs(m[1])
		cjk := parseCJKSupport(lang)

		// cjk fonts usually claims all the langs, make clean
		if cjk[0] != "none" {
			tmp := lang
			slice.Remove(&tmp, cjk)
			slice.Remove(&lang, tmp)
		}

		width, weight, slant := parseFontStyle(m[1])
		spacing := parseSpacing(m[1])
		outline := parseOutline(m[1])
		dual := isDual(spacing, outline)
		c = append(c, Font{name, lang, cjk, file, hint, width, weight, slant, spacing, outline, dual, charset})
	}

	return c
}

//GetFontPaths Get Fonts' path information
func (c Collection) GetFontPaths() []string {
	paths := []string{}
	for _, font := range c {
		paths = append(paths, font.Path)
	}
	return paths
}

//FindByPath Find Fonts by path restricts
func (c Collection) FindByPath(restricts ...interface{}) Collection {
	newC := Collection{}
	for _, font := range c {
		if _, err := restrictPath(font.Path, restricts...); err == nil {
			newC = append(newC, font)
		}
	}
	return newC
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
	if i, ok := c.Contains(f); ok {
		(*c)[i].AppendCharset(f.Charset)
	} else {
		*c = append(*c, f)
	}
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
	CJK     []string
	Path    string
	Hinting bool
	Width   int
	Weight  int
	Slant   int
	Spacing int
	Outline bool
	Dual    int
	Charset
}

//UnstyledName Get unstyled font names, eg: without "Bold/Italic"
func (f Font) UnstyledName() []string {
	names := f.Name
	s, _ := slice.ShortestString(f.Name)
	// Trim "Noto Sans Display UI"
	if strings.HasSuffix(s, "UI") {
		s = strings.TrimRight(s, " UI")
	}
	unstyled := []string{s}
	slice.Remove(&names, s)
	for _, name := range names {
		if !strings.Contains(name, s) {
			unstyled = append(unstyled, name)
		}
	}
	return unstyled
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

//Charset Font's Charset
type Charset []string

func (c Charset) Len() int { return len(c) }

func (c Charset) Swap(i, j int) { c[i], c[j] = c[j], c[i] }

func (c Charset) Less(i, j int) bool {
	a, _ := strconv.ParseInt(c[i], 16, 0)
	b, _ := strconv.ParseInt(c[j], 16, 0)
	return a < b
}

//NewCharset generate charset array of a font
func NewCharset(font string) Charset {
	if _, err := os.Stat(font); !os.IsNotExist(err) {
		out, _ := exec.Command("/usr/bin/fc-scan", "--format", "%{charset}", font).Output()
		return genPlainCharset(strings.Split(string(out), " "))
	}
	return Charset{}
}

//Intersect get intersected array of two charsets
func (c Charset) Intersect(charset Charset) Charset {
	slice.Intersect(&charset, c)
	sort.Sort(charset)
	return genRangedCharset(charset)
}

//Concat concat two charsets into one
func (c Charset) Concat(charset Charset) Charset {
	c1 := genPlainCharset(c)
	c2 := genPlainCharset(charset)
	slice.Concat(&c1, c2)
	return genRangedCharset(c1)
}

// genRangedCharset convert int to range in charset
func genRangedCharset(c Charset) Charset {
	if len(c) < 2 {
		return c
	}

	charset := Charset{}

	sort.Sort(c)
	idx := -1

	for i := 0; i < len(c); i++ {
		if i == len(c)-1 {
			if idx >= 0 && minusChar(c[i], c[i-1]) == 1 {
				charset = append(charset, c[idx]+".."+c[i])
			} else {
				charset = append(charset, c[i])
			}
			continue
		}

		if minusChar(c[i+1], c[i]) == 1 {
			if idx < 0 {
				idx = i
			}
		} else {
			if idx >= 0 {
				charset = append(charset, c[idx]+".."+c[i])
				idx = -1
			} else {
				charset = append(charset, c[i])
			}
		}
	}

	return charset
}

func minusChar(c1, c2 string) int64 {
	a, _ := strconv.ParseInt(c1, 16, 0)
	b, _ := strconv.ParseInt(c2, 16, 0)
	return a - b
}

func parseCharsetRange(s string) Charset {
	r := Charset{}
	// some Simplified Chinese contains 2fa15-2fa1c20-7e, which is actually
	// 3 loops of a same charset. the break point is not "-7e", but "20-7e",
	// which is the begin char of the next loop
	// eg: fc-scan --format "%{charset}" /usr/share/fonts/truetype/wqy-zenhei.ttc
	re := regexp.MustCompile(`^(\w+)-(\w+)(20-7[0-9a-f])?$`)
	m := re.FindStringSubmatch(s)
	start, _ := strconv.ParseInt(m[1], 16, 0)
	stop, _ := strconv.ParseInt(m[2], 16, 0)
	for i := start; i <= stop; i++ {
		r = append(r, fmt.Sprintf("%x", i))
	}
	return r
}

func genPlainCharset(charset Charset) Charset {
	c := Charset{}
	// unique the ranged charset string to reduce calculation
	slice.Unique(&charset)
	for _, char := range charset {
		if strings.Contains(char, "-") {
			for _, r := range parseCharsetRange(char) {
				c = append(c, r)
			}
		} else {
			c = append(c, char)
		}
	}
	return c
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

//parseCJKSupport if lang supports CJK and what CJK it supports
func parseCJKSupport(langs []string) []string {
	cjk := []string{"ja", "ko", "zh-cn", "zh-sg", "zh-tw", "zh-mo", "zh-hk", "zh"}
	out := []string{}
	for _, lang := range langs {
		if b, _ := slice.Contains(cjk, lang); b {
			out = append(out, lang)
		}
	}
	if len(out) > 0 {
		return out
	}
	return []string{"none"}
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

func parseSpacing(data string) int {
	reSpacing := regexp.MustCompile(`spacing: ([^(]+)`)
	m := reSpacing.FindStringSubmatch(data)
	if len(m) > 0 {
		spacing, _ := strconv.Atoi(m[1])
		return spacing
	}
	return -1
}

func parseOutline(data string) bool {
	reOutline := regexp.MustCompile(`outline: ([^(]+)`)
	m := reOutline.FindStringSubmatch(data)
	outline, _ := strconv.ParseBool(m[1])
	return outline
}

//isDual find spacing=dual/mono/charcell
func isDual(spacing int, outline bool) int {
	if spacing > 90 && !outline {
		return 1
	}
	if spacing == 90 {
		return 0
	}
	return -1
}

//LoadFonts Incrementally load global and local fonts.
func LoadFonts(c Collection) Collection {
	newC := c
	// Existing collection
	pathsExisting := c.GetFontPaths()
	// Installed Fonts
	fontsInstalled := GetFontsInstalled()

	tmp := fontsInstalled
	slice.Intersect(&tmp, pathsExisting)

	// Get those not in pathsExisting
	slice.Remove(&fontsInstalled, tmp)

	wg := sync.WaitGroup{}
	wg.Add(len(fontsInstalled))
	mux := sync.Mutex{}
	ch := make(chan struct{}, 100) // ch is a chan to avoid "too many open files" when os exec

	for _, font := range fontsInstalled {
		log.Printf("Parsing %s...", font)
		go func(path string) {
			defer wg.Done()
			defer func() { <-ch }() // release chan
			ch <- struct{}{}        // acquire chan
			mux.Lock()
			slice.Concat(&newC, NewCollection(path))
			mux.Unlock()
		}(font)
	}

	wg.Wait()

	return newC
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
	return "", fmt.Errorf("no matched path found")
}

//GetFontsInstalled Get all font files installed on your system
func GetFontsInstalled() []string {
	local := filepath.Join(GetEnv("HOME"), ".fonts")
	candidates := []string{}

	for _, dir := range []string{local, "/usr/share/fonts"} {
		fonts, _ := dirutils.Ls(dir)
		for _, font := range fonts {
			if fileutils.HasPrefixOrSuffix(filepath.Base(font), "font", ".", ".dir", ".afm", ".gz", ".rpmsave") == 0 {
				candidates = append(candidates, font)
			}
		}
	}
	return candidates
}
