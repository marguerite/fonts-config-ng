package lib

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/marguerite/util/slice"
)

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
	return genPlainCharset(strings.Split(font, " "))
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
