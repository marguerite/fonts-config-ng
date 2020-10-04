package sysconfig

import (
	"bufio"
	"io"
	"reflect"
	"strconv"
	"strings"
)

// SysConfig dump /etc/sysconfig/*.sysconfig to golang structs
type SysConfig map[string]interface{}

// Bool find a bool value
func (sys SysConfig) Bool(s string) bool {
	if val, ok := sys[s]; ok {
		b, _ := val.(bool)
		return b
	}
	return false
}

// Int find an integer value
func (sys SysConfig) Int(s string) int {
	if val, ok := sys[s]; ok {
		i, _ := val.(int)
		return i
	}
	return 0
}

// String find a string value
func (sys SysConfig) String(s string) string {
	if val, ok := sys[s]; ok {
		s1, _ := val.(string)
		return s1
	}
	return ""
}

// Marshal sysconfig to bytes again from template f.
func (sys SysConfig) Marshal(f io.Reader, b []byte) {
	s := bufio.NewScanner(f)
	for s.Scan() {
		if strings.HasPrefix(s.Text(), "#") || len(s.Text()) == 0 {
			for _, v := range s.Bytes() {
				b = append(b, v)
			}
			continue
		}
		arr := strings.Split(strings.ReplaceAll(s.Text(), "\"", ""), "=")
		if val, ok := sys[arr[0]]; ok {
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
func (sys SysConfig) Unmarshal(f io.Reader) {
	s := bufio.NewScanner(f)
	for s.Scan() {
		if strings.HasPrefix(s.Text(), "#") || len(s.Text()) == 0 {
			continue
		}
		arr := strings.Split(strings.ReplaceAll(s.Text(), "\"", ""), "=")
		if len(arr) > 1 {
			val, err := strconv.Atoi(arr[1])
			if err != nil {
				b, err1 := strconv.ParseBool(arr[1])
				if err1 != nil {
					sys[arr[0]] = arr[1]
					continue
				}
				sys[arr[0]] = b
				continue
			}
			sys[arr[0]] = val
			continue
		}
		var face interface{}
		sys[arr[0]] = face
	}
}
