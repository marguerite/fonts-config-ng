package lib

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

func genBitmapLanguagesConfig(opts Options) string {
	if opts.UseEmbeddedBitmaps {
		if opts.EmbeddedBitmapsLanguages == "no" || len(opts.EmbeddedBitmapsLanguages) == 0 {
			tmp := "\t<match target=\"font\">\n"
			tmp += "\t\t<edit name=\"embeddedbitmap\" mode=\"append\">\n"
			tmp += "\t\t\t<b>true</bool>\n"
			tmp += "\t\t</edit>\n"
			tmp += "\t</match>\n"
			return tmp
		}
		tmp := "\t<match target=\"font\">\n"
		tmp += "\t\t<edit name=\"embeddedbitmap\" mode=\"append\">\n"
		tmp += "\t\t\t<bool>false</bool>\n"
		tmp += "\t\t</edit>\n"
		tmp += "\t</match>\n"
		for _, v := range strings.Split(opts.EmbeddedBitmapsLanguages, ":") {
			tmp += "\t<match target=\"font\">\n"
			tmp += "\t\t<test name=\"lang\" compare=\"contains\"><string>" + v + "</string></test>\n"
			tmp += "\t\t<edit name=\"embeddedbitmap\" mode=\"append\"><bool>true</bool></edit>\n"
			tmp += "\t</match>\n"
		}
		return tmp
	}
	tmp := "\t<match target=\"font\">\n"
	tmp += "\t\t<edit name=\"embeddedbitmap\" mode=\"append\">\n"
	tmp += "\t\t\t<bool>false</bool>\n"
	tmp += "\t\t</edit>\n"
	tmp += "\t</match>\n"
	return tmp
}

// writeRenderingOptions check if our rendering options are same as the options in file and write
func writeRenderingOptions(f io.ReadWriter, out, opts string, verbosity int) {
	data, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatalf("Can not read from %s\n", out)
	}

	if opts == string(data) {
		debug(verbosity, VerbosityDebug, fmt.Sprintf("%s unchanged.\n", out))
	} else {
		debug(verbosity, VerbosityVerbose, fmt.Sprintf("Setting embedded bitmap usage in %s\n", out))
		debug(verbosity, VerbosityDebug, fmt.Sprintf("Writing %s.\n", out))
		n, err := f.Write([]byte(opts))
		if err != nil {
			log.Fatal(err)
		}
		if n != len(opts) {
			log.Fatal("Failed to write all data, configuration may be broken or incomplete.")
		}
	}
}

// GenRenderingOptions generates fontconfig rendering options conf
func GenRenderingOptions(userMode bool, opts Options) {
	/* # reflect fonts-config syconfig variables or
	   # parameters in fontconfig setting to control rendering */
	renderFile := ConfigLocation("rd", userMode)
	dat, err := os.OpenFile(renderFile, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalf("Can not open %s: %s", renderFile, err.Error())
	}
	defer dat.Close()

	debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("Generating %s.\n", renderFile))
	renderText := genRenderingOptions(opts, userMode)

	writeRenderingOptions(dat, renderFile, renderText, opts.Verbosity)
}

