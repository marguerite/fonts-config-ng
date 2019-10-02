package lib

import (
	"fmt"
	"github.com/marguerite/util/slice"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

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
