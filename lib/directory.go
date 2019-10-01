package lib

import (
	"bufio"
	"fmt"
	"github.com/marguerite/util/dirutils"
	"github.com/marguerite/util/fileutils"
	"github.com/marguerite/util/slice"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// FontScaleEntry presents an item in fonts.scale.
type FontScaleEntry struct {
	Font   string
	XLFD   string
	Option string
}

// FontScaleEntries presents the fonts.scale file in structs
type FontScaleEntries []FontScaleEntry

// Contains makes it possible to check key existence like native map
func (f FontScaleEntries) Contains(key string) (FontScaleEntry, bool) {
	for _, v := range f {
		if v.Font == key || v.XLFD == key || v.Option == key {
			return v, true
		}
	}
	return FontScaleEntry{}, false
}

// Replace will replace an item in the existing FontScaleEntries
func (f FontScaleEntries) Replace(old, new FontScaleEntry) {
	n := len(f)
	for i := 0; i < n; i++ {
		if f[i] == old {
			f = append(append(f[:i], new), f[i+1:]...)
		}
	}
}

func (f FontScaleEntries) Len() int {
	return len(f)
}

func (f FontScaleEntries) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

func (f FontScaleEntries) Less(i, j int) bool {
	if f[i].Font == f[j].Font {
		if f[i].Option == f[j].Option {
			return f[i].XLFD < f[j].XLFD
		}
		return f[i].Option < f[j].Option
	}
	return f[i].Font < f[j].Font
}

func x11FontDirs(opts Options) []string {
	blacklistDirs := []string{"/usr/share/fonts", "/usr/share/fonts/encodings", "/usr/share/fonts/encodings/large"}
	dirs, _ := dirutils.Ls("/usr/share/fonts", "dir")
	out := []string{}
	for _, d := range dirs {
		if ok, e := slice.Contains(blacklistDirs, d); !ok && e == nil {
			out = append(out, d)
		}
	}

	debugText := "--- font directories\n"
	for _, d := range out {
		debugText += d + "\n"
	}
	debugText += "---\n"
	debug(opts.Verbosity, VerbosityDebug, debugText)

	return out
}

// mtimeDifferOrMissing: check if src/dst exist and their modification times differs
func mtimeDifferOrMissing(src, dst string) bool {
	srcInfo, err := os.Stat(src)
	if os.IsNotExist(err) {
		return true
	}
	dstInfo, err := os.Stat(dst)
	if os.IsNotExist(err) {
		return true
	}
	if srcInfo.ModTime() != dstInfo.ModTime() {
		return true
	}
	return false
}

func createSymlink(d string) error {
	forbiddenChars := []string{" ", ":"}
	files, _ := dirutils.Ls(d)
	for _, f := range files {
		for _, v := range forbiddenChars {
			if strings.Contains(f, v) {
				nf := strings.Replace(f, v, "_", -1)
				err := os.Symlink(f, nf)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// switchTTCap switch between Freetype style or X-TT style TTCap
func switchTTCap(s string, opts Options) string {
	// http://x-tt.osdn.jp/xtt-1.3/INSTALL.eng.txt
	freetypeRe := regexp.MustCompile(`:(\d):`)
	xttRe := regexp.MustCompile(`:fn=(\d):`)
	ttcapRe := regexp.MustCompile(`(?i)[[:alpha:]]+=`)
	if opts.GenerateTtcapEntries {
		if freetypeRe.MatchString(s) {
			m := freetypeRe.FindStringSubmatch(s)
			debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("-ttcap option is set: convert face number to TTCap syntax: fn=%s\n", m[1]))
			s = strings.Replace(s, m[0], ":fn="+m[1]+":", 1)
		}
	} else {
		if xttRe.MatchString(s) {
			m := xttRe.FindStringSubmatch(s)
			debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("-ttcap option is not set: convert face number to Freetype syntax: :%s:\n", m[1]))
			s = strings.Replace(s, m[0], ":"+m[1]+":", 1)
		}
		if ttcapRe.MatchString(s) {
			// there's more than just a face number, better ignore it
			debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("Unsupported entry: %s\n", s))
		}
	}
	return s
}

func generateObliqueFromItalic(fontScales *FontScaleEntries, opts Options) {
	// generate an oblique entry if only italic is there and vice versa:
	re := regexp.MustCompile(`(?i)(-[^-]+-[^-]+-[^-]+)(-[io]-)([^-]+-[^-]*-\d+-\d+-\d+-\d+-[pmc]-\d+-[^-]+-[^-]+)`)

	for _, f := range *fontScales {
		if re.MatchString(f.XLFD) {
			m := re.FindStringSubmatch(f.XLFD)
			xlfd := ""
			if m[2] == "-i-" {
				xlfd = m[1] + "-o-" + m[3]
			} else {
				xlfd = m[1] + "-i-" + m[3]
			}
			if _, ok := fontScales.Contains(xlfd); !ok {
				slice.Concat(fontScales, FontScaleEntry{f.Font, xlfd, f.Option})
				debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("generated o/i: %s %s\n", f.Option+f.Font, xlfd))
			}
		}
	}
}

func generateTTCap(fontScales *FontScaleEntries, opts Options) {
	// https://wiki.archlinux.org/index.php/X_Logical_Font_Description
	if !opts.GenerateTtcapEntries {
		return
	}

	debug(opts.Verbosity, VerbosityDebug, "generating TTCap options ...\n")

	re := regexp.MustCompile(`-medium-r`)
	suffix := []string{".ttf", ".ttc", ".otf", ".otc", ".pfa", ".pfb"}
	artificialItalic := "ai=0.2:"
	doubleStrike := "ds=y:"

	for _, f := range *fontScales {
		// don't touch existing TTCap options.
		if re.MatchString(f.XLFD) {
			if !fileutils.HasPrefixOrSuffix(f.Font, suffix) {
				// the freetype module handles TrueType, OpenType, and Type1 fonts.
				continue
			}

			if len(f.Option) != 0 {
				// there are already some TTCap options, better don't touch this
				continue
			}

			italic := strings.Replace(f.XLFD, "-medium-r-", "-medium-i-", 1)
			oblique := strings.Replace(f.XLFD, "-medium-r-", "-medium-i-", 1)
			bold := strings.Replace(f.XLFD, "-medium-r-", "-medium-i-", 1)
			boldItalic := strings.Replace(f.XLFD, "-medium-r-", "-medium-i-", 1)
			boldOblique := strings.Replace(f.XLFD, "-medium-r-", "-meidum-i-", 1)

			if _, ok := fontScales.Contains(italic); ok {
				slice.Concat(fontScales, FontScaleEntry{f.Font, italic, artificialItalic})
				debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("generated TTCap entry: %s %s\n", artificialItalic+f.Font, italic))
			}

			if _, ok := fontScales.Contains(oblique); ok {
				slice.Concat(fontScales, FontScaleEntry{f.Font, oblique, artificialItalic})
				debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("generated TTCap entry: %s %s\n", artificialItalic+f.Font, oblique))
			}

			if _, ok := fontScales.Contains(bold); ok {
				slice.Concat(fontScales, FontScaleEntry{f.Font, bold, doubleStrike})
				debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("generated TTCap entry: %s %s\n", doubleStrike+f.Font, bold))
			}

			if _, ok := fontScales.Contains(boldItalic); ok {
				slice.Concat(fontScales, FontScaleEntry{f.Font, boldItalic, doubleStrike + artificialItalic})
				debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("generated TTCap entry: %s %s\n", doubleStrike+artificialItalic+f.Font, boldItalic))
			}

			if _, ok := fontScales.Contains(boldOblique); ok {
				slice.Concat(fontScales, FontScaleEntry{f.Font, boldOblique, doubleStrike + artificialItalic})
				debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("generated TTCap entry: %s %s\n", doubleStrike+artificialItalic+f.Font, boldOblique))
			}
		}
	}

	// add bw=0.5 option when necessary:
	for _, f := range *fontScales {

		if !fileutils.HasPrefixOrSuffix(f.Font, suffix) {
			// the freetype module handles TrueType, OpenType, and Type1 fonts.
			continue
		}

		if strings.Contains(f.Option, "bw=") {
			// there is already a bw=<something> TTCap option, better don't touch this
			continue
		}

		if strings.Contains(f.XLFD, "c-0-jisx0201.1976-0") {
			slice.Replace(fontScales, f, FontScaleEntry{f.Font, f.XLFD, f.Option + "bw=0.5:"})
			debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("added bw=0.5 option: %s %s\n", f.Option+"bw=0.5:"+f.Font, f.XLFD))
		}
	}
}

