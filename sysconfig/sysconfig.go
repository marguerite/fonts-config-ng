package sysconfig

import (
	"bufio"
	"io"
	"reflect"
	"strconv"
	"strings"
)

// Config dump /etc/sysconfig/*.sysconfig to golang structs
type Config map[string]interface{}

// Bool find a bool value
func (cfg Config) Bool(s string) bool {
	if val, ok := cfg[s]; ok {
		b, _ := val.(bool)
		return b
	}
	return false
}

// Int find an integer value
func (cfg Config) Int(s string) int {
	if val, ok := cfg[s]; ok {
		i, _ := val.(int)
		return i
	}
	return 0
}

// String find a string value
func (cfg Config) String(s string) string {
	if val, ok := cfg[s]; ok {
		s1, _ := val.(string)
		return s1
	}
	return ""
}

// Marshal sysconfig to bytes again from template f.
func (cfg Config) Marshal(f io.Reader, b []byte) {
	s := bufio.NewScanner(f)
	for s.Scan() {
		if strings.HasPrefix(s.Text(), "#") || len(s.Text()) == 0 {
			for _, v := range s.Bytes() {
				b = append(b, v)
			}
			continue
		}
		arr := strings.Split(strings.ReplaceAll(s.Text(), "\"", ""), "=")
		if val, ok := cfg[arr[0]]; ok {
			if reflect.TypeOf(val).Kind() == reflect.Int {
				i, _ := val.(int)
				val = i
			}
			if reflect.TypeOf(val).Kind() == reflect.Bool {
				b, _ := strconv.ParseBool(arr[1])
				if b == val {
					continue
				}
				ok, _ := val.(bool)
				val = strconv.FormatBool(ok)
			}
			s1, _ := val.(string)
			for _, v := range []byte(arr[0] + "=\"" + s1 + "\"") {
				b = append(b, v)
			}
		}
	}
}

// Unmarshal unmarshal sysconfig
func (cfg Config) Unmarshal(f io.Reader) {
	s := bufio.NewScanner(f)
	for s.Scan() {
		if strings.HasPrefix(s.Text(), "#") || len(s.Text()) == 0 {
			continue
		}
		arr := strings.Split(strings.ReplaceAll(s.Text(), "\"", ""), "=")
		if len(arr) > 1 {
			val, err := strconv.Atoi(arr[1])
			if err != nil {
				switch arr[1] {
				case "yes":
					arr[1] = "true"
				case "no":
					arr[1] = "false"
				}
				b, err1 := strconv.ParseBool(arr[1])
				if err1 != nil {
					cfg[arr[0]] = arr[1]
					continue
				}
				cfg[arr[0]] = b
				continue
			}
			cfg[arr[0]] = val
			continue
		}
		var face interface{}
		cfg[arr[0]] = face
	}
}
