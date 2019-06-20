package lib

import (
	"fmt"
	"github.com/marguerite/util/dirutils"
	"github.com/marguerite/util/fileutils"
	"github.com/marguerite/util/slice"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// ReadFontFilesFromDir read font files from specific dir
func ReadFontFilesFromDir(d string, emoji bool) []string {
	files, _ := dirutils.Ls(d, "file")
	fonts := []string{}

	for _, f := range files {
		file := filepath.Base(f)
		if fileutils.HasPrefixSuffixInGroup(file, []string{"fonts", "."}, true) || strings.HasSuffix(file, ".dir") {
			continue
		}
		if emoji && strings.Contains(file, "Emoji") {
			fonts = append(fonts, f)
		}
		if !emoji && !strings.Contains(file, "Emoji") {
			fonts = append(fonts, f)
		}
	}
	return fonts
}

func generateBlacklistConfig(f EnhancedFont) string {
	conf := "\t<match target=\"scan\">\n\t\t<test name=\"family\">\n\t\t\t<string>" + f.Name[0] + "</string>\n\t\t</test>\n"
	if !(f.Width == 0 && f.Weight == 0 && f.Slant == 0) {
		if f.Width != 100 {
			conf += "\t\t<test name=\"width\">\n\t\t\t<int>" + strconv.Itoa(f.Width) + "</int>\n\t\t</test>\n"
		}
		if f.Weight != 80 {
			conf += "\t\t<test name=\"weight\">\n\t\t\t<int>" + strconv.Itoa(f.Weight) + "</int>\n\t\t</test>\n"
		}
		if f.Slant != 0 {
			conf += "\t\t<test name=\"slant\">\n\t\t\t<int>" + strconv.Itoa(f.Slant) + "</int>\n\t\t</test>\n"
		}
	}
	conf += "\t\t<edit name=\"charset\" mode=\"assign\">\n\t\t\t<minus>\n\t\t\t\t<name>charset</name>\n"
	conf += CharsetToFontConfig(f.Charset)
	conf += "\t\t\t</minus>\n\t\t</edit>\n\t</match>\n\n"
	return conf
}

func appendBlacklist(b EnhancedFonts, f EnhancedFont) EnhancedFonts {
	if i, ok := b.Contains(f); ok {
		b[i].AppendCharset(f.Charset)
	} else {
		b = append(b, f)
	}
	return b
}

func getEmojiFontFilesByName(emojis string) []string {
	emojiFonts := ReadFontFilesFromDir("/usr/share/fonts/truetype", true)
	matched := []string{}
	m := make(map[string]string)
	for _, v := range emojiFonts {
		out, _ := exec.Command("/usr/bin/fc-scan", "--format", "%{family}", v).Output()
		m[string(out)] = v
	}
	for i, v := range strings.Split(emojis, ":") {
		if _, ok := m[v]; ok {
			matched = append(matched, m[v])
		} else {
			if i == len(emojis)-1 {
				fmt.Printf("%s not found in installed emoji fonts, maybe not installed at all?", v)
			}
		}
	}
	return matched
}

// GenerateEmojiBlacklist generate 81-emoji-blacklist-glyphs.conf
func GenerateEmojiBlacklist(userMode bool, opts Options) error {
	nonEmojiFonts := ReadFontFilesFromDir("/usr/share/fonts/truetype", false)
	emojiFonts := getEmojiFontFilesByName(opts.SystemEmojis)
	emojiCharset := BuildEmojiCharset(emojiFonts)

	emojiConf := configPreamble(userMode, "")

	blacklist := EnhancedFonts{}

	debug(opts.Verbosity, VerbosityDebug, "--- Blacklisting glyphs from system emoji fonts in non-emoji fonts.\n")

	if len(emojiCharset) == 0 {
		debug(opts.Verbosity, VerbosityDebug, "---")
		return nil
	}

	wg := sync.WaitGroup{}
	wg.Add(len(nonEmojiFonts))
	mux := sync.Mutex{}
	ch := make(chan struct{}, 100) // ch is a chan to avoid "too many open files" when os exec

	for _, font := range nonEmojiFonts {
		go func(fontFile string, verbosity int) {
			defer wg.Done()
			defer func() { <-ch }() // release chan
			ch <- struct{}{}        // acquire chan
			charset := BuildCharset(fontFile)
			in := IntersectCharset(charset, emojiCharset)
			debug(verbosity, VerbosityDebug, fmt.Sprintf("Calculating glyphs for %s\nIntersected charsets: %v\n", fontFile, in))

			if len(in) > 0 {
				names := GetFontName(fontFile)
				if len(names) > 1 {
					s := Style{}
					s.Load(fontFile)
					unstyled := GetUnstyledFontName(Font{names, []string{}, false})

					for _, f := range unstyled {
						c := EnhancedFont{Font{[]string{f}, []string{}, false}, in, s}
						mux.Lock()
						blacklist = appendBlacklist(blacklist, c)
						mux.Unlock()
					}

					slice.Remove(&names, unstyled)

					for _, f := range names {
						c := EnhancedFont{Font{[]string{f}, []string{}, false}, in, Style{}}
						mux.Lock()
						blacklist = appendBlacklist(blacklist, c)
						mux.Unlock()
					}
				} else {
					c := EnhancedFont{Font{[]string{names[0]}, []string{}, false}, in, Style{}}
					mux.Lock()
					blacklist = appendBlacklist(blacklist, c)
					mux.Unlock()
				}
			}
		}(font, opts.Verbosity)
	}

	wg.Wait()

	debug(opts.Verbosity, VerbosityDebug, "---")

	for _, f := range blacklist {
		emojiConf += generateBlacklistConfig(f)
	}

	emojiConf += "</fontconfig>\n"

	blacklistFile := filepath.Join("/etc/fonts/conf.d/81-emoji-blacklist-glyphs.conf")
	if userMode {
		blacklistFile = filepath.Join(GetEnv("HOME"), ".config/fontconfig/emoji-blacklist-glyphs.conf")
	}

	debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("blacklist file location: %s\n", blacklistFile))

	err := ioutil.WriteFile(blacklistFile, []byte(emojiConf), 0644)

	return err
}