// decodeXLFD split a XLFD description to font options, human readable family name, and XLFD font name
func decodeXLFD(s string) (string, string, string, error) {
	// ds=y:ai=0.2: NotoSansJP-Regular.otf -adobe-noto sans jp regular-bold-i-normal--0-0-0-0-p-0-iso10646-1
	re := regexp.MustCompile(`^(.*?)([^:\s]+)\s+(-.+?)\s*$`)
	if re.MatchString(s) {
		m := re.FindStringSubmatch(s)
		return m[1], m[2], m[3], nil
	}
	return "", "", "", fmt.Errorf("Not a valid XLFD description")
}

// fixHomeMadeFontScales fix homemade font scale entries in d/font.scale.*
func fixHomeMadeFontScales(d string, fontScale string, opts Options, fontScales *FontScaleEntries) (map[string]bool, error) {
	blacklist := make(map[string]bool)

	data, err := os.Open(fontScale)
	if err != nil {
		return blacklist, err
	}
	defer data.Close()

	debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("reading %s ...\n", filepath.Join(d, fontScale)))

	scanner := bufio.NewScanner(data)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		line := scanner.Text()
		ttOptions, familyName, xlfd, err := decodeXLFD(line)
		if err != nil {
			continue
		}

		debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("handmade entry found: options=%s font=%s xlfd=%s\n", ttOptions, familyName, xlfd))

		switchTTCap(ttOptions, opts)

		if !strings.HasSuffix(familyName, ".cid") {
			/* For font file name entries ending with ".cid", such a file
			usually doesn't exist and it doesn't need to. The backend which
			renders CID-keyed fonts just parses this name to find the real
			font files and mapping tables

			For other entries, we check whether the file exists. */
			if _, err := os.Stat(filepath.Join(d, familyName)); os.IsNotExist(err) {
				debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("file %s doesn't exist, discard enntry %s\n", filepath.Join(d, familyName), line))
				continue
			}
		}

		debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("adding handmade entry %s\n", line))
		slice.Concat(fontScales, FontScaleEntry{familyName, xlfd, ttOptions})
		/* This font has "handmade" fonts.scale entries.
		Add it to the blacklist to discard any entries for this font
		which which might have been automatically created
		by mkfontscale: */
		blacklist[familyName] = true
	}
	return blacklist, nil
}

