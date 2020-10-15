package lib

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	dirutils "github.com/marguerite/util/dir"
	"github.com/marguerite/util/fileutils"
	"github.com/marguerite/util/slice"
	"github.com/openSUSE/fonts-config/sysconfig"
)

// FontScaleEntry presents an entry in fonts.scale.
type FontScaleEntry struct {
	Font   string
	XLFD   string
	Option string
}

// FontScale presents the fonts.scale file in structs
type FontScale []FontScaleEntry

// Find find FontScaleEntry who's Font/XLFD/Option equals to key.
func (f FontScale) Find(key string) (FontScaleEntry, bool) {
	for _, v := range f {
		if v.Font == key || v.XLFD == key || v.Option == key {
			return v, true
		}
	}
	return FontScaleEntry{}, false
}

// Replace replace an entry in the existing FontScale
func (f FontScale) Replace(old, new FontScaleEntry) {
	n := len(f)
	for i := 0; i < n; i++ {
		if f[i] == old {
			f = append(append(f[:i], new), f[i+1:]...)
		}
	}
}

func (f FontScale) Len() int {
	return len(f)
}

func (f FontScale) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

func (f FontScale) Less(i, j int) bool {
	if f[i].Font == f[j].Font {
		if f[i].Option == f[j].Option {
			return f[i].XLFD < f[j].XLFD
		}
		return f[i].Option < f[j].Option
	}
	return f[i].Font < f[j].Font
}

// getX11FontDirs get all directories containing fonts except those in the blacklist
func getX11FontDirs(cfg sysconfig.SysConfig) []string {
	blacklist := []string{"/usr/share/fonts", "/usr/share/fonts/encodings", "/usr/share/fonts/encodings/large"}
	systemFontDirs, _ := dirutils.Ls("/usr/share/fonts", true, true, "dir")
	fontDirs := []string{}
	for _, d := range systemFontDirs {
		if ok, e := slice.Contains(blacklist, d); !ok && e == nil {
			fontDirs = append(fontDirs, d)
		}
	}

	Dbg(cfg.Int("VERBOSITY"), Debug, func() string {
		str := "--- Font Directories\n"
		for _, d := range fontDirs {
			str += "\t" + d + "\n"
		}
		str += "---\n"
		return str
	})

	return fontDirs
}

// mtimeDifferOrMissing: check if src/dst exists and their modification times differs
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
	files, _ := dirutils.Ls(d, true, true)
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
func switchTTCap(s string, cfg sysconfig.SysConfig) string {
	// http://x-tt.osdn.jp/xtt-1.3/INSTALL.eng.txt
	freetypeRe := regexp.MustCompile(`:(\d):`)
	xttRe := regexp.MustCompile(`:fn=(\d):`)
	ttcapRe := regexp.MustCompile(`(?i)[[:alpha:]]+=`)
	if cfg.Bool("GENERATE_TTCAP_ENTRIES") {
		if freetypeRe.MatchString(s) {
			m := freetypeRe.FindStringSubmatch(s)
			Dbg(cfg.Int("VERBOSITY"), Debug, fmt.Sprintf("-ttcap option is set: convert face number to TTCap syntax: fn=%s\n", m[1]))
			s = strings.Replace(s, m[0], ":fn="+m[1]+":", 1)
		}
	} else {
		if xttRe.MatchString(s) {
			m := xttRe.FindStringSubmatch(s)
			Dbg(cfg.Int("VERBOSITY"), Debug, fmt.Sprintf("-ttcap option is not set: convert face number to Freetype syntax: :%s:\n", m[1]))
			s = strings.Replace(s, m[0], ":"+m[1]+":", 1)
		}
		if ttcapRe.MatchString(s) {
			// there's more than just a face number, better ignore it
			Dbg(cfg.Int("VERBOSITY"), Debug, fmt.Sprintf("Unsupported entry: %s\n", s))
		}
	}
	return s
}

