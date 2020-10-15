package lib

import (
	"bufio"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	dirutils "github.com/marguerite/util/dir"
	ft "github.com/openSUSE/fonts-config/font"
	"github.com/openSUSE/fonts-config/sysconfig"
)

var (
	DEFAULT_JAVA_FONTS = map[string][]string{
		"SANS_JAPANESE":             []string{"/usr/share/fonts/truetype/sazanami-gothic.ttf", "-misc-sazanami gothic-"},
		"SERIF_JAPANESE":            []string{"/usr/share/fonts/truetype/sazanami-mincho.ttf", "-misc-sazanami mincho-"},
		"MONO_JAPANESE":             []string{"/usr/share/fonts/truetype/sazanami-gothic.ttf", "-misc-sazanami gothic-"},
		"SANS_SIMPLIFIED_CHINESE":   []string{"/usr/share/fonts/truetype/gbsn001p.ttf", "-arphic-ar pl sungtil gb-"},
		"SERIF_SIMPLIFIED_CHINESE":  []string{"/usr/share/fonts/truetype/gbsn001p.ttf", "-arphic-ar pl sungtil gb-"},
		"SANS_TRADITIONAL_CHINESE":  []string{"/usr/share/fonts/truetype/bsmi001p.ttf", "-arphic-ar pl mingti2l big5-"},
		"SERIF_TRADITIONAL_CHINESE": []string{"/usr/share/fonts/truetype/bsmi001p.ttf", "-arphic-ar pl mingti2l big5-"},
		"SANS_KOREAN":               []string{"/usr/share/fonts/truetype/dotum.ttf", "-baekmukttf-dotum-"},
		"SERIF_KOREAN":              []string{"/usr/share/fonts/truetype/batang.ttf", "-baekmukttf-batang-"},
	}

	DEFAULT_JAVA_XLFDS = map[string]string{"MS Gothic": "-ricoh-ms gothic-",
		"HGGothicB":            "-ricoh-hggothicb-",
		"IPAGothic":            "-misc-ipagothic-",
		"IPAPGothic":           "-misc-ipapgothic-",
		"IPAexGothic":          "-misc-ipaexgothic-",
		"Sazanami Gothic":      "-misc-sazanami gothic-",
		"MS Mincho":            "-ricoh-ms mincho-",
		"HGMinchoL":            "-ricoh-hgminchol-",
		"IPAMincho":            "-misc-ipamincho-",
		"IPAPMincho":           "-misc-ipapmincho-",
		"IPAexMincho":          "-misc-ipaexmincho-",
		"Sazanami Mincho":      "-misc-sazanami mincho-",
		"FZSongTi":             "-*-SongTi-",
		"FZMingTiB":            "-*-MingTiB-",
		"AR PL ShanHeiSun Uni": "-*-ar pl shanheisun uni-",
		"AR PL SungtiL GB":     "-arphic-ar pl sungtil gb-",
		"AR PL Mingti2L Big5":  "-arphic-ar pl mingti2l big5-",
		"UnDotum":              "-misc-undotum-",
		"Baekmuk Gulim":        "-baekmukttf-gulim-",
		"Baekmuk Dotum":        "-baekmukttf-dotum-",
		"UnBatang":             "-misc-unbatang-",
		"Baekmuk Batang":       "-baekmukttf-batang-",
		"Noto Sans SC":         "-goog-noto sans sc-",
		"Noto Sans TC":         "-goog-noto sans tc-",
		"Noto Sans KR":         "-goog-noto sans kr-"}

	DEFAULT_JAVA_FPL = map[string][]string{
		"SANS_JAPANESE":             []string{"MS Gothic", "HGGothicB", "IPAPGothic", "IPAexGothic", "Sazanami Gothic"},
		"SERIF_JAPANESE":            []string{"MS Mincho", "HGMinchoL", "IPAPMincho", "IPAexMincho", "Sazanami Mincho"},
		"MONO_JAPANESE":             []string{"MS Gothic", "HGGothicB", "IPAGothic", "Sazanamii Gothic"},
		"SANS_SIMPLIFIED_CHINESE":   []string{"Noto Sans SC:style=Regular:weight=80", "FZsongTi", "AR PL ShanHeiSun Uni", "AR PL SungtiL GB"},
		"SERIF_SIMPLIFIED_CHINESE":  []string{"Noto Serif SC:style=Regular:weight=80", "FZsongTi", "AR PL ShanHeiSun Uni", "AR PL SungtiL GB"},
		"SANS_TRADITIONAL_CHINESE":  []string{"Noto Sans TC:style=Regular:weight=80", "AR PL ShanHeiSun Uni", "FZMingTiB", "AR PL Mingti2L Big5"},
		"SERIF_TRADITIONAL_CHINESE": []string{"Noto Serif TC:style=Regular:weight=80", "AR PL ShanHeiSun Uni", "FZMingTiB", "AR PL Mingti2L Big5"},
		"SANS_KOREAN":               []string{"Noto Sans KR:style=Regular:weight=80", "UnDotum", "Baekmuk Gulim", "Baekmuk Dotum"},
		"SERIF_KOREAN":              []string{"Noto Serif KR:style=Regular:weight=80", "UnBatang", "Baekmuk Batang"},
		"SANS_LATIN1":               []string{"DejaVu Sans:style=Book:width=100", "Liberation Sans:style=Regular", "Droid Sans:style=Regular"},
		"SERIF_LATIN1":              []string{"DejaVu Serif:style=Book:width=100", "Liberation Serif:style=Regular", "Droid Serif:style=Regular"},
		"MONO_LATIN1":               []string{"DejaVu Sans Mono:style=Book", "Liberation Mono:style=Regular", "Droid Sans Mono:style=Regular"},
	}

	XLFD_REGEX = regexp.MustCompile(`_(\w+_\w+(_\w+_)?)_XLFD_(\w+_\w+_)?`)
	FILE_REGEX = regexp.MustCompile(`_(\w+_\w+(_\w+_)?)_FILE_`)
)

