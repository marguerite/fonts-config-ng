package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	ft "github.com/marguerite/fonts-config-ng/font"
	dirutils "github.com/marguerite/go-stdlib/dir"
)

var (
	DEFAULT_JAVA_FONTS = map[string][]string{
		"SANS_JAPANESE":             {"/usr/share/fonts/truetype/sazanami-gothic.ttf", "-misc-sazanami gothic-"},
		"SERIF_JAPANESE":            {"/usr/share/fonts/truetype/sazanami-mincho.ttf", "-misc-sazanami mincho-"},
		"MONO_JAPANESE":             {"/usr/share/fonts/truetype/sazanami-gothic.ttf", "-misc-sazanami gothic-"},
		"SANS_SIMPLIFIED_CHINESE":   {"/usr/share/fonts/truetype/gbsn001p.ttf", "-arphic-ar pl sungtil gb-"},
		"SERIF_SIMPLIFIED_CHINESE":  {"/usr/share/fonts/truetype/gbsn001p.ttf", "-arphic-ar pl sungtil gb-"},
		"SANS_TRADITIONAL_CHINESE":  {"/usr/share/fonts/truetype/bsmi001p.ttf", "-arphic-ar pl mingti2l big5-"},
		"SERIF_TRADITIONAL_CHINESE": {"/usr/share/fonts/truetype/bsmi001p.ttf", "-arphic-ar pl mingti2l big5-"},
		"SANS_KOREAN":               {"/usr/share/fonts/truetype/dotum.ttf", "-baekmukttf-dotum-"},
		"SERIF_KOREAN":              {"/usr/share/fonts/truetype/batang.ttf", "-baekmukttf-batang-"},
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

	DEFAULT_JAVA_FPLISTS = FamilyPreferLists{
		NewFamilyPreferList("SANS_JAPANESE", "MS Gothic", "HGGothicB", "IPAPGothic", "IPAexGothic", "Sazanami Gothic"),
		NewFamilyPreferList("SERIF_JAPANESE", "MS Mincho", "HGMinchoL", "IPAPMincho", "IPAexMincho", "Sazanami Mincho"),
		NewFamilyPreferList("MONO_JAPANESE", "MS Gothic", "HGGothicB", "IPAGothic", "Sazanamii Gothic"),
		NewFamilyPreferList("SANS_SIMPLIFIED_CHINESE", "Noto Sans SC:style=Regular:weight=80", "FZsongTi"),
		NewFamilyPreferList("SERIF_SIMPLIFIED_CHINESE", "Noto Serif SC:style=Regular:weight=80", "FZsongTi"),
		NewFamilyPreferList("SANS_TRADITIONAL_CHINESE", "Noto Sans TC:style=Regular:weight=80", "AR PL ShanHeiSun Uni", "FZMingTiB", "AR PL Mingti2L Big5"),
		NewFamilyPreferList("SERIF_TRADITIONAL_CHINESE", "Noto Serif TC:style=Regular:weight=80", "AR PL ShanHeiSun Uni", "FZMingTiB", "AR PL Mingti2L Big5"),
		NewFamilyPreferList("SANS_KOREAN", "Noto Sans KR:style=Regular:weight=80", "UnDotum", "Baekmuk Gulim", "Baekmuk Dotum"),
		NewFamilyPreferList("SERIF_KOREAN", "Noto Serif KR:style=Regular:weight=80", "UnBatang", "Baekmuk Batang"),
		NewFamilyPreferList("SANS_LATIN1", "DejaVu Sans:style=Book:width=100", "Liberation Sans:style=Regular", "Droid Sans:style=Regular"),
		NewFamilyPreferList("SERIF_LATIN1", "DejaVu Serif:style=Book:width=100", "Liberation Serif:style=Regular", "Droid Serif:style=Regular"),
		NewFamilyPreferList("MONO_LATIN1", "DejaVu Sans Mono:style=Book", "Liberation Mono:style=Regular", "Droid Sans Mono:style=Regular"),
	}
)

type Java_XLFD struct {
	XLFD    string
	NoSpace string
	File    string
}

func selectJavaFonts(c ft.Collection) map[string]Java_XLFD {
	re := regexp.MustCompile(`([^:]+)(:.*)?$`)
	m := make(map[string]Java_XLFD)
	for _, v := range DEFAULT_JAVA_FPLISTS {
		var found bool
		for _, v1 := range v.List {
			c1 := c.FindByName(re.ReplaceAllString(v1.Item, `$1`))
			if len(c1) > 0 {
				found = true
				xlfd := getJavaXLFD(c1[0].Name[0])
				m[v.GenericName] = Java_XLFD{xlfd, strings.ReplaceAll(xlfd, " ", "_"), c1[0].File}
				break
			}
		}
		if !found {
			if val, ok := DEFAULT_JAVA_FONTS[v.GenericName]; ok {
				m[v.GenericName] = Java_XLFD{val[1], strings.ReplaceAll(val[1], " ", "_"), val[0]}
			}
		}
	}

	m["X11FONTDIR"] = Java_XLFD{"", "", "/usr/share/fonts/truetype"}
	return m
}

func getJavaXLFD(name string) string {
	if str, ok := DEFAULT_JAVA_XLFDS[name]; ok {
		return str
	}
	return "-misc-" + strings.ToLower(name) + "-"
}

// GenerateJavaFontSetup generates fontconfig properties conf for java
func GenerateJavaFontSetup(c ft.Collection) error {
	Dbg(cfg.Int("VERBOSITY"), Verbose, "Generating java font setup ...\n")

	tmpl, err := template.ParseFiles("/usr/share/fonts-config/fontconfig.SUSE.properties.template")
	if err != nil {
		panic(err)
	}

	fonts := selectJavaFonts(c)

	Dbg(cfg.Int("VERBOSITY"), Debug, func(fonts map[string]Java_XLFD) string {
		var str string
		for k, v := range fonts {
			str += fmt.Sprintf("%s_file=%s\n", k, v.File)
			str += fmt.Sprintf("%s_xlfd=%s\n", k, v.XLFD)
			str += fmt.Sprintf("%s_xlfd_no_space=%s\n", k, v.NoSpace)
		}
		return str
	}, fonts)

	paths, err := dirutils.Glob("/usr/lib*/jvm/*/jre/lib")
	if err != nil {
		return err
	}

	for _, path := range paths {
		f, err := os.OpenFile(filepath.Join(path, "fontconfig.SUSE.properties"), os.O_RDWR|os.O_CREATE, 0755)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		err = tmpl.Execute(f, fonts)
		if err != nil {
			panic(err)
		}
	}

	return nil
}