func genRenderingOptions(opts Options, userMode bool) string {
	config := genConfigPreamble(userMode, "<!-- using target=\"pattern\", because we want to change pattern in 60-family-prefer.conf\n\tregarding to this setting -->\n")
	config += genStringOptionConfig(opts.Verbosity, opts.ForceHintstyle, "Forcing hintstyle:",
		"<!-- Choose prefered common hinting style here.  -->\n<!-- Possible values: no, hitnone, hitslight, hintmedium and hintfull. -->\n<!-- Can be overriden with some other options, e. g. force_bw\n\tor force_bw_monospace => hintfull -->\n",
		"force_hintstyle", false, true)
	config += genBoolOptionConfig(opts.Verbosity, opts.ForceAutohint, "Forcing autohint:",
		"<!-- Force autohint always. -->\n<!-- If false, for well hinted fonts, their instructions are used for rendering. -->\n",
		"force_autohint", true)
	config += genBoolOptionConfig(opts.Verbosity, opts.ForceBw, "Forcing black and white:",
		"<!-- Do not use font smoothing (black&white rendering) at all.  -->\n",
		"force_bw", true)
	config += genBoolOptionConfig(opts.Verbosity, opts.ForceBwMonospace, "Forcing black and white for good hinted monospace:",
		"<!-- Do not use font smoothing for some monospaced fonts.  -->\n<!-- Liberation Mono, Courier New, Andale Mono, Monaco, etc. -->\n",
		"force_bw_monospace", true)
	config += genStringOptionConfig(opts.Verbosity, opts.UseLcdfilter, "Lcdfilter:",
		"<!-- Set LCD filter. Amend when you want use subpixel rendering. -->\n<!-- Don't forgot to set correct subpixel ordering in 'rgba' element. -->\n<!-- Possible values: lcddefault, lcdlight, lcdlegacy, lcdnone -->\n",
		"lcdfilter", true, false)
	config += genStringOptionConfig(opts.Verbosity, opts.UseRgba, "Subpixel arrangement:",
		"<!-- Set LCD subpixel arrangement and orientation.  -->\n<!-- Possible values: unknown, none, rgb, bgr, vrgb, vbgr. -->\n",
		"rgba", true, false)
	config += genBitmapLanguagesConfig(opts)
	config += genBoolOptionConfig(opts.Verbosity, opts.SearchMetricCompatible, "Search metric compatible fonts:",
		"<!-- Search for metric compatible families? -->\n",
		"search_metric_aliases", false)
	config += genUserInclude(userMode)
	config += "</fontconfig>\n"
	return config
}

// validStringOption return false if a string is "null", has suffix "none" or just empty.
func validStringOption(opt string) bool {
	if len(opt) == 0 || opt == "null" || strings.HasSuffix(opt, "none") {
		return false
	}
	return true
}

// genConfigPreamble generate fontconfig preamble
func genConfigPreamble(userMode bool, comment string) string {
	config := "<?xml version=\"1.0\"?>\n<!DOCTYPE fontconfig SYSTEM \"fonts.dtd\">\n\n<!-- DO NOT EDIT; this is a generated file -->\n<!-- modify "
	config += ConfigLocation("fc", false)
	config += " && run /usr/bin/fonts-config "
	if userMode {
		config += "-\\-user "
	}
	config += "instead. -->\n"
	config += comment
	config += "\n<fontconfig>\n"
	return config
}

func genStringOptionConfig(verbosity int, opt, dbgOutput, comment, editName string, cst, force bool) string {
	if !validStringOption(opt) {
		return ""
	}
	debug(verbosity, VerbosityDebug, fmt.Sprintf(dbgOutput+" %s\n", opt))
	config := comment
	config += "\t<match target=\"pattern\" >\n\t\t<edit name=\""
	config += editName
	config += "\" mode=\""
	if force {
		config += "assign"
	} else {
		config += "append"
	}
	config += "\">\n\t\t\t"
	if cst {
		config += "<const>"
	} else {
		config += "<string>"
	}
	config += opt
	if cst {
		config += "</const>"
	} else {
		config += "</string>"
	}
	config += "\n\t\t</edit>\n\t</match>\n"
	return config
}

func genBoolOptionConfig(verbosity int, opt bool, dbgOutput, comment, editName string, force bool) string {
	if strings.HasPrefix(editName, "force") && !opt {
		return ""
	}
	debug(verbosity, VerbosityDebug, fmt.Sprintf(dbgOutput+" %t\n", opt))
	config := comment
	config += "\t<match target=\"pattern\">\n\t\t<edit name=\""
	config += editName
	config += "\" mode=\""
	if force {
		config += "assign"
	} else {
		config += "append"
	}
	config += "\">\n\t\t\t<bool>"
	config += fmt.Sprintf("%t", opt)
	config += "</bool>\n\t\t</edit>\n\t</match>\n"
	return config
}

func genUserInclude(userMode bool) string {
	if userMode {
		return "\t<include ignore_missing=\"yes\" prefix=\"xdg\">fontconfig/rendering-options.conf</include>\n"
	}
	return ""
}