func generateObliqueFromItalic(fontScale *FontScale, cfg sysconfig.SysConfig) {
	// generate an oblique entry if only italic is there and vice versa:
	re := regexp.MustCompile(`(?i)(-[^-]+-[^-]+-[^-]+)(-[io]-)([^-]+-[^-]*-\d+-\d+-\d+-\d+-[pmc]-\d+-[^-]+-[^-]+)`)

	for _, f := range *fontScale {
		if re.MatchString(f.XLFD) {
			m := re.FindStringSubmatch(f.XLFD)
			xlfd := ""
			if m[2] == "-i-" {
				xlfd = m[1] + "-o-" + m[3]
			} else {
				xlfd = m[1] + "-i-" + m[3]
			}
			if _, ok := fontScale.Find(xlfd); !ok {
				slice.Concat(fontScale, FontScaleEntry{f.Font, xlfd, f.Option})
				Dbg(cfg.Int("VERBOSITY"), Debug, fmt.Sprintf("generated o/i: %s %s\n", f.Option+f.Font, xlfd))
			}
		}
	}
}

func generateTTCap(fontScale *FontScale, cfg sysconfig.SysConfig) {
	// https://wiki.archlinux.org/index.php/X_Logical_Font_Description
	if !cfg.Bool("GENERATE_TTCAP_ENTRIES") {
		return
	}

	Dbg(cfg.Int("VERBOSITY"), Debug, "generating TTCap options ...\n")

	re := regexp.MustCompile(`-medium-r`)
	suffix := []string{".ttf", ".ttc", ".otf", ".otc", ".pfa", ".pfb"}
	artificialItalic := "ai=0.2:"
	doubleStrike := "ds=y:"

	for _, f := range *fontScale {
		// don't touch existing TTCap options.
		if re.MatchString(f.XLFD) {
			if fileutils.HasPrefixOrSuffix(f.Font, suffix) == 0 {
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

			if _, ok := fontScale.Find(italic); ok {
				slice.Concat(fontScale, FontScaleEntry{f.Font, italic, artificialItalic})
				Dbg(cfg.Int("VERBOSITY"), Debug, fmt.Sprintf("generated TTCap entry: %s %s\n", artificialItalic+f.Font, italic))
			}

			if _, ok := fontScale.Find(oblique); ok {
				slice.Concat(fontScale, FontScaleEntry{f.Font, oblique, artificialItalic})
				Dbg(cfg.Int("VERBOSITY"), Debug, fmt.Sprintf("generated TTCap entry: %s %s\n", artificialItalic+f.Font, oblique))
			}

			if _, ok := fontScale.Find(bold); ok {
				slice.Concat(fontScale, FontScaleEntry{f.Font, bold, doubleStrike})
				Dbg(cfg.Int("VERBOSITY"), Debug, fmt.Sprintf("generated TTCap entry: %s %s\n", doubleStrike+f.Font, bold))
			}

			if _, ok := fontScale.Find(boldItalic); ok {
				slice.Concat(fontScale, FontScaleEntry{f.Font, boldItalic, doubleStrike + artificialItalic})
				Dbg(cfg.Int("VERBOSITY"), Debug, fmt.Sprintf("generated TTCap entry: %s %s\n", doubleStrike+artificialItalic+f.Font, boldItalic))
			}

			if _, ok := fontScale.Find(boldOblique); ok {
				slice.Concat(fontScale, FontScaleEntry{f.Font, boldOblique, doubleStrike + artificialItalic})
				Dbg(cfg.Int("VERBOSITY"), Debug, fmt.Sprintf("generated TTCap entry: %s %s\n", doubleStrike+artificialItalic+f.Font, boldOblique))
			}
		}
	}

	// add bw=0.5 option when necessary:
	for _, f := range *fontScale {

		if fileutils.HasPrefixOrSuffix(f.Font, suffix) == 0 {
			// the freetype module handles TrueType, OpenType, and Type1 fonts.
			continue
		}

		if strings.Contains(f.Option, "bw=") {
			// there is already a bw=<something> TTCap option, better don't touch this
			continue
		}

		if strings.Contains(f.XLFD, "c-0-jisx0201.1976-0") {
			slice.Replace(fontScale, f, FontScaleEntry{f.Font, f.XLFD, f.Option + "bw=0.5:"})
			Dbg(cfg.Int("VERBOSITY"), Debug, fmt.Sprintf("added bw=0.5 option: %s %s\n", f.Option+"bw=0.5:"+f.Font, f.XLFD))
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
func fixHomeMadeFontScales(d string, fontScale string, cfg sysconfig.SysConfig, fontScales *FontScale) (map[string]bool, error) {
	blacklist := make(map[string]bool)

	data, err := os.Open(fontScale)
	if err != nil {
		return blacklist, err
	}
	defer data.Close()

	Dbg(cfg.Int("VERBOSITY"), Debug, fmt.Sprintf("reading %s ...\n", filepath.Join(d, fontScale)))

	scanner := bufio.NewScanner(data)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		line := scanner.Text()
		ttOptions, familyName, xlfd, err := decodeXLFD(line)
		if err != nil {
			continue
		}

		Dbg(cfg.Int("VERBOSITY"), Debug, fmt.Sprintf("handmade entry found: options=%s font=%s xlfd=%s\n", ttOptions, familyName, xlfd))

		switchTTCap(ttOptions, cfg)

		if !strings.HasSuffix(familyName, ".cid") {
			/* For font file name entries ending with ".cid", such a file
			usually doesn't exist and it doesn't need to. The backend which
			renders CID-keyed fonts just parses this name to find the real
			font files and mapping tables

			For other entries, we check whether the file exists. */
			if _, err := os.Stat(filepath.Join(d, familyName)); os.IsNotExist(err) {
				Dbg(cfg.Int("VERBOSITY"), Debug, fmt.Sprintf("file %s doesn't exist, discard enntry %s\n", filepath.Join(d, familyName), line))
				continue
			}
		}

		Dbg(cfg.Int("VERBOSITY"), Debug, fmt.Sprintf("adding handmade entry %s\n", line))
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
func fixSystemFontScale(d string, cfg sysconfig.SysConfig, fontScales *FontScale, blacklist map[string]bool) error {
	systemFileScale := filepath.Join(d, "fonts.scale")

	data, err := os.Open(systemFileScale)
	if err != nil {
		return err
	}
	defer data.Close()

	Dbg(cfg.Int("VERBOSITY"), Debug, fmt.Sprintf("reading %s ...\n", systemFileScale))

	scanner := bufio.NewScanner(data)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		line := scanner.Text()
		ttOptions, familyName, xlfd, err := decodeXLFD(line)
		if err != nil {
			continue
		}

		Dbg(cfg.Int("VERBOSITY"), Debug, fmt.Sprintf("mkfontscale entry found: options=%s font=%s xlfd=%s\n", ttOptions, familyName, xlfd))

		/* mkfontscale apparently doesn't yet generate the special options for
		the freetype module to use different face numbers in .ttc files.
		But this might change, therefore it is probably better to check this as well: */
		switchTTCap(ttOptions, cfg)

		if blacklist[familyName] {
			Dbg(cfg.Int("VERBOSITY"), Debug, fmt.Sprintf("%s is blacklisted, ignored.\n", filepath.Join(d, familyName)))
			continue
		}
		slice.Concat(fontScales, FontScaleEntry{familyName, xlfd, ttOptions})
	}

	return nil
}

// writeSystemFontScale write fontScales into dst file
func writeSystemFontScale(dst string, fontScales FontScale, verbosity int) error {
	Dbg(verbosity, Debug, fmt.Sprintf("writing %s ...\n", dst))

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

func fixFontScales(d string, cfg sysconfig.SysConfig) error {
	Dbg(cfg.Int("VERBOSITY"), Debug, fmt.Sprintf("------\nfix fonts.scale in %s\n", d))

	fontScale := FontScale{}
	blacklist := map[string]bool{}

	// first parse the "handmade" fonts.scale.* files:
	handmadeScales, err := filepath.Glob(filepath.Join(d, "*fonts.scale.*"))
	if err != nil {
		return err
	}

	for _, f := range handmadeScales {
		suffix := []string{".swp", ".bak", ".sav", ".save", ".rpmsave", ".rpmorig", ".rpmnew"}
		if fileutils.HasPrefixOrSuffix(f, suffix) != 0 {
			Dbg(cfg.Int("VERBOSITY"), Debug, fmt.Sprintf("%s is considered a backup file, ignored.\n", f))
			continue
		}

		blacklist, err = fixHomeMadeFontScales(d, f, cfg, &fontScale)
		if err != nil {
			return err
		}
	}

	// Now parse the fonts.scale file automatically created by mkfontscale:
	err = fixSystemFontScale(d, cfg, &fontScale, blacklist)
	if err != nil {
		return err
	}

	generateObliqueFromItalic(&fontScale, cfg)
	generateTTCap(&fontScale, cfg)

	err = writeSystemFontScale(filepath.Join(d, "fonts.scale"), fontScale, cfg.Int("VERBOSITY"))
	if err != nil {
		return err
	}

	return nil
}

// chkScaleAndDirUpdate: check if we need to update fonts.scale and fonts.dir for dst directory
func chkScaleAndDirUpdate(dst, timestamp, fontScale, fontDir string, verbosity int) bool {
	for _, d := range []string{dst, fontScale, fontDir} {
		if mtimeDifferOrMissing(timestamp, d) {
			return true
		}
	}
	Dbg(verbosity, Debug, fmt.Sprintf("%s is up to date.\n", dst))
	return false
}

func cleanScaleAndDir(fontScale, fontDir string) {
	for _, d := range []string{fontScale, fontDir} {
		os.Remove(d)
	}
}

// createEmptyScaleFile if fonts.scale is not there as expected, create an empty one
func createEmptyFontScaleFile(fontScale string, verbosity int) bool {
	if _, err := os.Stat(fontScale); os.IsNotExist(err) {
		Dbg(verbosity, Debug, "mkfontscale is not available or it failed\n-> creating an empty fonts.scale file.")
		fileutils.Touch(fontScale)
		return true
	}
	return false
}

// createOrCopyFontDirFile create an empty fonts.dir file or copy fonts.scale as fonts.dir
func createOrCopyFontDirFile(fontDir, fontScale string, verbosity int) bool {
	if _, err := os.Stat(fontDir); os.IsNotExist(err) {
		Dbg(verbosity, Debug, "mkfontdir is not available or it failed -> ")
		if _, err := os.Stat(fontScale); !os.IsNotExist(err) {
			Dbg(verbosity, Debug, "a fonts.scale file exists, copy it to fonts.dir.")
			fileutils.Copy(fontScale, fontDir)
		} else {
			Dbg(verbosity, Debug, "no fonts.scale exists either, create an empty fonts.dir.")
			fileutils.Touch(fontDir)
		}
		return true
	}
	return false
}

// rmFontCache remove fonts.cache-* in dst
func rmFontCache(dst string) {
	caches, _ := filepath.Glob(filepath.Join(dst, "/fonts.cache-*"))
	for _, cache := range caches {
		os.Remove(cache)
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
func makeFontScaleAndDir(d string, cfg sysconfig.SysConfig, force bool) error {
	timestamp := filepath.Join(d, "/.fonts-config-timestamp")
	fontScale := filepath.Join(d, "/fonts.scale")
	fontDir := filepath.Join(d, "/fonts.dir")

	if force || chkScaleAndDirUpdate(d, timestamp, fontScale, fontDir, cfg.Int("VERBOSITY")) {

		Dbg(cfg.Int("VERBOSITY"), Debug, fmt.Sprintf("%s: creating fonts.{scale,dir}\n", d))

		cleanScaleAndDir(fontScale, fontDir)
		createSymlink(d)

		if _, err := os.Stat("/usr/bin/mkfontscale"); !os.IsNotExist(err) {
			cmd, _ := exec.Command("/usr/bin/mkfontscale", d).Output()
			Dbg(cfg.Int("VERBOSITY"), Debug, string(cmd)+"\n")
		}

		tryAgain := createEmptyFontScaleFile(fontScale, cfg.Int("VERBOSITY"))

		err := fixFontScales(d, cfg)
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
			Dbg(cfg.Int("VERBOSITY"), Debug, string(cmd)+"\n")
		}

		tryAgain = createOrCopyFontDirFile(fontDir, fontScale, cfg.Int("VERBOSITY"))

		// Directory done. Now update time stamps:
		if tryAgain {
			/* mkfontscale and/or mkfontdir failed or didn't exist. Remove the
			   timestamp to make sure this script tries again next time
			   when the problem with mkfontscale and/or mkfontdir is fixed: */
			os.Remove(timestamp)
		} else {
			/* fonts.cache-* files are now generated in /var/cache/fontconfig,
			   remove old cache files in the individual directories
			   (fc-cache does this as well when the cache files are out of date
			   but it can't hurt to remove them here as well just to make sure). */
			rmFontCache(d)
			applyTimestamp(timestamp, d, fontScale, fontDir)
		}

	}

	return nil
}

// MkFontScaleDir make fonts.scale and fonts.dir in font directories based on our fonts-config options
func MkFontScaleDir(c sysconfig.SysConfig, force bool) error {
	for _, d := range getX11FontDirs(c) {
		err := makeFontScaleAndDir(d, c, force)
		if err != nil {
			return err
		}
	}
	return nil
}
