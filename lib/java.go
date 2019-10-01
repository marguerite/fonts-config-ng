package lib

import (
	"bufio"
	"fmt"
	"github.com/marguerite/util/command"
	"github.com/marguerite/util/fileutils"
	"github.com/marguerite/util/slice"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
)

// FontCandidates a struct containing family preference list for a generic font name for a specific CJK language
type FontCandidates struct {
	GenericName string
	Lang        string
	FPL         []string
}

// JavaFontProperty a struct containing font's path and XLFD
type JavaFontProperty struct {
	Path string
	XLFD string
}

// JavaFont a struct containing font's name and JavaFontProperty
type JavaFont struct {
	Name string
	JavaFontProperty
}

// JavaFonts a slice of JavaFont
type JavaFonts []JavaFont

// FindByName finds a JavaFont struct by its Name property
func (f JavaFonts) FindByName(name string) (JavaFont, bool) {
	if strings.HasPrefix(name, "_") {
		name = strings.TrimPrefix(name, "_")
	}

	if strings.HasSuffix(name, "_") {
		name = strings.TrimSuffix(name, "_")
	}

	for _, v := range f {
		if v.Name == name {
			return v, true
		}
	}
	return JavaFont{"", JavaFontProperty{"", ""}}, false
}

// generatePresetCJKJavaFonts generate a JavaFonts slice with openSUSE default CJK choices for Java
func getPresetJavaFonts() JavaFonts {
	return JavaFonts{
		JavaFont{"sans_japanese", JavaFontProperty{"/usr/share/fonts/truetype/sazanami-gothic.ttf", "-misc-sazanami gothic-"}},
		JavaFont{"serif_japanese", JavaFontProperty{"/usr/share/fonts/truetype/sazanami-mincho.ttf", "-misc-sazanami mincho-"}},
		JavaFont{"mono_japanese", JavaFontProperty{"/usr/share/fonts/truetype/sazanami-gothic.ttf", "-misc-sazanami gothic-"}},
		JavaFont{"sans_simplified_chinese", JavaFontProperty{"/usr/share/fonts/truetype/gbsn001p.ttf", "-arphic-ar pl sungtil gb-"}},
		JavaFont{"serif_simplified_chinese", JavaFontProperty{"/usr/share/fonts/truetype/gbsn001p.ttf", "-arphic-ar pl sungtil gb-"}},
		JavaFont{"sans_traditional_chinese", JavaFontProperty{"/usr/share/fonts/truetype/bsmi001p.ttf", "-arphic-ar pl mingti2l big5-"}},
		JavaFont{"serif_traditional_chinese", JavaFontProperty{"/usr/share/fonts/truetype/bsmi001p.ttf", "-arphic-ar pl mingti2l big5-"}},
		JavaFont{"sans_korean", JavaFontProperty{"/usr/share/fonts/truetype/dotum.ttf", "-baekmukttf-dotum-"}},
		JavaFont{"serif_korean", JavaFontProperty{"/usr/share/fonts/truetype/batang.ttf", "-baekmukttf-batang-"}},
	}
}

// trimEndingColon removes whitespaces and the ending colon in font path
func trimEndingColon(path string) string {
	path = strings.TrimSpace(path)
	path = strings.Replace(path, ":", "", -1)
	return path
}

func fclistArgToFontName(name string) string {
	re := regexp.MustCompile(`^(.*?):.*$`)
	if re.MatchString(name) {
		return re.FindStringSubmatch(name)[1]
	}
	return name
}

