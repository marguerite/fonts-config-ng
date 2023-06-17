package lib

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"

	"github.com/marguerite/fonts-config-ng/sysconfig"
)

type FamilyPreferLists []FamilyPreferList

type FamilyPreferList struct {
	GenericName string
	List        OrderedList
}

func NewFamilyPreferList(name string, list ...string) FamilyPreferList {
	return FamilyPreferList{name, NewOrderedList(list...)}
}

type OrderedList []Ordered

type Ordered struct {
	Order int
	Item  string
}

func (ord OrderedList) Len() int {
	return len(ord)
}

func (ord OrderedList) Less(i, j int) bool {
	return ord[i].Order < ord[j].Order
}

func (ord OrderedList) Swap(i, j int) {
	ord[i], ord[j] = ord[j], ord[i]
}

func NewOrderedList(list ...string) (ord OrderedList) {
	for i := 0; i < len(list); i++ {
		ord = append(ord, Ordered{i, list[i]})
	}
	sort.Sort(ord)
	return ord
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

func buildFPL(genericName, preferredFamiliesInString string, userMode bool, cfg sysconfig.Config) string {
	families := strings.Split(preferredFamiliesInString, ":")
	genericName = fixFamilyName(genericName)
	fpl := ""

	if len(families) < 2 {
		return ""
	}

	Dbg(cfg.Int("VERBOSITY"), Debug, func(force bool) string {
		if force {
			return fmt.Sprintf("Strongly preferred %s families: ", genericName)
		}
		return fmt.Sprintf("Preferred %s families: ", genericName)
	}, cfg.Bool("FORCE_FAMILY_PREFERENCE_LISTS"))

	if cfg.Bool("FORCE_FAMILY_PREFERENCE_LISTS") {
		fpl += "\t<match>\n" +
			"\t\t<test name=\"family\"><string>$family</string></test>\n" +
			"\t\t<edit name=\"family\" mode=\"prepend_first\" binding=\"strong\">\n"

		for _, font := range families {
			font = fixFamilyName(font)
			fpl += "\t\t\t<string>" + font + "</string>\n"
			Dbg(cfg.Int("VERBOSITY"), Debug, "["+font+"]\n")
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
			Dbg(cfg.Int("VERBOSITY"), Debug, "["+font+"]\n")
		}
		fpl += "\t\t</prefer>\n\t</alias>\n"
	}

	return fpl
}

// GenFamilyPreferenceLists generates fontconfig fpl conf with user's explicit choices
func GenFamilyPreferenceLists(userMode bool, cfg sysconfig.Config) {
	fplFile := GetFcConfig("fpl", userMode)
	Dbg(cfg.Int("VERBOSITY"), Debug, fmt.Sprintf("Generating %s", fplFile))

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
	fplText += buildFPL("sans-serif", cfg.String("PREFER_SANS_FAMILIES"), userMode, cfg)
	fplText += buildFPL("serif", cfg.String("PREFER_SERIF_FAMILIES"), userMode, cfg)
	fplText += buildFPL("monospace", cfg.String("PREFER_MONO_FAMILIES"), userMode, cfg)
	fplText += FcSuffix

	Dbg(cfg.Int("VERBOSITY"), Debug, fmt.Sprintf("Writing %s.", fplFile))

	err := overwriteOrRemoveFile(fplFile, []byte(fplText))
	if err != nil {
		log.Fatalf("Can not write %s: %s\n", fplFile, err.Error())
	}
}
