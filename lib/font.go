package lib

import (
	"encoding/json"
	"fmt"
	"github.com/marguerite/util/slice"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Style font's width, weight and slant
type Style struct {
	Width  int
	Weight int
	Slant  int
}

// Load font width/weight/slant
func (s *Style) Load(fontFile string) {
	raw, _ := exec.Command("/usr/bin/fc-scan", fontFile).Output()
	widthR := regexp.MustCompile(`(?m)width:\s(\d+)`)
	weightR := regexp.MustCompile(`(?m)weight:\s(\d+)`)
	slantR := regexp.MustCompile(`(?m)slant:\s(\d+)`)

	var width, weight, slant int
	if widthR.MatchString(string(raw)) {
		width, _ = strconv.Atoi(widthR.FindStringSubmatch(string(raw))[1])
		s.Width = width
	}
	if weightR.MatchString(string(raw)) {
		weight, _ = strconv.Atoi(weightR.FindStringSubmatch(string(raw))[1])
		s.Weight = weight
	}
	if slantR.MatchString(string(raw)) {
		slant, _ = strconv.Atoi(slantR.FindStringSubmatch(string(raw))[1])
		s.Slant = slant
	}
}

// Font struct contains various font informations.
type Font struct {
	Name    []string
	Lang    []string
	Hinting bool
}

// Collection a collection of Font bundled in one rpm.
type Collection []Font

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

// Charset font charset
type Charset []string

func (c Charset) Len() int { return len(c) }

func (c Charset) Swap(i, j int) { c[i], c[j] = c[j], c[i] }

func (c Charset) Less(i, j int) bool {
	a, _ := strconv.ParseInt(c[i], 16, 0)
	b, _ := strconv.ParseInt(c[j], 16, 0)
	return a < b
}

// EnhancedFont with Charset and Style
type EnhancedFont struct {
	Font
	Charset
	Style
}

// AppendCharset append charset to an existing EnhancedFont
func (e *EnhancedFont) AppendCharset(c Charset) {
	c1 := e.Charset
	slice.Concat(&c1, c)
	e.Charset = c1
}

// EnhancedFonts a slice of EnhancedFont
type EnhancedFonts []EnhancedFont

// Contains if an element of the EnhancedFonts has the same name and style with the provided EnhancedFont
func (e EnhancedFonts) Contains(f EnhancedFont) (int, bool) {
	fv := reflect.ValueOf(f)
	for i, j := range e {
		if reflect.DeepEqual(reflect.ValueOf(j).FieldByName("Name").Interface(), fv.FieldByName("Name").Interface()) &&
			reflect.DeepEqual(reflect.ValueOf(j).FieldByName("Style").Interface(), fv.FieldByName("Style").Interface()) {
			return i, true
		}
	}
	return 0, false
}

// fix AR PL UMing/AR PL UKai fonts
func fixARPLFont(s *string) {
	if strings.HasPrefix(*s, "AR PL") {
		n := ""
		for _, v := range strings.Split(*s, "AR PL") {
			if len(v) > 0 {
				n += "AR PL" + v + ","
			}
		}
		n = strings.TrimRight(n, ",")
		*s = n
	}
}

// fix WQY Zen Hei/WQY Micro Hei fonts
func fixWQYFont(s *string) {
	if strings.HasPrefix(*s, "文泉") {
		n := ""
		re := regexp.MustCompile(`文泉[^文泉]+`)
		for _, v := range strings.Split(*s, ",") {
			m := re.FindAllStringSubmatch(v, -1)
			if len(m) > 1 {
				for _, i := range m {
					n += i[0] + ","
				}
			} else {
				n += v + ","
			}
		}
		n = strings.TrimRight(n, ",")
		*s = n
	}
}

func fixFontName(s *string) {
	fixARPLFont(s)
	fixWQYFont(s)
}

// GetFontName get font name
func GetFontName(fontFile string) []string {
	out, _ := exec.Command("/usr/bin/fc-scan", "--format", "%{family}", fontFile).Output()
	names := string(out)
	fixFontName(&names)
	if strings.Contains(names, ",") {
		s := strings.Split(names, ",")
		// strip Regular/Book
		for i := 1; i < len(s); i++ {
			if strings.HasSuffix(s[i], "Regular") || strings.HasSuffix(s[i], "Book") {
				s = append(s[:i], s[i+1:]...)
			}
		}
		return s
	}
	return []string{names}
}

// GetFontLang get font languages
func GetFontLang(fontFile string) []string {
	langs, _ := exec.Command("/usr/bin/fc-scan", "--format", "%{lang}", fontFile).Output()
	if strings.Contains(string(langs), "|") {
		s := strings.Split(string(langs), "|")
		return s
	}
	return []string{string(langs)}
}

// ParseFontInfoFromFile read various font infos with fc-scan
func ParseFontInfoFromFile(ttf string) Font {
	ok, _ := Hinting(ttf)
	return Font{GetFontName(ttf), GetFontLang(ttf), ok}
}

// GenericFamily find generic name through font name
func GenericFamily(fontName string) string {
	if strings.Contains(fontName, " Symbols") {
		return "symbol"
	}
	if strings.Contains(fontName, " Mono") || strings.Contains(fontName, " HW") {
		return "monospace"
	}
	if strings.HasSuffix(fontName, "Emoji") {
		return "emoji"
	}
	if strings.Contains(fontName, " Serif") {
		return "serif"
	}
	return "sans-serif"
}

// GetUnstyledFontName pick unstyled font names
func GetUnstyledFontName(f Font) []string {
	names := f.Name
	s, _ := slice.ShortestString(names)
	slice.Remove(&names, s)
	// trim "Noto Sans Display UI"
	if strings.HasSuffix(s, "UI") {
		s = strings.TrimRight(s, " UI")
	}
	out := []string{s}
	for _, n := range names {
		if !strings.Contains(n, s) {
			out = append(out, n)
		}
	}

	return out
}

// GenerateDefaultFamily return a default family fontconfig block
func GenerateDefaultFamily(fontName string) string {
	return "\t<alias>\n\t\t<family>" + fontName + "</family>\n\t\t<default>\n\t\t\t<family>" +
		GenericFamily(fontName) + "</family>\n\t\t</default>\n\t</alias>\n\n"
}

func generateFontTypeByHinting(fontName string, hinting bool) string {
	txt := "\t<match target=\"font\">\n\t\t<test name=\"family\">\n\t\t\t<string>" + fontName + "</string>\n\t\t</test>\n"
	txt += "\t\t<edit name=\"font_type\" mode=\"assign\">\n\t\t\t<string>"
	if hinting {
		txt += "TT Instructed Font"
	} else {
		txt += "NON TT Instructed Font"
	}
	txt += "</string>\n\t\t</edit>\n\t</match>\n\n"
	return txt
}

// GenerateFontTypeByHinting generate font_type block based on hinting
func GenerateFontTypeByHinting(f Font) string {
	if len(f.Name) > 1 {
		txt := ""
		for _, v := range f.Name {
			txt += generateFontTypeByHinting(v, f.Hinting)
		}
		return txt
	}
	return generateFontTypeByHinting(f.Name[0], f.Hinting)
}

// GenerateFamilyPreferListForLang generate family preference list of fonts for a generic font name
// and a specific language
func GenerateFamilyPreferListForLang(generic, lang string, fonts []string) string {
	txt := "\t<match>\n\t\t<test name=\"family\">\n\t\t\t<string>" + generic + "</string>\n\t\t</test>\n"
	txt += "\t\t<test name=\"lang\">\n\t\t\t<string>" + lang + "</string>\n\t\t</test>\n"
	txt += "\t\t<edit name=\"family\" mode=\"prepend\">\n"
	for _, f := range fonts {
		txt += "\t\t\t<string>" + f + "</string>\n"
	}
	txt += "\t\t</edit>\n\t</match>\n\n"
	return txt
}

func extractCharsetRange(s string) []string {
	r := []string{}
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

func createPlainCharset(charset []string) Charset {
	c := Charset{}
	// unique the ranged charset string to reduce calculation
	slice.Unique(&charset)
	for _, char := range charset {
		if strings.Contains(char, "-") {
			for _, r := range extractCharsetRange(char) {
				c = append(c, r)
			}
		} else {
			c = append(c, char)
		}
	}
	return c
}

func substractChar(c1, c2 string) int64 {
	a, _ := strconv.ParseInt(c1, 16, 0)
	b, _ := strconv.ParseInt(c2, 16, 0)
	return a - b
}

// BuildCharset build charset array of a font
func BuildCharset(f string) Charset {
	if _, err := os.Stat(f); !os.IsNotExist(err) {
		out, e := exec.Command("/usr/bin/fc-scan", "--format", "%{charset}", f).Output()
		ErrChk(e)
		return createPlainCharset(strings.Split(string(out), " "))
	}
	return Charset{}
}

// BuildEmojiCharset build charset array of a emoji font
func BuildEmojiCharset(f []string) Charset {
	charset := Charset{}

	for _, v := range f {
		slice.Concat(&charset, BuildCharset(v))
	}

	slice.Unique(&charset)

	/* common emojis that almost every font has
	   "#","*","0","1","2","3","4","5","6","7","8","9","©","®","™"," ",
	   "‼","↔","↕","↖","↗","↘","↙","▪","▫","☀","⁉","ℹ",
	   "▶","◀","☑","↩","↪","➡","⬅","⬆","⬇","♀","♂" */
	commonEmoji := Charset{"0", "20", "23", "2a", "30", "31", "32", "33", "34", "35", "36", "37",
		"38", "39", "a9", "ae", "200d", "203c", "2049", "20e3", "2122",
		"2139", "2194", "2195", "2196", "2197", "2198", "2199", "21a9",
		"21aa", "25aa", "25ab", "25b6", "25c0", "2600", "2611", "2640",
		"2642", "27a1", "2b05", "2b06", "2b07"}
	slice.Remove(&charset, commonEmoji)
	return charset
}

// IntersectCharset build intersected charset array of two fonts
func IntersectCharset(charset, emoji Charset) Charset {
	in := charset
	slice.Intersect(&in, emoji)
	sort.Sort(in)
	return RangedCharset(in)
}

// RangedCharset convert int to range in charset
func RangedCharset(c Charset) Charset {
	if len(c) < 2 {
		return c
	}

	charset := Charset{}

	sort.Sort(c)
	idx := -1

	for i := 0; i < len(c); i++ {
		if i == len(c)-1 {
			if idx >= 0 && substractChar(c[i], c[i-1]) == 1 {
				charset = append(charset, c[idx]+".."+c[i])
			} else {
				charset = append(charset, c[i])
			}
			continue
		}

		if substractChar(c[i+1], c[i]) == 1 {
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

// ConcatCharset concat two charsets into one
func ConcatCharset(c1, c2 Charset) Charset {
	c1 = createPlainCharset(c1)
	c2 = createPlainCharset(c2)
	slice.Concat(&c1, c2)
	return RangedCharset(c1)
}

// CharsetToFontConfig convert Charset to fontconfig conf
func CharsetToFontConfig(c Charset) string {
	str := "\t\t\t\t<charset>\n"
	for _, v := range c {
		if strings.Contains(v, "..") {
			str += "\t\t\t\t\t<range>\n"
			for _, s := range strings.Split(v, "..") {
				str += "\t\t\t\t\t\t<int>0x" + s + "</int>\n"
			}
			str += "\t\t\t\t\t</range>\n"
		} else {
			str += "\t\t\t\t\t<int>0x" + v + "</int>\n"
		}
	}
	str += "\t\t\t\t</charset>\n"
	return str
}
