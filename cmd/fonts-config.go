package main

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/marguerite/fonts-config-ng/font"
	"github.com/marguerite/fonts-config-ng/lib"
	"github.com/marguerite/fonts-config-ng/sysconfig"
	"github.com/marguerite/go-stdlib/dir"
	"github.com/marguerite/go-stdlib/ioutils"
	"github.com/marguerite/go-stdlib/slice"
	"github.com/urfave/cli"
)

// VERSION fonts-config's version
const VERSION string = "20201005"

func rmUserFcConfig(userMode bool) {
	if !userMode {
		return
	}
	path := filepath.Join(os.Getenv("HOME"), ".config/fontconfig")
	cfgs, _ := dir.Glob(path, "\\.conf$")
	slice.Remove(&cfgs, filepath.Join(path, "fonts.conf"))
	for _, f := range cfgs {
		os.Remove(f)
	}
}

func yastInfo() {
	// compatibility only, no actual use.
	fmt.Printf("Involved Files\n" +
		"  rendering config: /etc/fonts/conf.d/10-rendering-options.conf\n" +
		"  java fontconfig properties: /usr/lib*/jvm/jre/lib/fontconfig.SUSE.properties\n" +
		"  user sysconfig file: fontconfig/fonts-config\n" +
		"  metric compatibility avail: /usr/share/fontconfig/conf.avail/30-metric-aliases.conf\n" +
		"  metric compatibility bw symlink: /etc/fonts/conf.d/31-metric-aliases-bw.conf\n" +
		"  metric compatibility config: /etc/fonts/conf.d/30-metric-aliases.conf\n" +
		"  local family list: /etc/fonts/conf.d/58-family-prefer-local.conf\n" +
		"  metric compatibility symlink: /etc/fonts/conf.d/30-metric-aliases.conf\n" +
		"  user family list: fontconfig/family-prefer.conf\n" +
		"  java fontconfig properties template: /usr/share/fonts-config/fontconfig.SUSE.properties.template\n" +
		"  rendering config template: /usr/share/fonts-config/10-rendering-options.conf.template\n" +
		"  sysconfig file: /etc/sysconfig/fonts-config\n" +
		"  user rendering config: fontconfig/rendering-options.conf\n")
}