// fixSystemFontScale fix font scale entries in d/fonts.scale file
func fixSystemFontScale(d string, opts Options, fontScales *FontScaleEntries, blacklist map[string]bool) error {
	systemFileScale := filepath.Join(d, "fonts.scale")

	data, err := os.Open(systemFileScale)
	if err != nil {
		return err
	}
	defer data.Close()

	debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("reading %s ...\n", systemFileScale))

	scanner := bufio.NewScanner(data)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		line := scanner.Text()
		ttOptions, familyName, xlfd, err := decodeXLFD(line)
		if err != nil {
			continue
		}

		debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("mkfontscale entry found: options=%s font=%s xlfd=%s\n", ttOptions, familyName, xlfd))

		/* mkfontscale apparently doesn't yet generate the special options for
		the freetype module to use different face numbers in .ttc files.
		But this might change, therefore it is probably better to check this as well: */
		switchTTCap(ttOptions, opts)

		if blacklist[familyName] {
			debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("%s is blacklisted, ignored.\n", filepath.Join(d, familyName)))
			continue
		}
		slice.Concat(fontScales, FontScaleEntry{familyName, xlfd, ttOptions})
	}

	return nil
}

// writeSystemFontScale write fontScales into dst file
func writeSystemFontScale(dst string, fontScales FontScaleEntries, verbosity int) error {
	debug(verbosity, VerbosityDebug, fmt.Sprintf("writing %s ...\n", dst))

	info, _ := os.Stat(dst)

	file, err := os.OpenFile(dst, os.O_WRONLY, info.Mode())
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(fmt.Sprintf("%d\n", len(fontScales)))
	if err != nil {
		return err
	}

	sort.Sort(fontScales)

	for _, font := range fontScales {
		_, err = file.WriteString(fmt.Sprintf("%s%s %s\n", font.Option, font.Font, font.XLFD))
		if err != nil {
			return err
		}
	}

	return nil
}

