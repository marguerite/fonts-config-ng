package lib

import (
	"fmt"
	"log"
	"regexp"
	"strings"
)

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
			fmt.Printf("Strongly preferred %s families: ", genericName)
		} else {
			fmt.Printf("Preferred %s families: ", genericName)
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

// GenFamilyPreferenceLists generates fontconfig fpl conf with user's explicit choices
func GenFamilyPreferenceLists(userMode bool, opts Options) {
	fplFile := GetConfigLocation("fpl", userMode)
	debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("Generating %s", fplFile))

	fplText := genFcPreamble(userMode, "")

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
	fplText += FcSuffix

	debug(opts.Verbosity, VerbosityDebug, fmt.Sprintf("Writing %s.", fplFile))

	err := overwriteOrRemoveFile(fplFile, []byte(fplText), 0644)
	if err != nil {
		log.Fatalf("Can not write %s: %s\n", fplFile, err.Error())
	}
}
