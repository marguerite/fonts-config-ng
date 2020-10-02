package lib

import (
	"math"
	"sort"
	"strconv"
	"strings"
)

// Charset collection of CharsetRange
type Charset []CharsetRange

func (c Charset) Len() int {
	return len(c)
}

func (c Charset) Less(i, j int) bool {
	return c[i].Max < c[j].Min
}

func (c Charset) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

// Append add c1 to c. ignore existing one
func (c *Charset) Append(c1 CharsetRange) {
	found := false
	for i, v := range *c {
		if v.Min == c1.Min && v.Max == c1.Max {
			found = true
			break
		}
		if c1.Min-v.Max == 1 {
			tmp := append((*c)[:i], CharsetRange{v.Min, c1.Max, int(c1.Max - v.Min + 1)})
			*c = append(tmp, (*c)[i+1:]...)
			found = true
			break
		}
		if v.Min-c1.Max == 1 {
			tmp := append((*c)[:i], CharsetRange{c1.Min, v.Max, int(v.Max - c1.Min + 1)})
			*c = append(tmp, (*c)[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		*c = append(*c, c1)
	}
}

// Intersect common value both in c and c1
func (c Charset) Intersect(c1 Charset) Charset {
	c2 := Charset{}
	for _, v := range c {
		for _, v1 := range c1 {
			v2, ok := v.Intersect(v1)
			if ok {
				c2 = append(c2, v2)
			}
		}
	}
	return c2
}

// Union merge c and c1 to c2
func (c Charset) Union(c1 Charset) (c4 Charset) {
	c2 := c.Substract(c1)
	c3 := c1

	for _, v := range c2 {
		c3.Append(v)
	}

	sort.Sort(c3)

	for i := 0; i < len(c3); i++ {
		if i == len(c3)-1 {
			c4.Append(c3[i])
			break
		}
		stop := c3[i].Max
		jump := 0
		for j := i + 1; j < len(c3); j++ {
			if c3[j].Min-c3[j-1].Max == 1 {
				stop = c3[j].Max
				jump++
			} else {
				break
			}
		}
		c4.Append(CharsetRange{c3[i].Min, stop, int(stop - c3[i].Min + 1)})
		i += jump
	}

	return c4
}

// Substract substract the CharsetRanges in c1 from c
func (c Charset) Substract(c1 Charset) (c2 Charset) {
	for _, v := range c {
		idx := 0
		for i, v1 := range c1 {
			val := v.Spaceship(v1)
			switch val {
			case 0, -0.5:
				continue
			case 1:
				r := CharsetRange{v.Min, v1.Min - 1, int(v1.Min - v.Min)}
				tc := Charset([]CharsetRange{r}).Substract(c1[:i])
				if len(tc) > 0 {
					for _, v2 := range tc {
						c2.Append(v2)
					}
				} else {
					c2.Append(r)
				}
			case -1:
				r := CharsetRange{v1.Max + 1, v.Max, int(v.Max - v1.Max)}
				tc := Charset([]CharsetRange{r}).Substract(c1[i+1:])
				if len(tc) > 0 {
					for _, v2 := range tc {
						c2.Append(v2)
					}
				} else {
					c2.Append(r)
				}
				// two value case
			case 0.5:
				if v.Min == v1.Min {
					r := CharsetRange{v1.Max + 1, v.Max, int(v.Max - v1.Max)}
					tc := Charset([]CharsetRange{r}).Substract(c1[i+1:])
					if len(tc) > 0 {
						for _, v2 := range tc {
							c2.Append(v2)
						}
					} else {
						c2.Append(r)
					}
					continue
				}
				if v.Max == v1.Max {
					r := CharsetRange{v.Min, v1.Min - 1, int(v1.Min - v.Min)}
					tc := Charset([]CharsetRange{r}).Substract(c1[:i])
					if len(tc) > 0 {
						for _, v2 := range tc {
							c2.Append(v2)
						}
					} else {
						c2.Append(r)
					}
					continue
				}
				r := CharsetRange{v1.Max + 1, v.Max, int(v.Max - v1.Max)}
				tc := Charset([]CharsetRange{r}).Substract(c1[i+1:])
				if len(tc) > 0 {
					for _, v2 := range tc {
						c2.Append(v2)
					}
				} else {
					c2.Append(r)
				}
				r1 := CharsetRange{v.Min, v1.Min - 1, int(v1.Min - v.Min)}
				tc1 := Charset([]CharsetRange{r1}).Substract(c1[:i])
				if len(tc1) > 0 {
					for _, v2 := range tc1 {
						c2.Append(v2)
					}
				} else {
					c2.Append(r1)
				}
			default:
				idx++
			}
			if idx == len(c1) {
				c2.Append(v)
			}
		}
	}
	sort.Sort(c2)
	return c2
}

// String echo the Charset as string
func (c Charset) String() string {
	str := ""
	for _, v := range c {
		if v.Len == 1 {
			str += strconv.FormatUint(v.Min, 16) + " "
			continue
		}
		str += strconv.FormatUint(v.Min, 16) + "-" + strconv.FormatUint(v.Max, 16) + " "
	}
	return str
}

// CharsetRange like 1f9a3-1f9cb
type CharsetRange struct {
	Min uint64
	Max uint64
	Len int
}

// Spaceship the spaceship operator
// -2.0 means c1 is outside of c and is lower than c
// 2.0 means c1 is outside of c and is greater than c
// -1.0 means c1 intersects the lower side of c
// 1.0 means c1 intersects the greater side of c
// 0.5 means c1 is in c
// -0.5 means c1 fully covers c
// 0 means c1 is equal to c
func (c CharsetRange) Spaceship(c1 CharsetRange) float64 {
	if c.Min > c1.Max {
		return -2.0
	}
	if c.Max < c1.Min {
		return 2.0
	}
	if c.Min == c1.Min && c.Max == c1.Max {
		return 0.0
	}
	// inside
	if c.Min < c1.Min && c.Max > c1.Max {
		return 0.5
	}
	// covered
	if c.Min >= c1.Min && c.Max <= c1.Max {
		return -0.5
	}
	// left intersect
	if c.Max > c1.Max {
		return -1.0
	}
	// right intersect
	return 1.0
}

// Intersect get the intersection of two CharsetRanges
func (c CharsetRange) Intersect(c1 CharsetRange) (CharsetRange, bool) {
	val := c.Spaceship(c1)
	if val == 0.0 || val == -0.5 {
		return c, true
	}
	if math.Abs(val) > 1.0 {
		return CharsetRange{uint64(0), uint64(0), 0}, false
	}
	min := c.Min
	max := c.Max
	if c1.Min > min {
		min = c1.Min
	}
	if c1.Max < max {
		max = c1.Max
	}
	len := int(max - min + 1)
	return CharsetRange{min, max, len}, true
}

// NewCharset initialize a Charset from string
func NewCharset(in string) (charset Charset) {
	for _, i := range strings.Split(in, " ") {
		if strings.Contains(i, "-") {
			arr := strings.Split(i, "-")
			a, _ := strconv.ParseUint(arr[0], 16, 32)
			b, _ := strconv.ParseUint(arr[1], 16, 32)
			charset = append(charset, CharsetRange{a, b, int(b - a + 1)})
			continue
		}
		j, _ := strconv.ParseUint(i, 16, 32)
		charset = append(charset, CharsetRange{j, j, 1})
	}
	return charset
}