func getJavaXlfdByName(name string) string {
	xlfds := map[string]string{"MS Gothic": "-ricoh-ms gothic-",
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

	if str, ok := xlfds[name]; ok {
		return str
	}
	return "-misc-" + strings.ToLower(name) + "-"
}

func substituteSpaceInJavaXLFD(xlfd string) string {
	return strings.Replace(xlfd, " ", "_", -1)
}

// getInstalledFontNameAndPathFromList selects a font both in the lst and installed on your system for Java
// fc-list should return only one result otherwise last one is taken in present
func getInstalledFontNameAndPathFromList(lst FontCandidates, verbosity int) JavaFontProperty {
	fontfile := ""
	fontname := ""

	for _, font := range lst.FPL {
		if out, _, _ := command.Run("/usr/bin/fc-list", []string{font, "file"}, verbosity); len(out) != 0 {
			for _, f := range strings.Split(trimEndingColon(out), "\n") {
				if fileutils.HasPrefixOrSuffix(f, []string{".ttf", ".otf", ".ttc"}) {
					if info, _ := os.Stat(f); info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0 {
						fontfile = f
						fontname = fclistArgToFontName(font)
						break
					}
				}
			}
		}
	}

	if len(fontfile) == 0 {
		debug(verbosity, VerbosityDebug, fmt.Sprintf(" warning: cannot find a %s %s font, %s might not work in Java.\n", lst.Lang, lst.GenericName, lst.Lang))
		return JavaFontProperty{"", ""}
	}
	return JavaFontProperty{fontfile, getJavaXlfdByName(fontname)}
}

// overrideOrAppendJavaFont override existing JavaFont in JavaFonts or append to it.
func overrideOrAppendJavaFont(f *JavaFonts, genericFont JavaFont) {
	if len(genericFont.Path) > 0 {
		notFound := true
		v := reflect.ValueOf(f).Elem()
		for i := 0; i < v.Len(); i++ {
			if reflect.DeepEqual(v.Index(i).FieldByName("Name").Interface(), reflect.ValueOf(genericFont).FieldByName("Name").Interface()) {
				slice.Replace(f, v.Index(i), genericFont)
				notFound = false
				break
			}
		}
		if notFound {
			slice.Concat(f, genericFont)
		}
	}
}

// GenerateJavaFontSetup generates fontconfig properties conf for java
func GenerateJavaFontSetup(verbosity int) error {
	debug(verbosity, VerbosityVerbose, "generating java font setup ...\n")

	template := "/usr/share/fonts-config/fontconfig.SUSE.properties.template"

	sansJP := FontCandidates{"sans serif", "Japanese", []string{"MS Gothic", "HGGothicB", "IPAPGothic", "IPAexGothic", "Sazanami Gothic"}}
	serifJP := FontCandidates{"serif", "Japanese", []string{"MS Mincho", "HGMinchoL", "IPAPMincho", "IPAexMincho", "Sazanami Mincho"}}
	monoJP := FontCandidates{"monospace", "Japanese", []string{"MS Gothic", "HGGothicB", "IPAGothic", "Sazanamii Gothic"}}
	sansSC := FontCandidates{"sans serif", "Simplified Chinese", []string{"Noto Sans SC:style=Regular:weight=80", "FZsongTi", "AR PL ShanHeiSun Uni", "AR PL SungtiL GB"}}
	serifSC := FontCandidates{"serif", "Simplified Chinese", []string{"Noto Serif SC:style=Regular:weight=80", "FZsongTi", "AR PL ShanHeiSun Uni", "AR PL SungtiL GB"}}
	sansTC := FontCandidates{"sans serif", "Traditional Chinese", []string{"Noto Sans TC:style=Regular:weight=80", "AR PL ShanHeiSun Uni", "FZMingTiB", "AR PL Mingti2L Big5"}}
	serifTC := FontCandidates{"serif", "Traditional Chinese", []string{"Noto Serif TC:style=Regular:weight=80", "AR PL ShanHeiSun Uni", "FZMingTiB", "AR PL Mingti2L Big5"}}
	sansKR := FontCandidates{"sans serif", "Korean", []string{"Noto Sans KR:style=Regular:weight=80", "UnDotum", "Baekmuk Gulim", "Baekmuk Dotum"}}
	serifKR := FontCandidates{"serif", "Korean", []string{"Noto Serif KR:style=Regular:weight=80", "UnBatang", "Baekmuk Batang"}}
	sansLatin1 := FontCandidates{"sans serif", "Latin 1", []string{"DejaVu Sans:style=Book:width=100", "Liberation Sans:style=Regular", "Droid Sans:style=Regular"}}
	serifLatin1 := FontCandidates{"serif", "Latin 1", []string{"DejaVu Serif:style=Book:width=100", "Liberation Serif:style=Regular", "Droid Serif:style=Regular"}}
	monoLatin1 := FontCandidates{"monospace", "Latin 1", []string{"DejaVu Sans Mono:style=Book", "Liberation Mono:style=Regular", "Droid Sans Mono:style=Regular"}}

	fonts := getPresetJavaFonts()
	overrideOrAppendJavaFont(&fonts, JavaFont{"sans_japanese", getInstalledFontNameAndPathFromList(sansJP, verbosity)})
	overrideOrAppendJavaFont(&fonts, JavaFont{"serif_japanese", getInstalledFontNameAndPathFromList(serifJP, verbosity)})
	overrideOrAppendJavaFont(&fonts, JavaFont{"mono_japanese", getInstalledFontNameAndPathFromList(monoJP, verbosity)})
	overrideOrAppendJavaFont(&fonts, JavaFont{"sans_simplified_chinese", getInstalledFontNameAndPathFromList(sansSC, verbosity)})
	overrideOrAppendJavaFont(&fonts, JavaFont{"serif_simplified_chinese", getInstalledFontNameAndPathFromList(serifSC, verbosity)})
	overrideOrAppendJavaFont(&fonts, JavaFont{"sans_traditional_chinese", getInstalledFontNameAndPathFromList(sansTC, verbosity)})
	overrideOrAppendJavaFont(&fonts, JavaFont{"serif_traditional_chinese", getInstalledFontNameAndPathFromList(serifTC, verbosity)})
	overrideOrAppendJavaFont(&fonts, JavaFont{"sans_korean", getInstalledFontNameAndPathFromList(sansKR, verbosity)})
	overrideOrAppendJavaFont(&fonts, JavaFont{"serif_korean", getInstalledFontNameAndPathFromList(serifKR, verbosity)})
	overrideOrAppendJavaFont(&fonts, JavaFont{"sans_latin1", getInstalledFontNameAndPathFromList(sansLatin1, verbosity)})
	overrideOrAppendJavaFont(&fonts, JavaFont{"serif_latin1", getInstalledFontNameAndPathFromList(serifLatin1, verbosity)})
	overrideOrAppendJavaFont(&fonts, JavaFont{"mono_latin1", getInstalledFontNameAndPathFromList(monoLatin1, verbosity)})

	debugText := ""
	for _, v := range fonts {
		debugText += fmt.Sprintf("%s_file=%s\n", v.Name, v.Path)
		debugText += fmt.Sprintf("%s_xlfd=%s\n", v.Name, v.XLFD)
		debugText += fmt.Sprintf("%s_xlfd_no_space=%s\n", v.Name, substituteSpaceInJavaXLFD(v.XLFD))
	}
	debug(verbosity, VerbosityDebug, debugText)

	tmpl, err := os.Open(template)
	if err != nil {
		return err
	}
	defer tmpl.Close()

	scanner := bufio.NewScanner(tmpl)
	scanner.Split(bufio.ScanLines)
	xlfdRe := regexp.MustCompile(`_(\w+_\w+(_\w+_)?)_XLFD_(\w+_\w+_)?`)
	fileRe := regexp.MustCompile(`_(\w+_\w+(_\w+_)?)_FILE_`)

	javaText := ""

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			javaText += "\n"
		}
		if strings.Contains(line, "_XLFD_") {
			m := xlfdRe.FindStringSubmatch(line)
			font, ok := fonts.FindByName(strings.ToLower(m[1]))
			if !ok {
				debug(verbosity, VerbosityDebug, fmt.Sprintf("cannot find value for %s to replace.\n", m[0]))
				continue
			}
			if m[len(m)-1] == "NO_SPACE_" {
				line = strings.Replace(line, m[0], substituteSpaceInJavaXLFD(font.XLFD), -1)
			} else {
				line = strings.Replace(line, m[0], font.XLFD, -1)
			}
		}
		if strings.Contains(line, "_FILE_") {
			m := fileRe.FindStringSubmatch(line)
			font, ok := fonts.FindByName(strings.ToLower(m[1]))
			if !ok {
				debug(verbosity, VerbosityDebug, fmt.Sprintf("cannot find value for %s to replace.\n", m[0]))
				continue
			}
			line = strings.Replace(line, m[0], font.Path, -1)
		}
		if strings.Contains(line, "_X11FONTDIR_") {
			line = strings.Replace(line, "_X11FONTDIR_", "/usr/share/fonts/truetype", -1)
		}
		if len(line) != 0 {
			javaText += line + "\n"
		}
	}

	javaFiles, err := filepath.Glob("/usr/lib*/jvm/jre/lib/fontconfig.SUSE.properties")
	if err != nil {
		return err
	}

	for _, f := range javaFiles {
		err := persist(f, []byte(javaText), 0644)
		if err != nil {
			return err
		}
	}

	return nil
}
