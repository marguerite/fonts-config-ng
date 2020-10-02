package lib

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/marguerite/util/slice"
)

// Options our fonts-config program's inner options, controls the generated fontconfig conf
type Options struct {
	Verbosity                                  int
	ForceHintstyle                             string
	ForceAutohint                              bool
	ForceBw                                    bool
	ForceBwMonospace                           bool
	UseLcdfilter                               string
	UseRgba                                    string
	UseEmbeddedBitmaps                         bool
	EmbeddedBitmapsLanguages                   string
	PreferSansFamilies                         string
	PreferSerifFamilies                        string
	PreferMonoFamilies                         string
	SearchMetricCompatible                     bool
	ForceFamilyPreferenceLists                 bool
	GenerateTtcapEntries                       bool
	GenerateJavaFontSetup                      bool
	ForceModifyDefaultFontSettingsInNextUpdate bool
}

// FindByName find an option's value through its name
func (opt Options) FindByName(name string) interface{} {
	vo := reflect.ValueOf(opt)
	v := vo.FieldByName(name)
	if v.IsValid() {
		switch v.Kind() {
		case reflect.Bool:
			return v.Bool()
		case reflect.Int:
			return v.Int()
		default:
			return v.String()
		}
	}
	return nil
}

// Bounce bounce Options as string
func (opt Options) Bounce() string {
	vo := reflect.ValueOf(opt)
	str := ""
	for i := 0; i < vo.NumField(); i++ {
		name := vo.Type().Field(i).Name
		value := vo.Field(i)
		str += fmt.Sprintf("%s=%v\n", name, value)
	}
	return str
}

// Merge two Options, []int indicates which option was modified.
func (opt *Options) Merge(dst Options, idx []int) {
	vs := reflect.ValueOf(opt)
	vd := reflect.ValueOf(dst)

	for i := 0; i < vd.NumField(); i++ {
		ok, _ := slice.Contains(idx, i)
		// not modified, skip
		if !ok {
			continue
		}
		name := vd.Type().Field(i).Name
		sv := reflect.Indirect(vs).FieldByName(name)
		if !reflect.DeepEqual(vd.Field(i).Interface(), sv.Interface()) {
			if sv.IsValid() && sv.CanSet() {
				switch sv.Kind() {
				case reflect.Int:
					sv.SetInt(vd.Field(i).Int())
				case reflect.Bool:
					sv.SetBool(vd.Field(i).Bool())
				default:
					sv.SetString(vd.Field(i).String())
				}
			}
		}
	}
}

// FillTemplate convert options to string with the help of template
func (opt Options) FillTemplate(f io.Reader) string {
	text := ""
	re := regexp.MustCompile(`^([^#]+\w)="(.*)"$`)

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		if re.MatchString(line) {
			m := re.FindStringSubmatch(line)
			ov := opt.FindByName(optionNameFromSysconfig(m[1]))
			str := ""

			if v, ok := ov.(string); ok {
				str = v
			}

			if v, ok := ov.(bool); ok {
				str = parseBool(v)
			}

			if v, ok := ov.(int64); ok {
				str = strconv.Itoa(int(v))
			}

			text += m[1] + "=\"" + str + "\"\n"
		} else {
			text += line + "\n"
		}
	}

	return text
}

// WriteOptions write options to file
func WriteOptions(f io.Writer, text string) {
	n, err := f.Write([]byte(text))
	if err != nil {
		log.Fatal(err)
	}

	if n != len(text) {
		log.Fatal("Failed to write all data, configuration may be broken or incomplete.")
	}
}

func formatBool(s string) bool {
	if strings.HasPrefix(strings.ToLower(s), "y") {
		return true
	}
	return false
}

func parseBool(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// NewOptions generate default Options
func NewOptions() Options {
	return Options{0, "", false, false, false, "", "", false,
		"", "", "", "", false, false,
		false, false, false}
}

func optionNameFromSysconfig(s string) string {
	re := regexp.MustCompile(`(_|^)[[:lower:]]`)
	s = strings.ToLower(s)
	m := re.FindAllString(s, -1)
	for _, char := range m {
		s = strings.Replace(s, char, strings.ToUpper(char), 1)
	}
	return strings.Replace(s, "_", "", -1)
}

// LoadOptions load options from config file
func LoadOptions(conf io.Reader, opts Options) Options {
	re := regexp.MustCompile(`^(.*)="(.*)"$`)
	reInlineComment := regexp.MustCompile(`([^#]*)#?.*`)

	scanner := bufio.NewScanner(conf)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "#") {
			if reInlineComment.MatchString(line) {
				line = reInlineComment.FindStringSubmatch(line)[1]
			}
			if re.MatchString(line) {
				m := re.FindStringSubmatch(line)
				v := reflect.ValueOf(&opts).Elem()
				f := v.FieldByName(optionNameFromSysconfig(m[1]))
				if f.IsValid() {
					if f.CanSet() {
						// skip empty value
						if len(m[2]) == 0 {
							continue
						}
						switch f.Kind() {
						case reflect.Int:
							i, _ := strconv.Atoi(m[2])
							f.SetInt(int64(i))
						case reflect.Bool:
							f.SetBool(formatBool(m[2]))
						default:
							f.SetString(m[2])
						}
					}
				} else {
					continue
				}
			}
		}
	}
	return opts
}
