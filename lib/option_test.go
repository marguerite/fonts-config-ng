package lib

import (
	"bytes"
	"reflect"
	"testing"
)

var opt Options = Options{0, "", false, false, false, "", "", false,
	"", "", "", "", "", false, false,
	false, false}

func TestNewOptions(t *testing.T) {
	got := NewOptions()
	if reflect.DeepEqual(got, opt) {
		t.Log("lib.NewOptions test passed")
	} else {
		t.Errorf("lib.NewOptions test failed. expected: %v, got %v.\n", opt, got)
	}
}

func TestLoadOptions(t *testing.T) {
	s := "VERBOSITY=\"\"\nFORCE_HINTSTYLE=\"\"\nFORCE_AUTOHINT=\"\"\nFORCE_BW=\"\"\n" +
		"FORCE_BW_MONOSPACE=\"\"\nUSE_LCDFILTER=\"\"\nUSE_RGBA=\"\"\nUSE_EMBEDDED_BITMAPS=\"\"\n" +
		"EMBEDDED_BITMAPS_LANGUAGES=\"\"\nPREFER_SANS_FAMILIES=\"\"\nPREFER_SERIF_FAMILIES=\"\"\n" +
		"PREFER_MONO_FAMILIES=\"\"\nSEARCH_METRIC_COMPATIBLE=\"\"\nFORCE_FAMILY_PREFERENCE_LISTS=\"\"\n" +
		"GENERATE_TTCAP_ENTRIES=\"\"\nGENERATE_JAVA_FONT_SETUP=\"\"\n"
	reader := bytes.NewBufferString(s)
	got := LoadOptions(reader, opt)
	if reflect.DeepEqual(got, opt) {
		t.Log("lib.LoadOptions test passed")
	} else {
		t.Errorf("lib.LoadOptions test failed. expected: %v, got %v.\n", opt, got)
	}
}
