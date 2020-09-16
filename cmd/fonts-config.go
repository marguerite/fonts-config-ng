package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"strings"
	"unsafe"

	dirutils "github.com/marguerite/util/dir"
	"github.com/marguerite/util/slice"
	"github.com/openSUSE/fonts-config/lib"
	"github.com/urfave/cli"
)

// Version fonts-config's version
const Version string = "20190608"

func removeUserSetting(prefix string) error {
	if len(prefix) == 0 {
		return nil
	}
	for _, f := range []string{
		filepath.Join(prefix, "fonts-config"),
		filepath.Join(prefix, "rendering-options.conf"),
		filepath.Join(prefix, "family-prefer.conf"),
	} {
		err := os.Remove(f)
		if err != nil {
			return err
		}
	}
	return nil
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

// cliFlagsRltPos return relative positions in Options for flags set by cli
func cliFlagsRltPos(c *cli.Context) []int {
	rp := []int{}

	// read and dump private field "flagSet" of *cli.Context
	flagSet := reflect.ValueOf(c).Elem().FieldByName("flagSet")
	// make it readable
	flagSet = reflect.NewAt(flagSet.Type(), unsafe.Pointer(flagSet.UnsafeAddr())).Elem()
	cliFlags, _ := flagSet.Interface().(*flag.FlagSet)

	// check verbosity
	cliFlags.Visit(func(f *flag.Flag) {
		ok, _ := slice.Contains([]string{"quiet", "verbose", "debug"}, f.Name)
		if ok {
			rp = append(rp, 0)
		}
	})

	// check all other options
	for i, v := range c.App.Flags[6:22] {
		name := v.GetName()
		cliFlags.Visit(func(f *flag.Flag) {
			if strings.Split(name, ",")[0] == f.Name {
				rp = append(rp, i+1) // the verbosity
			}
		})
	}
	return rp
}

// verbosityLevel decide the verbosity level
func verbosityLevel(quiet, verbose, debug bool) int {
	m := map[bool]int{quiet: lib.VerbosityQuiet, verbose: lib.VerbosityVerbose, debug: lib.VerbosityDebug}
	for k, v := range m {
		if k {
			return v
		}
	}
	return 0
}

func getUserPrefix(userMode bool, verbosity int) string {
	if !userMode {
		return ""
	}
	prefix := filepath.Join(lib.GetEnv("HOME"), ".config/fontconfig")
	err := dirutils.MkdirP(prefix)
	if err != nil {
		log.Fatalf("Can not create %s: %s\n", prefix, err.Error())
	}
	return prefix
}

func loadOptions(opt lib.Options, c *cli.Context, userMode bool) lib.Options {
	sys := lib.NewReader(lib.GetConfigLocation("fc", false))
	config := lib.LoadOptions(sys, lib.NewOptions())
	log.Printf("System Configuration: %s", config.Bounce())

	if userMode {
		user := lib.NewReader(lib.GetConfigLocation("fc", true))
		config = lib.LoadOptions(user, config)
		log.Printf("With user configuration prepended: %s", config.Bounce())
	}

	config.Merge(opt, cliFlagsRltPos(c))
	log.Printf("With command line configuration prepended: %s", config.Bounce())

	writeOptions(config, userMode)

	return config
}

func writeOptions(opt lib.Options, userMode bool) {
	tmpl := lib.NewReader(lib.GetConfigLocation("fc", userMode))
	config := opt.FillTemplate(tmpl)

	f, err := os.OpenFile(lib.GetConfigLocation("fc", userMode), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalf("Can not open %s to write: %s.\n", lib.GetConfigLocation("fc", userMode), err.Error())
	}
	defer f.Close()

	lib.WriteOptions(f, config)
}

func main() {
	var userMode, remove, force, ttcap, enableJava, quiet, verbose, debug, autohint, bw, bwMono, ebitmaps, info, metric, forceFPL bool
	var hintstyle, lcdfilter, rgba, ebitmapsLang, emojis, preferredSans, preferredSerif, preferredMono string

	cli.VersionFlag = cli.BoolFlag{
		Name:  "version",
		Usage: "Display version and exit.",
	}
	app := cli.NewApp()
	app.Usage = "openSUSE fontconfig presets generator."
	app.Description = "openSUSE fontconfig presets generator."
	app.Version = Version
	app.Authors = []cli.Author{
		{Name: "Marguerite Su", Email: "marguerite@opensuse.org"},
	}
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:        "user, u",
			Usage:       "Run fontconfig setup for user.",
			Destination: &userMode,
		},
		cli.BoolFlag{
			Name:        "remove-user-setting, r",
			Usage:       "Remove current user's fontconfig setup.",
			Destination: &remove,
		},
		cli.BoolFlag{
			Name:        "force, f",
			Usage:       "Force the update of all generated files even if it appears unnecessary according to the time stamps",
			Destination: &force,
		},
		cli.BoolTFlag{
			Name:        "quiet, q",
			Usage:       "Work silently, unless an error occurs.",
			Destination: &quiet,
		},
		cli.BoolFlag{
			Name:        "verbose, v",
			Usage:       "Print some progress messages to standard output.Print some progress messages to standard output.",
			Destination: &verbose,
		},
		cli.BoolFlag{
			Name:        "debug, d",
			Usage:       "Print a lot of debugging messages to standard output.",
			Destination: &debug,
		},
		cli.StringFlag{
			Name:        "force-hintstyle",
			Usage:       "Which 'hintstyle' to enforce globally: hintfull, hintmedium, hintslight or hintnone.",
			Destination: &hintstyle,
		},
		cli.BoolFlag{
			Name:        "autohint",
			Usage:       "Use autohint even for well hinted fonts.",
			Destination: &autohint,
		},
		cli.BoolFlag{
			Name:        "force-bw",
			Usage:       "Do not use antialias.",
			Destination: &bw,
		},
		cli.BoolFlag{
			Name:        "force-bw-monospace",
			Usage:       "Do not use antialias for well instructed monospace fonts.",
			Destination: &bwMono,
		},
		cli.StringFlag{
			Name:        "lcdfilter",
			Usage:       "Which `lcdfilter` to use: lcdnone, lcddefault, lcdlight, lcdlegacy.",
			Destination: &lcdfilter,
		},
		cli.StringFlag{
			Name:        "rgba",
			Usage:       "Which `subpixel arrangement` your monitor use: none, rgb, vrgb, bgr, vbgr, unknown.",
			Destination: &rgba,
		},
		cli.BoolFlag{
			Name:        "ebitmaps",
			Usage:       "Whether to use embedded bitmaps or not",
			Destination: &ebitmaps,
		},
		cli.StringFlag{
			Name:        "ebitmapslang",
			Usage:       "Argument contains a `list` of colon separated languages, for example \"ja:ko:zh-CN\" which means \"use embedded bitmaps only for fonts supporting Japanese, Korean, or Simplified Chinese.",
			Destination: &ebitmapsLang,
		},
		cli.StringFlag{
			Name:        "emojis",
			Usage:       "Default emoji fonts. for example\"Noto Color Emoji:Twemoji Mozilla\", glyphs from these fonts will be blacklisted in other non-emoji fonts",
			Destination: &emojis,
		},
		cli.StringFlag{
			Name:        "sans-serif",
			Usage:       "Global preferred sans-serif families, separated by colon, which overrides any existing preference list, eg: \"Noto Sans SC:Noto Sans JP\".",
			Destination: &preferredSans,
		},
		cli.StringFlag{
			Name:        "serif",
			Usage:       "Global preferred serif families, separated by colon, which overrides any existing preference list, eg: \"Noto Serif SC:Noto Serif JP\".",
			Destination: &preferredSerif,
		},
		cli.StringFlag{
			Name:        "monospace",
			Usage:       "Global preferred sans-serif families, separated by colon, which overrides any existing preference list, eg: \"Noto Sans Mono CJK SC:Noto Sans Mono CJK JP\".",
			Destination: &preferredMono,
		},
		cli.BoolFlag{
			Name:        "metriccompatible",
			Usage:       "Use metric compatible fonts.",
			Destination: &metric,
		},
		cli.BoolFlag{
			Name:        "forceFPL",
			Usage:       "Force Family preference list, use together with -sansSerif/-serif/-monospace.",
			Destination: &forceFPL,
		},
		cli.BoolFlag{
			Name:        "ttcap",
			Usage:       "Generate TTCap entries..",
			Destination: &ttcap,
		},
		cli.BoolFlag{
			Name:        "java",
			Usage:       "Generate font setup for Java.",
			Destination: &enableJava,
		},
		cli.BoolFlag{
			Name:        "info",
			Usage:       "Print files used by fonts-config for YaST Fonts module.",
			Destination: &info,
		},
	}

	app.Action = func(c *cli.Context) error {

		if info {
			yastInfo()
			os.Exit(0)
		}

		currentUser, _ := user.Current()
		if !userMode && currentUser.Uid != "0" && currentUser.Username != "root" {
			log.Fatal("*** error: no root permissions; rerun with --user for user fontconfig setting.")
		}

		// parse verbosity
		verbosity := verbosityLevel(quiet, verbose, debug)
		userPrefix := getUserPrefix(userMode, verbosity)

		if remove {
			err := removeUserSetting(userPrefix)
			if err != nil {
				log.Fatalf("Can not remove configuration file: %s", err.Error())
			}
			os.Exit(0)
		}

		options := lib.Options{verbosity, hintstyle, autohint, bw, bwMono,
			lcdfilter, rgba, ebitmaps, ebitmapsLang,
			emojis, preferredSans, preferredSerif,
			preferredMono, metric, forceFPL, ttcap, enableJava}

		log.Printf("Command line options: %s", options.Bounce())

		config := loadOptions(options, c, userMode)

		if verbosity >= lib.VerbosityDebug {
			if userMode {
				log.Printf("USER mode (%s)\n", lib.GetEnv("USER"))
			} else {
				log.Println("SYSTEM mode")
			}

			text := "Sysconfig options (read from /etc/sysconfig/fonts-config"
			if userMode {
				text += fmt.Sprintf(", %s)\n", lib.GetConfigLocation("fc", userMode))
			} else {
				text += ")\n"
			}
			log.Println(text)
			log.Println(config.Bounce())
		}

		if !userMode {
			err := lib.MkFontScaleDir(config, force)
			lib.ErrChk(err)
			lib.GenMetricCompatibility(verbosity)
		}

		/*	# The following two calls may change files in /etc/fonts, therefore
			# they have to be called *before* fc-cache. If anything is
			# changed in /etc/fonts after calling fc-cache, fontconfig
			# will think that the cache files are out of date again. */

		collection := lib.Collection{}
		cache := lib.GetCacheLocation(userMode)
		if _, err := os.Stat(cache); !os.IsNotExist(err) {
			collection.Decode(lib.NewReader(cache).Bytes())
		}
		collection = lib.LoadFonts(collection)

		lib.GenTTType(collection, userMode)
		lib.GenRenderingOptions(userMode, config)
		lib.GenFamilyPreferenceLists(userMode, config)
		lib.GenEmojiBlacklist(collection, userMode, config)
		lib.GenNotoConfig(collection, userMode)
		lib.FixDualSpacing(collection, userMode)

		b, _ := collection.Encode()
		ioutil.WriteFile(cache, b, 0644)

		if !userMode {
			lib.FcCache(config.Verbosity)
			lib.FpRehash(config.Verbosity)
			if config.GenerateJavaFontSetup {
				lib.GenerateJavaFontSetup(config.Verbosity)
			}
			lib.ReloadXfsConfig(config.Verbosity)
		}

		return nil
	}

	_ = app.Run(os.Args)
}
