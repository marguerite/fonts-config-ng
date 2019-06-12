package lib

import (
	"bufio"
	"fmt"
	"github.com/marguerite/util/dirutils"
	"github.com/marguerite/util/fileutils"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func generateBitmapLanguagesConfig(opts Options) string {
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

func getPlaceholderType(line string) string {
	placeholders := []string{
		"_FORCE_HINTSTYLE_PLACEHOLDER_",
		"_FORCE_AUTOHINT_PLACEHOLDER_",
		"_FORCE_BW_PLACEHOLDER_",
		"_FORCE_BW_MONOSPACE_PLACEHOLDER_",
		"_USE_LCDFILTER_PLACEHOLDER_",
		"_USE_RGBA_PLACEHOLDER_",
		"_USE_EMBEDDED_BITMAPS_PLACEHOLDER_",
		"_SYSCONFIG_FILE_PLACEHOLDER_",
		"_METRIC_ALIASES_PLACEHOLDER_",
		"_INCLUDE_USER_RENDERING_PLACEHOLDER_",
	}

	for _, v := range placeholders {
		if strings.Contains(line, v) {
			return v
		}
	}
	return ""
}

func parseRenderingTemplatePlaceholderInLine(line string, opts Options, userMode bool) string {
	switch placeholder := getPlaceholderType(line); placeholder {
	case "_USE_EMBEDDED_BITMAPS_PLACEHOLDER_":
		return generateBitmapLanguagesConfig(opts)
	default:
	}
	return line + "\n"
}

// writeRenderingOptionsToFile check if our rendering options are same as the options in file and write
func writeRenderingOptionsToFile(file, optionText string, verbosity int) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	if optionText == string(data) {
		debug(verbosity, VerbosityDebug, fmt.Sprintf("--- %s unchanged ---\n", file))
	} else {
		debug(verbosity, VerbosityVerbose, fmt.Sprintf("Setting embedded bitmap usage in %s\n", file))
		debug(verbosity, VerbosityDebug, fmt.Sprintf("--- writing %s ---\n", file))
		err := ioutil.WriteFile(file, []byte(optionText), 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

// buildRenderingOptionsFromTemplate fill template with our rendering options
func buildRenderingOptionsFromTemplate(tmplFile string, opts Options, userMode bool) (string, error) {
	str := ""
	tmpl, err := os.Open(tmplFile)
	if err != nil {
		return "", err
	}
	defer tmpl.Close()

	scanner := bufio.NewScanner(tmpl)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		line := scanner.Text()
		str += parseRenderingTemplatePlaceholderInLine(line, opts, userMode)
	}
	return str, nil
}

// GenerateDefaultRenderingOptions generates default fontconfig rendering options conf
func GenerateDefaultRenderingOptions(userMode bool, opts Options) error {
	/* # reflect fonts-config syconfig variables or
	   # parameters in fontconfig setting to control rendering */
	tmplFile := "/usr/share/fonts-config/10-rendering-options.conf.template"
	renderFile := ""

	if userMode {
		renderFile = filepath.Join(GetEnv("HOME"), "/.config/fontconfig/rendering-options.conf")
		err := dirutils.MkdirP(renderFile, opts.Verbosity)
		if err != nil {
			return err
		}
		err = fileutils.Touch(renderFile)
		if err != nil {
			return err
		}
	} else {
		renderFile = "/etc/fonts/conf.d/10-rendering-options.conf"
	}

	debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("--- generating %s ---\n", renderFile))

	if _, err := os.Stat(tmplFile); os.IsNotExist(err) {
		debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("--- WARNING: %s doesn't exist!\n", tmplFile))
		return nil
	}

	renderText, err := buildRenderingOptionsFromTemplate(tmplFile, opts, userMode)
	if err != nil {
		return err
	}

	err = writeRenderingOptionsToFile(renderFile, renderText, opts.Verbosity)
	if err != nil {
		return err
	}
	return nil
}

// ValidStringOption return false if a string is "null", has suffix "none" or just empty.
func ValidStringOption(opt string) bool {
	if len(opt) == 0 || opt == "null" || strings.HasSuffix(opt, "none") {
		return false
	}
	return true
}

func renderingOptionsPreamble(userMode bool) string {
	config := "<?xml version=\"1.0\"?>\n<!DOCTYPE fontconfig SYSTEM \"fonts.dtd\">\n\n<!-- DO NOT EDIT; this is a generated file -->\n<!-- modify "
	config += SysconfigLoc(false)
	config += " && run /usr/bin/fonts-config "
	if userMode {
		config += "-\\-user "
	}
	config += "instead. -->\n<!-- using target=\"pattern\", because we want to change pattern in 60-family-prefer.conf\n\tregarding to this setting -->\n\n<fontconfig>\n"
	return config
}

func renderingOptionsForceHintstyle(opts Options) string {
	if !ValidStringOption(opts.ForceHintstyle) {
		return ""
	}
	debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("Forcing hintstyle: %s\n", opts.ForceHintstyle))
	config += "<!-- Choose prefered common hinting style here.  -->\n<!-- Possible values: no, hitnone, hitslight, hintmedium and hintfull. -->\n<!-- Can be overriden with some other options, e. g. force_bw\n\tor force_bw_monospace => hintfull -->\n"
	config += "\t<match target=\"pattern\" >\n\t\t<edit name=\"force_hintstyle\" mode=\"assign\">\n\t\t\t<string>"
	config += opt.ForceHintstyle
	config += "</string>\n\t\t</edit>\n\t</match>\n"
	return config
}

