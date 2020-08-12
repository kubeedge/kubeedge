// +build !windows

package interact

import (
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
)

// ReadPassword from terminal
func ReadPassword(question ...string) string {
	if len(question) > 0 {
		print(question[0])
	} else {
		print("Enter Password: ")
	}

	bs, err := terminal.ReadPassword(syscall.Stdin)
	if err != nil {
		return ""
	}

	println() // new line
	return string(bs)
}