func fixFontScales(d string, opts Options) error {
	debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("------\nfix fonts.scale in %s\n", d))

	fontScaleEntries := FontScaleEntries{}
	blacklist := map[string]bool{}

	// first parse the "handmade" fonts.scale.* files:
	handmadeScales, err := filepath.Glob(filepath.Join(d, "*fonts.scale.*"))
	if err != nil {
		return err
	}

	for _, f := range handmadeScales {
		suffix := []string{".swp", ".bak", ".sav", ".save", ".rpmsave", ".rpmorig", ".rpmnew"}
		if fileutils.HasPrefixOrSuffix(f, suffix) {
			debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("%s is considered a backup file, ignored.\n", f))
			continue
		}

		blacklist, err = fixHomeMadeFontScales(d, f, opts, &fontScaleEntries)
		if err != nil {
			return err
		}
	}

	// Now parse the fonts.scale file automatically created by mkfontscale:
	err = fixSystemFontScale(d, opts, &fontScaleEntries, blacklist)
	if err != nil {
		return err
	}

	generateObliqueFromItalic(&fontScaleEntries, opts)
	generateTTCap(&fontScaleEntries, opts)

	err = writeSystemFontScale(filepath.Join(d, "fonts.scale"), fontScaleEntries, opts.Verbosity)
	if err != nil {
		return err
	}

	return nil
}

// checkScaleAndDirUpdate: check if we need to update fonts.scale and fonts.dir for dst directory
func checkScaleAndDirUpdate(dst, timestamp, fontScale, fontDir string, verbosity int) bool {
	for _, d := range []string{dst, fontScale, fontDir} {
		if mtimeDifferOrMissing(timestamp, d) {
			return true
		}
	}
	debug(verbosity, VerbosityDebug, fmt.Sprintf("%s is up to date.\n", dst))
	return false
}

func cleanScaleAndDir(fontScale, fontDir string) {
	for _, d := range []string{fontScale, fontDir} {
		fileutils.Remove(d)
	}
}

// createEmptyScaleFile if fonts.scale is not there as expected, create an empty one
func createEmptyFontScaleFile(fontScale string, verbosity int) bool {
	if _, err := os.Stat(fontScale); os.IsNotExist(err) {
		debug(verbosity, VerbosityDebug, "mkfontscale is not available or it failed\n-> creating an empty fonts.scale file.")
		fileutils.Touch(fontScale)
		return true
	}
	return false
}

