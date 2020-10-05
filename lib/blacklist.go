package lib

import (
	"fmt"
	"log"
	"sync"

	fccharset "github.com/openSUSE/fonts-config/fc-charset"
	"github.com/openSUSE/fonts-config/sysconfig"
)

// getEmojiFonts get all system emoji fonts
func getEmojiFonts(c Collection) Collection {
	c1 := Collection{}
	for _, font := range c {
		if font.IsEmoji() {
			c1 = append(c1, font)
		}
	}
	return c1
}

// Blacklist the font name and blacklisted charset
type Blacklist struct {
	Name string
	fccharset.Charset
}

// GenEmojiBlacklist generate 81-emoji-blacklist-glyphs.conf
// 1. blacklist charsets < 200d in emoji fonts, they are everywhere and non-emoji
// 2. balcklist emoji unicode codepoints in other fonts
func GenEmojiBlacklist(collection Collection, userMode bool, cfg sysconfig.SysConfig) {
	emojis := getEmojiFonts(collection)

	// no emoji fonts on the system
	if len(emojis) == 0 {
		return
	}

	Dbg(cfg.Int("VERBOSITY"), Debug, "blacklisting charsets < 200d in emoji fonts")

	var emojiConf, nonEmojiConf string
	var charset fccharset.Charset

	for _, font := range emojis {
		c := fccharset.Charset{}
		c1 := fccharset.Charset{}

		// select CharsetRange < 200d
		for _, v := range font.Charset {
			if v.Max < 8205 {
				c.Append(v)
			} else {
				c1.Append(v)
			}
		}

		charset = charset.Union(c1)

		// black'em
		if len(c) > 0 {
			b := Blacklist{}
			b.Name = font.Name[0]
			if len(font.Name) > 1 {
				b.Name = font.Name[len(font.Name)-1]
			}
			b.Charset = c
			emojiConf += genBlacklistConfig(b)
		}
	}

	Dbg(cfg.Int("VERBOSITY"), Debug, "blacklisting emoji glyphs from non-emoji fonts")

	wg := sync.WaitGroup{}
	wg.Add(len(collection) - len(emojis))
	mux := sync.Mutex{}
	ch := make(chan struct{}, 100) // ch is a chan to avoid "too many open files" when os exec

	for _, font := range collection {
		if !font.IsEmoji() {
			go func(f Font, verbosity int) {
				defer wg.Done()
				defer func() { <-ch }() // release chan
				ch <- struct{}{}        // acquire chan
				in := f.Charset.Intersect(charset)

				if len(in) > 0 {
					b := Blacklist{}
					b.Charset = in
					b.Name = f.Name[0]
					if len(f.Name) > 1 {
						b.Name = f.Name[len(f.Name)-1]
					}

					Dbg(verbosity, Debug, fmt.Sprintf("Processing font %s with intersected charset: %s", b.Name, b.Charset.String()))
					mux.Lock()
					nonEmojiConf += genBlacklistConfig(b)
					mux.Unlock()
				}
			}(font, cfg.Int("VERBOSITY"))
		}
	}

	wg.Wait()

	conf := genFcPreamble(userMode, "") + emojiConf + nonEmojiConf + FcSuffix
	blacklist := GetFcConfig("blacklist", userMode)
	err := overwriteOrRemoveFile(blacklist, []byte(conf))

	if err != nil {
		log.Fatalf("Can not write %s: %s\n", blacklist, err.Error())
	}
}
