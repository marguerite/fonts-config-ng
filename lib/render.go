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
	case "_FORCE_HINTSTYLE_PLACEHOLDER_":
		debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("--- forcing hintstyle: %s\n", opts.ForceHintstyle))
		return strings.Replace(line, placeholder, opts.ForceHintstyle, 1) + "\n"
	case "_FORCE_AUTOHINT_PLACEHOLDER_":
		debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("--- forcing autohint: %t\n", opts.ForceAutohint))
		return strings.Replace(line, placeholder, fmt.Sprintf("%t", opts.ForceAutohint), 1) + "\n"
	case "_FORCE_BW_PLACEHOLDER_":
		debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("--- forcing black and white: %t\n", opts.ForceBw))
		return strings.Replace(line, placeholder, fmt.Sprintf("%t", opts.ForceBw), 1) + "\n"
	case "_FORCE_BW_MONOSPACE_PLACEHOLDER_":
		debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("-- forcing black and white for good hinted monospace: %t\n", opts.ForceBwMonospace))
		return strings.Replace(line, placeholder, fmt.Sprintf("%t", opts.ForceBwMonospace), 1) + "\n"
	case "_USE_LCDFILTER_PLACEHOLDER_":
		debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("--- lcdfilter: %s\n", opts.UseLcdfilter))
		return strings.Replace(line, placeholder, opts.UseLcdfilter, 1) + "\n"
	case "_USE_RGBA_PLACEHOLDER_":
		debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("--- subpixel arrangement: %s\n", opts.UseRgba))
		return strings.Replace(line, placeholder, opts.UseRgba, 1) + "\n"
	case "_USE_EMBEDDED_BITMAPS_PLACEHOLDER_":
		return generateBitmapLanguagesConfig(opts)
	case "_SYSCONFIG_FILE_PLACEHOLDER_":
		line = strings.Replace(line, placeholder, "/etc/sysconfig/fonts-config", 1)
		if userMode {
			line = strings.Replace(line, "_FONTSCONFIG_RUN_PLACEHOLDER_", "/usr/bin/fonts-config -\\-user", 1)
		} else {
			line = strings.Replace(line, "_FONTSCONFIG_RUN_PLACEHOLDER_", "/usr/bin/fonts-config", 1)
		}
		return line + "\n"
	case "_METRIC_ALIASES_PLACEHOLDER_":
		return strings.Replace(line, placeholder, fmt.Sprintf("%t", opts.SearchMetricCompatible), 1) + "\n"
	case "_INCLUDE_USER_RENDERING_PLACEHOLDER_":
		if !userMode {
			// let user have a possiblity to override system settings
			return strings.Replace(line, placeholder, "<include ignore_missing=\"yes\" prefix=\"xdg\">fontconfig/rendering-options.conf</include>", 1) + "\n"
		}
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
		renderFile = filepath.Join(os.Getenv("HOME"), "/.config/fontconfig/rendering-options.conf")
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