// createOrCopyFontDirFile create an empty fonts.dir file or copy fonts.scale as fonts.dir
func createOrCopyFontDirFile(fontDir, fontScale string, verbosity int) bool {
	if _, err := os.Stat(fontDir); os.IsNotExist(err) {
		debug(verbosity, VerbosityDebug, "mkfontdir is not available or it failed -> ")
		if _, err := os.Stat(fontScale); !os.IsNotExist(err) {
			debug(verbosity, VerbosityDebug, "a fonts.scale file exists, copy it to fonts.dir.")
			fileutils.Copy(fontScale, fontDir)
		} else {
			debug(verbosity, VerbosityDebug, "no fonts.scale exists either, create an empty fonts.dir.")
			fileutils.Touch(fontDir)
		}
		return true
	}
	return false
}

// removeFontCache remove fonts.cache-* in dst
func removeFontCache(dst string) {
	caches, _ := filepath.Glob(filepath.Join(dst, "/fonts.cache-*"))
	for _, cache := range caches {
		fileutils.Remove(cache)
	}
}

// applyTimestamp create timestamp file and alter the modification time for those 4 files/directories
func applyTimestamp(timestamp, dst, fontScale, fontDir string) {
	if _, err := os.Stat(timestamp); os.IsNotExist(err) {
		fileutils.Touch(timestamp)
		t := time.Now()
		for _, d := range []string{timestamp, dst, fontScale, fontDir} {
			os.Chtimes(d, t, t)
		}
	}
}

// makeFontScaleAndDir: make fonts.scale and fonts.dir in the provided directory.
func makeFontScaleAndDir(d string, opts Options, force bool) error {
	timestamp := filepath.Join(d, "/.fonts-config-timestamp")
	fontScale := filepath.Join(d, "/fonts.scale")
	fontDir := filepath.Join(d, "/fonts.dir")

	if force || checkScaleAndDirUpdate(d, timestamp, fontScale, fontDir, opts.Verbosity) {

		debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("%s: creating fonts.{scale,dir}\n", d))

		cleanScaleAndDir(fontScale, fontDir)
		createSymlink(d)

		if _, err := os.Stat("/usr/bin/mkfontscale"); !os.IsNotExist(err) {
			cmd, _ := exec.Command("/usr/bin/mkfontscale", d).Output()
			debug(opts.Verbosity, VerbosityDebug, string(cmd)+"\n")
		}

		tryAgain := createEmptyFontScaleFile(fontScale, opts.Verbosity)

		err := fixFontScales(d, opts)
		if err != nil {
			return err
		}

		if _, err := os.Stat("/usr/bin/mkfontdir"); !os.IsNotExist(err) {
			cmdFlags := []string{}
			for _, v := range []string{"/usr/share/fonts/encodings", "/usr/share/fonts/encodings/large"} {
				if _, err := os.Stat(v); !os.IsNotExist(err) {
					cmdFlags = append(cmdFlags, "-e")
					cmdFlags = append(cmdFlags, v)
				}
			}
			cmdFlags = append(cmdFlags, d)
			cmd, _ := exec.Command("/usr/bin/mkfontdir", cmdFlags...).Output()
			debug(opts.Verbosity, VerbosityDebug, string(cmd)+"\n")
		}

		tryAgain = createOrCopyFontDirFile(fontDir, fontScale, opts.Verbosity)

		// Directory done. Now update time stamps:
		if tryAgain {
			/* mkfontscale and/or mkfontdir failed or didn't exist. Remove the
			   timestamp to make sure this script tries again next time
			   when the problem with mkfontscale and/or mkfontdir is fixed: */
			fileutils.Remove(timestamp)
		} else {
			/* fonts.cache-* files are now generated in /var/cache/fontconfig,
			   remove old cache files in the individual directories
			   (fc-cache does this as well when the cache files are out of date
			   but it can't hurt to remove them here as well just to make sure). */
			removeFontCache(d)
			applyTimestamp(timestamp, d, fontScale, fontDir)
		}

	}

	return nil
}

// MkFontScaleDir make fonts.scale and fonts.dir in font directories based on our fonts-config options
func MkFontScaleDir(opts Options, force bool) error {
	for _, d := range x11FontDirs(opts) {
		err := makeFontScaleAndDir(d, opts, force)
		if err != nil {
			return err
		}
	}
	return nil
}
