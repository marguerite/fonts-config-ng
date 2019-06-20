package lib

import (
	"bufio"
	"fmt"
	"github.com/marguerite/util/dirutils"
	"github.com/marguerite/util/fileutils"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func mkMetricCompatibility(avail string, userMode bool) (string, error) {
	metric := ""
	f, err := os.Open(avail)
	if err != nil {
		return metric, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		line := scanner.Text()
		metric += line + "\n"
		if strings.Contains(line, "<alias ") {
			metric += "\t  <test name=\"search_metric_aliases\"><bool>true</bool></test>\n"
		}
		if strings.Contains(line, "<!DOCTYPE ") {
			metric += "\n<!-- DO NOT EDIT; this is a generated file -->\n<!-- modify "
			if userMode {
				metric += filepath.Join(GetEnv("HOME") + ".config/fontconfig/fonts-config")
			} else {
				metric += "/etc/sysconfig/fonts-config"
			}
			metric += " && run /usr/bin/fonts-config instead -->\n\n"
		}
	}

	return metric, nil
}

func writeMetricCompatibilityFile(userMode bool, verbosity int) error {
	// replace fontconfig's /etc/fonts/conf.d/30-metric-aliases.conf
	// by fonts-config's one

	metricAvail := "/usr/share/fontconfig/conf.avail/30-metric-aliases.conf"
	metricFile := "/etc/fonts/conf.d/30-metric-aliases.conf"
	if _, err := os.Stat(metricAvail); os.IsNotExist(err) {
		debug(verbosity, VerbosityDebug, fmt.Sprintf("--- WARNING: %s not found, not writing %s ---\n", metricAvail, metricFile))
	}

	fileutils.Remove(metricFile, verbosity)

	metricText, err := mkMetricCompatibility(metricAvail, userMode)
	if err != nil {
		return err
	}
	debug(verbosity, VerbosityDebug, fmt.Sprintf("--- writing %s ---\n", metricFile))
	// same name as symlink from fontconfig
	err = ioutil.WriteFile(metricFile, []byte(metricText), 0644)
	if err != nil {
		return err
	}

	return nil
}

func fixFamilyName(name string) string {
	// remove comma and the rest of the family string #bsc998300
	re := regexp.MustCompile(`^(.*?),.*$`)
	if re.MatchString(name) {
		name = re.FindStringSubmatch(name)[1]
	}
	name = strings.Replace(name, "&", "&amp;", -1)
	return name
}

func buildFPL(genericName, preferredFamiliesInString string, userMode bool, opts Options) string {
	families := strings.Split(preferredFamiliesInString, ":")
	genericName = fixFamilyName(genericName)
	fpl := ""

	if len(families) < 2 {
		return ""
	}

	if opts.Verbosity >= VerbosityDebug {
		if opts.ForceFamilyPreferenceLists {
			fmt.Printf("--- Strongly preferred %s families: ", genericName)
		} else {
			fmt.Printf("--- Preferred %s families: ", genericName)
		}
	}

	if opts.ForceFamilyPreferenceLists {
		fpl += "\t<match>\n" +
			"\t\t<test name=\"family\"><string>$family</string></test>\n" +
			"\t\t<edit name=\"family\" mode=\"prepend_first\" binding=\"strong\">\n"

		for _, font := range families {
			font = fixFamilyName(font)
			fpl += "\t\t\t<string>" + font + "</string>\n"
			debug(opts.Verbosity, VerbosityDebug, "["+font+"]\n")
		}

		fpl += "\t\t</edit>\n\t</match>\n"
	} else {
		fpl += "\t<alias>\n"
		if !userMode {
			fpl += "\t\t<test name=\"user_preference_list\"><bool>false</bool></test>\n"
		}
		fpl += "\t\t<family>" + genericName + "</family>\n\t\t<prefer>\n"

		for _, font := range families {
			font = fixFamilyName(font)
			fpl += "\t\t\t<family>" + font + "</family>\n"
			debug(opts.Verbosity, VerbosityDebug, "["+font+"]\n")
		}
		fpl += "\t\t</prefer>\n\t</alias>\n"
	}

	return fpl
}

// GenerateFamilyPreferenceLists generates fontconfig fpl conf with user's explicit choices
func GenerateFamilyPreferenceLists(userMode bool, opts Options) error {
	fplFile := ""

	if userMode {
		fplFile = filepath.Join(GetEnv("HOME"), ".config/fontconfig/family-prefer.conf")
		err := dirutils.MkdirP(fplFile, opts.Verbosity)
		if err != nil {
			return err
		}
		err = fileutils.Touch(fplFile)
		if err != nil {
			return err
		}
	} else {
		fplFile = "/etc/fonts/conf.d/58-family-prefer-local.conf"
		err := writeMetricCompatibilityFile(userMode, opts.Verbosity)
		if err != nil {
			return err
		}
	}

	debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("--- generating %s ---\n", fplFile))

	fplText := configPreamble(userMode, "")

	if userMode {
		fplText += "\t<match target=\"pattern\">\n\t\t<edit name=\"user_preference_list\" mode=\"assign\">\n" +
			"\t\t\t<bool>true</bool>\n\t\t</edit>\n\t</match>\n"
	} else {
		fplText += "\t<!-- Let user override here defined system setting. -->\n" +
			"\t<match target=\"pattern\">\n\t\t<edit name=\"user_preference_list\" mode=\"assign\">\n" +
			"\t\t\t<bool>false</bool>\n\t\t</edit>\n\t</match>\n" +
			"\t<include ignore_missing=\"yes\" prefix=\"xdg\">fontconfig/family-prefer.conf</include>\n"
	}

	fplText += "\n"
	fplText += buildFPL("sans-serif", opts.PreferSansFamilies, userMode, opts)
	fplText += buildFPL("serif", opts.PreferSerifFamilies, userMode, opts)
	fplText += buildFPL("monospace", opts.PreferMonoFamilies, userMode, opts)
	fplText += "</fontconfig>\n"

	debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("--- writing %s ---\n", fplFile))

	err := ioutil.WriteFile(fplFile, []byte(fplText), 0644)
	if err != nil {
		return err
	}

	return nil
}
