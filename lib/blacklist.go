package lib

import (
	"fmt"
	"github.com/marguerite/util/slice"
	"log"
	"strings"
	"sync"
)

func getEmojiFontsByName(fonts Collection, emoji string) Collection {
	c := Collection{}
	// Prepare restricts
	for _, v := range strings.Split(emoji, ":") {
		tmp := fonts.FindByName(v)
		slice.Concat(&c, tmp)
	}
	return c
}

//genEmojiCharset generate charset array of a emoji font
func genEmojiCharset(fonts Collection) Charset {
	charset := Charset{}

	for _, font := range fonts {
		slice.Concat(&charset, font.Charset)
	}

	slice.Unique(&charset)

	/* common emojis that almost every font has
	   "#","*","0","1","2","3","4","5","6","7","8","9","©","®","™"," ",
	   "‼","↔","↕","↖","↗","↘","↙","▪","▫","☀","⁉","ℹ",
	   "▶","◀","☑","↩","↪","➡","⬅","⬆","⬇","♀","♂" */
	emojis := Charset{"0", "20", "23", "2a", "30", "31", "32", "33", "34", "35", "36", "37",
		"38", "39", "a9", "ae", "200d", "203c", "2049", "20e3", "2122",
		"2139", "2194", "2195", "2196", "2197", "2198", "2199", "21a9",
		"21aa", "25aa", "25ab", "25b6", "25c0", "2600", "2611", "2640",
		"2642", "27a1", "2b05", "2b06", "2b07"}
	slice.Remove(&charset, emojis)
	return charset
}

// GenEmojiBlacklist generate 81-emoji-blacklist-glyphs.conf
func GenEmojiBlacklist(fonts Collection, userMode bool, opts Options) {
	allEmojiFonts := fonts.FindByName("Emoji")
	nonEmojiFonts := fonts
	slice.Remove(&nonEmojiFonts, allEmojiFonts)
	emojiFonts := getEmojiFontsByName(allEmojiFonts, opts.SystemEmojis)
	emojiCharset := genEmojiCharset(emojiFonts)
	blacklist := Collection{}

	debug(opts.Verbosity, VerbosityDebug, "Blacklisting glyphs from chosen emoji fonts in non-emoji fonts.")

	if len(emojiCharset) == 0 {
		return
	}

	wg := sync.WaitGroup{}
	wg.Add(len(nonEmojiFonts))
	mux := sync.Mutex{}
	ch := make(chan struct{}, 100) // ch is a chan to avoid "too many open files" when os exec

	for _, font := range nonEmojiFonts {
		go func(f Font, verbosity int) {
			defer wg.Done()
			defer func() { <-ch }() // release chan
			ch <- struct{}{}        // acquire chan
			in := f.Charset.Intersect(emojiCharset)

			if len(in) > 0 {
				debug(verbosity, VerbosityDebug, fmt.Sprintf("Calculating glyphs for %s\nIntersected charsets: %v", f.Name[0], in))
				names := f.Name
				if len(names) > 1 {
					unstyled := f.UnstyledName()

					for _, u := range unstyled {
						newF := f
						newF.SetName([]string{u})
						newF.SetCharset(in)
						mux.Lock()
						blacklist.AppendCharsetOrFont(newF)
						mux.Unlock()
					}

					slice.Remove(&names, unstyled)

					for _, name := range names {
						newF := f
						newF.SetName([]string{name})
						newF.SetStyle(100, 80, 0)
						newF.SetCharset(in)
						mux.Lock()
						blacklist.AppendCharsetOrFont(newF)
						mux.Unlock()
					}
				} else {
					newF := f
					newF.SetCharset(in)
					newF.SetStyle(100, 80, 0)
					mux.Lock()
					blacklist.AppendCharsetOrFont(newF)
					mux.Unlock()
				}
			}
		}(font, opts.Verbosity)
	}

	wg.Wait()

	conf := ""
	emojiConf := ""

	for _, f := range blacklist {
		conf += genBlacklistConfig(f)
	}

	if len(conf) > 0 {
		emojiConf = genConfigPreamble(userMode, "") + conf + "</fontconfig>\n"
	}

	blacklistFile := GetConfigLocation("blacklist", userMode)

	debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("Blacklist file location: %s", blacklistFile))

	err := overwriteOrRemoveFile(blacklistFile, []byte(emojiConf), 0644)

	if err != nil {
		log.Fatalf("Can not write %s: %s\n", blacklistFile, err.Error())
	}
}
