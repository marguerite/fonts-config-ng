package lib

import (
	"fmt"
	"log"
	"sync"

	"github.com/marguerite/fonts-config-ng/charset"
	ft "github.com/marguerite/fonts-config-ng/font"
	"github.com/marguerite/fonts-config-ng/sysconfig"
)

// getEmojiFonts get all system emoji fonts
func getEmojiFonts(c ft.Collection) ft.Collection {
	c1 := ft.Collection{}
	for _, ft := range c {
		if ft.IsEmoji() {
			c1 = append(c1, ft)
		}
	}
	return c1
}

// Blacklist the font name and blacklisted charset
type Blacklist struct {
	Name string
	charset.Charset
}

// GenEmojiBlacklist generate 81-emoji-blacklist-glyphs.conf
// 1. blacklist charsets < 200d in emoji fonts, they are everywhere and non-emoji
// 2. balcklist emoji unicode codepoints in other fonts
func GenEmojiBlacklist(collection ft.Collection, userMode bool, cfg sysconfig.Config) {
	emojis := getEmojiFonts(collection)

	// no emoji fonts on the system
	if len(emojis) == 0 {
		return
	}

	Dbg(cfg.Int("VERBOSITY"), Debug, "blacklisting charsets < 200d in emoji fonts")

	var emojiConf, nonEmojiConf string
	var cs charset.Charset

	for _, ft := range emojis {
		var c, c1 charset.Charset

		// select CharsetRange < 200d
		for _, v := range ft.Charset {
			if v.Max < 8205 {
				c.Append(v)
			} else {
				c1.Append(v)
			}
		}

		cs = cs.Union(c1)

		// black'em
		if len(c) > 0 {
			b := Blacklist{}
			b.Name = ft.Name[0]
			if len(ft.Name) > 1 {
				b.Name = ft.Name[len(ft.Name)-1]
			}
			b.Charset = c
			emojiConf += genBlacklistConfig(b)
		}
	}

	Dbg(cfg.Int("VERBOSITY"), Debug, "blacklisting emoji glyphs from non-emoji fonts")

	wg := sync.WaitGroup{}
	wg.Add(len(collection) - len(emojis))
	mux := sync.Mutex{}

	for _, font := range collection {
		if !font.IsEmoji() {
			go func(f ft.Font, verbosity int) {
				defer wg.Done()
				in := f.Charset.Intersect(cs)

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