func getJavaXLFD(name string) string {
	if str, ok := DEFAULT_JAVA_XLFDS[name]; ok {
		return str
	}
	return "-misc-" + strings.ToLower(name) + "-"
}

// GenerateJavaFontSetup generates fontconfig properties conf for java
func GenerateJavaFontSetup(c ft.Collection, cfg sysconfig.SysConfig) error {
	Dbg(cfg.Int("VERBOSITY"), Verbose, "Generating java font setup ...\n")

	tmpl := NewReader("/usr/share/fonts-config/fontconfig.SUSE.properties.template")

	fonts := make(map[string][]string)
	re := regexp.MustCompile(`([^:]+)(:.*)?$`)
	for k, v := range DEFAULT_JAVA_FPL {
		found := false
		for _, font := range v {
			c1 := c.FindByName(re.ReplaceAllString(font, `$1`))
			if len(c1) > 0 {
				fonts[k] = []string{c1[0].File, getJavaXLFD(c1[0].Name[0])}
				found = true
				break
			}
		}
		if !found {
			if val, ok := DEFAULT_JAVA_FONTS[k]; ok {
				fonts[k] = val
			}
		}
	}

	Dbg(cfg.Int("VERBOSITY"), Debug, func(fonts map[string][]string) string {
		var str string
		for k, v := range fonts {
			str += fmt.Sprintf("%s_file=%s\n", k, v[0])
			str += fmt.Sprintf("%s_xlfd=%s\n", k, v[1])
			str += fmt.Sprintf("%s_xlfd_no_space=%s\n", k, strings.ReplaceAll(v[1], " ", "_"))
		}
		return str
	}, fonts)

	scanner := bufio.NewScanner(tmpl)

	var text string

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			text += "\n"
		}
		if strings.Contains(line, "_XLFD_") {
			m := XLFD_REGEX.FindStringSubmatch(line)
			val, ok := fonts[m[1]]
			if !ok {
				Dbg(cfg.Int("VERBOSITY"), Debug, fmt.Sprintf("cannot find value for %s to replace.\n", m[0]))
				continue
			}
			if m[len(m)-1] == "NO_SPACE_" {
				line = strings.ReplaceAll(line, m[0], strings.ReplaceAll(val[1], " ", "_"))
			} else {
				line = strings.ReplaceAll(line, m[0], strings.ReplaceAll(val[1], " ", "_"))
			}
		}
		if strings.Contains(line, "_FILE_") {
			m := FILE_REGEX.FindStringSubmatch(line)
			val, ok := fonts[m[1]]
			if !ok {
				Dbg(cfg.Int("VERBOSITY"), Debug, fmt.Sprintf("cannot find value for %s to replace.\n", m[0]))
				continue
			}
			line = strings.ReplaceAll(line, m[0], val[0])
		}
		if strings.Contains(line, "_X11FONTDIR_") {
			line = strings.ReplaceAll(line, "_X11FONTDIR_", "/usr/share/fonts/truetype")
		}
		if len(line) != 0 {
			text += line + "\n"
		}
	}

	paths, err := dirutils.Glob("/usr/lib*/jvm/*/jre/lib")
	if err != nil {
		return err
	}

	for _, path := range paths {
		err := overwriteOrRemoveFile(filepath.Join(path, "fontconfig.SUSE.properties"), []byte(text))
		if err != nil {
			return err
		}
	}

	return nil
}