func main() {
	cli.VersionFlag = cli.BoolFlag{
		Name:  "version",
		Usage: "Display version and exit.",
	}
	app := cli.NewApp()
	app.Usage = "openSUSE fontconfig presets generator."
	app.Description = "openSUSE fontconfig presets generator."
	app.Version = VERSION
	app.Authors = []cli.Author{
		{Name: "Marguerite Su", Email: "marguerite@opensuse.org"},
	}
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "user, u",
			Usage: "Run fontconfig setup for user.",
		},
		cli.BoolFlag{
			Name:  "remove-user-setting, r",
			Usage: "Remove current user's fontconfig setup.",
		},
		cli.BoolFlag{
			Name:  "force, f",
			Usage: "Force the update of all generated files even if it appears unnecessary according to the time stamps",
		},
		cli.BoolTFlag{
			Name:  "quiet, q",
			Usage: "Work silently, unless an error occurs.",
		},
		cli.BoolFlag{
			Name:  "verbose, v",
			Usage: "Print some progress messages to standard output.Print some progress messages to standard output.",
		},
		cli.BoolFlag{
			Name:  "debug, d",
			Usage: "Print a lot of debugging messages to standard output.",
		},
		cli.StringFlag{
			Name:  "force-hintstyle",
			Usage: "Which `hintstyle` to enforce globally: hintfull, hintmedium, hintslight or hintnone.",
		},
		cli.BoolFlag{
			Name:  "force-autohint",
			Usage: "Use autohint even for well hinted fonts.",
		},
		cli.BoolFlag{
			Name:  "force-bw",
			Usage: "Do not use antialias.",
		},
		cli.BoolFlag{
			Name:  "force-bw-monospace",
			Usage: "Do not use antialias for well instructed monospace fonts.",
		},
		cli.StringFlag{
			Name:  "use-lcdfilter",
			Usage: "Which `lcdfilter` to use: lcdnone, lcddefault, lcdlight, lcdlegacy.",
		},
		cli.StringFlag{
			Name:  "use-rgba",
			Usage: "Which `subpixel arrangement` your monitor use: none, rgb, vrgb, bgr, vbgr, unknown.",
		},
		cli.BoolFlag{
			Name:  "use-embedded-bitmaps",
			Usage: "Whether to use embedded bitmaps or not",
		},
		cli.StringFlag{
			Name:  "embedded-bitmaps-languages",
			Usage: "Colon-separated `language list`, for example \"ja:ko:zh-CN\" which means \"use embedded bitmaps only for fonts supporting Japanese, Korean, or Simplified Chinese.",
		},
		cli.StringFlag{
			Name:  "prefer-sans-families",
			Usage: "Global preferred `sans-serif` families, separated by colon, which overrides any existing preference list, eg: \"Noto Sans SC:Noto Sans JP\".",
		},
		cli.StringFlag{
			Name:  "prefer-serif-families",
			Usage: "Global preferred `serif` families, separated by colon, which overrides any existing preference list, eg: \"Noto Serif SC:Noto Serif JP\".",
		},
		cli.StringFlag{
			Name:  "prefer-mono-families",
			Usage: "Global preferred `monospace` families, separated by colon, which overrides any existing preference list, eg: \"Noto Sans Mono CJK SC:Noto Sans Mono CJK JP\".",
		},
		cli.BoolFlag{
			Name:  "search-metric-compatible",
			Usage: "Use metric compatible fonts.",
		},
		cli.BoolFlag{
			Name:  "force-family-preference-lists",
			Usage: "Force Family preference list, use together with -prefer-*-families.",
		},
		cli.BoolFlag{
			Name:  "generate-ttcap-entries",
			Usage: "Generate TTCap entries..",
		},
		cli.BoolFlag{
			Name:  "generate-java-font-setup",
			Usage: "Generate font setup for Java.",
		},
		cli.BoolFlag{
			Name:  "info",
			Usage: "Print files used by fonts-config for YaST Fonts module.",
		},
	}

	app.Action = func(c *cli.Context) error {

		if c.Bool("info") {
			yastInfo()
			os.Exit(0)
		}

		currentUser, _ := user.Current()
		if !c.Bool("u") && currentUser.Uid != "0" && currentUser.Username != "root" {
			log.Fatal("*** error: no root permissions; rerun with --user for user fontconfig setting.")
		}

		// parse verbosity
		verbosity := 0
		if c.Bool("d") {
			verbosity = 256
		}
		if c.Bool("v") {
			verbosity = 1
		}

		if c.Bool("r") {
			rmUserFcConfig(c.Bool("u"))
			os.Exit(0)
		}

		cfg := make(sysconfig.Config)
		f := ioutils.NewReaderFromFile("/etc/sysconfig/fonts-config")
		cfg.Unmarshal(f)
		cfg["VERBOSITY"] = verbosity

		// overwrite cfg with cli args
		for k, v := range cfg {
			flag := strings.ReplaceAll(strings.ToLower(k), "_", "-")
			if c.IsSet(flag) {
				if reflect.TypeOf(v).Kind() == reflect.Bool {
					cfg[k] = c.Bool(flag)
					continue
				}
				cfg[k] = c.String(flag)
			}
		}

		lib.Dbg(verbosity, lib.Debug, func(mode bool) string {
			if mode {
				return fmt.Sprintf("--- USER mode (%s)\n", os.Getenv("USER"))
			}
			return fmt.Sprintf("--- SYSTEM mode\n")
		}, c.Bool("u"))

		if !c.Bool("u") {
			err := lib.MkFontScaleDir(cfg, c.Bool("force"))
			if err != nil {
				log.Fatal(err)
			}
			lib.GenMetricCompatibility(verbosity)
		}

		/*	# The following two calls may change files in /etc/fonts, therefore
			# they have to be called *before* fc-cache. If anything is
			# changed in /etc/fonts after calling fc-cache, fontconfig
			# will think that the cache files are out of date again. */

		collection := font.NewCollection()
		lib.GenRenderingOptions(c.Bool("u"), cfg)
		lib.GenFamilyPreferenceLists(c.Bool("u"), cfg)
		lib.GenEmojiBlacklist(collection, c.Bool("u"), cfg)
		lib.GenNotoConfig(collection, c.Bool("u"))
		lib.GenCJKConfig(collection, c.Bool("u"))

		if !c.Bool("u") {
			lib.FcCache(cfg.Int("VERBOSITY"))
			lib.FpRehash(cfg.Int("VERBOSITY"))
			if cfg.Bool("GENERATE_JAVA_FONT_SETUP") {
				lib.GenerateJavaFontSetup(collection, cfg)
			}
			lib.ReloadXfsConfig(cfg.Int("VERBOSITY"))
		}

		return nil
	}

	_ = app.Run(os.Args)
}
