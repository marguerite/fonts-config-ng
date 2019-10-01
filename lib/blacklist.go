package lib

import (
	"fmt"
	"github.com/marguerite/util/dirutils"
	"github.com/marguerite/util/fileutils"
	"github.com/marguerite/util/slice"
	"log"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

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
	emojiFonts := ReadFontFiles("Emoji")
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
func GenerateEmojiBlacklist(userMode bool, opts Options) {
	nonEmojiFonts := ReadFontFilesFromDir("/usr/share/fonts/truetype", false)
	emojiFonts := getEmojiFontFilesByName(opts.SystemEmojis)
	emojiCharset := BuildEmojiCharset(emojiFonts)
	blacklist := EnhancedFonts{}

	debug(opts.Verbosity, VerbosityDebug, "Blacklisting glyphs from system emoji fonts in non-emoji fonts.\n")

	if len(emojiCharset) == 0 {
		return
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

	conf := ""
	emojiConf := ""

	for _, f := range blacklist {
		conf += generateBlacklistConfig(f)
	}

	if len(conf) > 0 {
		emojiConf = genConfigPreamble(userMode, "") + conf + "</fontconfig>\n"
	}

	blacklistFile := GenConfigLocation("blacklist", userMode)

	debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("Blacklist file location: %s\n", blacklistFile))

	err := persist(blacklistFile, []byte(emojiConf), 0644)

	if err != nil {
		log.Fatalf("Can not write %s: %s\n", blacklistFile, err.Error())
	}
}
