// +build windows

package command

import (
	"os"
)

func isExecutable(f os.FileInfo) bool {
	return true
}
