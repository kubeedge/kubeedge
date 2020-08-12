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

	// on windows, must convert 'syscall.Stdin' to int
	bs, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return ""
	}

	println() // new line
	return string(bs)
}