func renderingOptionsForceAutohint(opts Options) string {
	debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("Forcing autohint: %t\n", opts.ForceAutohint))
	config += "<!-- Force autohint always. -->\n<!-- If false, for well hinted fonts, their instructions are used for rendering. -->\n"
	config += "\t<match target=\"pattern\">\n\t\t<edit name=\"force_autohint\" mode=\"assign\">\n\t\t\t<bool>"
	config += fmt.Sprintf("%t", opts.ForceAutohint)
	config += "</bool>\n\t\t</edit>\n\t</match>\n"
	return config
}

func renderingOptionsBlackAndWhite(opts Options) string {
	debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("Forcing black and white: %t\n", opts.ForceBw))
	config += "<!-- Do not use font smoothing (black&white rendering) at all.  -->\n"
	config += "\t<match target=\"pattern\" >\n\t\t<edit name=\"force_bw\" mode=\"assign\">\n\t\t\t<bool>"
	config += fmt.Sprintf("%t", opts.ForceBW)
	config += "</bool>\n\t\t</edit>\t</match>\n"
	return config
}

func renderingOptionsMonospaceBlackAndWhite(opts Options) string {
	debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("Forcing black and white for good hinted monospace: %t\n", opts.ForceBwMonospace))
	config += "<!-- Do not use font smoothing for some monospaced fonts.  -->\n<!-- Liberation Mono, Courier New, Andale Mono, Monaco, etc. -->\n"
	config += "\t<match target=\"pattern\" >\t\t<edit name=\"force_bw_monospace\" mode=\"assign\">\n<bool>"
	config += fmt.Sprintf("%t", opts.ForceBwMonospace)
	config += "</bool>\n\t\t</edit>\n\t</match>\n"
	return config
}

func renderingOptionsLcdfilter(opts Options) string {
	if !ValidStringOption(opts.UseLcdfiler) {
		return ""
	}
	debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("Lcdfilter: %s\n", opts.UseLcdfilter))
	config += "<!-- Set LCD filter. Amend when you want use subpixel rendering. -->\n"
	config += "<!-- Don't forgot to set correct subpixel ordering in 'rgba' element. -->\n"
	config += "<!-- Possible values: lcddefault, lcdlight, lcdlegacy, lcdnone -->\n"
	config += "\t<match target=\"pattern\">\t\t<edit name=\"lcdfilter\" mode=\"append\">\n\t\t\t<const>"
	config += opts.UseLcdfiler
	config += "</const>\n\t\t</edit>\n\t</match>\n"
	return config
}

func renderingOptionsRGBA(opts Options) string {
	if !ValidStringOption(opts.UseRgba) {
		return ""
	}
	debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("Subpixel arrangement: %s\n", opts.UseRgba))
	config += "<!-- Set LCD subpixel arrangement and orientation.  -->\n"
	config += "<!-- Possible values: unknown, none, rgb, bgr, vrgb, vbgr. -->\n"
	config += "\t<match target=\"pattern\">\n\t\t<edit name=\"rgba\" mode=\"append\">\n<const>"
	config += opts.UseRgba
	config += "</const>\n\t\t</edit>\n\t</match>\n"
	return config
}

func renderingOptinsMetricCompatible(opts Options) string {
	config += "<!-- Search for metric compatible families? -->\n"
	config += "\t<match target=\"pattern\" >\n\t\t<edit name=\"search_metric_aliases\" mode=\"append\">\n<bool>"
	config += fmt.Sprintf("%t", opts.SearchMetricCompatible)
	config += "</bool>\n\t\t</edit>\n\t</match>\n"
	return config
}

func renderingOptionsUserOption(userMode bool) string {
	if userMode {
		return "\t<include ignore_missing=\"yes\" prefix=\"xdg\">fontconfig/rendering-options.conf</include>\n"
	}
	return ""
}
