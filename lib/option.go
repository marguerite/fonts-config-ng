package lib

import (
	"fmt"
	"github.com/marguerite/util/fileutils"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// Options our fonts-config program's inner options, controls the generated fontconfig conf
type Options struct {
	Verbosity                  int
	ForceHintstyle             string
	ForceAutohint              bool
	ForceBw                    bool
	ForceBwMonospace           bool
	UseLcdfilter               string
	UseRgba                    string
	UseEmbeddedBitmaps         bool
	EmbeddedBitmapsLanguages   string
	SystemEmojis               string
	PreferSansFamilies         string
	PreferSerifFamilies        string
	PreferMonoFamilies         string
	SearchMetricCompatible     bool
	ForceFamilyPreferenceLists bool
	GenerateTtcapEntries       bool
	GenerateJavaFontSetup      bool
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

// Bounce Options to string
func (opt Options) Bounce() {
	vo := reflect.ValueOf(opt)
	str := ""
	for i := 0; i < vo.NumField(); i++ {
		name := vo.Type().Field(i).Name
		value := vo.Field(i)
		str += fmt.Sprintf("%s=%v\n", name, value)
	}
	fmt.Println(str)
}

// Merge two Options
func (opt *Options) Merge(dst Options) {
	vs := reflect.ValueOf(opt)
	vd := reflect.ValueOf(dst)

	for i := 0; i < vd.NumField(); i++ {
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

// Write Options to file
func (opt Options) Write(userMode bool) error {
	dst := "/etc/sysconfig/fonts-config"
	if userMode {
		dst = filepath.Join(GetEnv("HOME"), ".config/fontconfig/fonts-config")
	}

	text := ""
	re := regexp.MustCompile(`^([^#]+\w)="(.*)"$`)
	f, err := ioutil.ReadFile(dst)
	if err != nil {
		return err
	}

	for _, line := range strings.Split(string(f), "\n") {
		if re.MatchString(line) {
			m := re.FindStringSubmatch(line)
			ov := opt.FindByName(convertSysconfigEntryToOptionName(m[1]))
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

	// always make backup
	fileutils.Copy(dst, dst+".bak")

	err = ioutil.WriteFile(dst, []byte(text), 0644)
	if err != nil {
		return err
	}

	return nil
}

func formatBool(s string) bool {
	if s == "yes" {
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

// generatePresetOptions generate a default Options
func generatePresetOptions() Options {
	return Options{0, "", false, false, false, "", "", false,
		"", "", "", "", "", false, false,
		false, false}
}

func convertSysconfigEntryToOptionName(s string) string {
	re := regexp.MustCompile(`(_|^)[[:lower:]]`)
	s = strings.ToLower(s)
	m := re.FindAllString(s, -1)
	for _, char := range m {
		s = strings.Replace(s, char, strings.ToUpper(char), 1)
	}
	return strings.Replace(s, "_", "", -1)
}

// LoadOptions load options from config file
func LoadOptions(conf string, opts Options) Options {
	if opts == (Options{}) {
		opts = generatePresetOptions()
	}

	f, e := ioutil.ReadFile(conf)
	if e != nil {
		fmt.Printf("NOTE: %s doesn't exist, using builtin defaults.\n", conf)
		return opts
	}

	re := regexp.MustCompile(`^(.*)="(.*)"$`)
	reComment := regexp.MustCompile(`([^#]*)#?.*`)

	for _, line := range strings.Split(string(f), "\n") {
		if !strings.HasPrefix(line, "#") {
			if reComment.MatchString(line) {
				line = reComment.FindStringSubmatch(line)[1]
			}
			if re.MatchString(line) {
				m := re.FindStringSubmatch(line)
				v := reflect.ValueOf(&opts).Elem()
				f := v.FieldByName(convertSysconfigEntryToOptionName(m[1]))
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
