package color

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// Support color:
// 	"TERM=xterm"
// 	"TERM=xterm-vt220"
// 	"TERM=xterm-256color"
// 	"TERM=screen-256color"
// 	"TERM=tmux-256color"
// 	"TERM=rxvt-unicode-256color"
// Don't support color:
// 	"TERM=cygwin"
var specialColorTerms = map[string]bool{
	"screen-256color":       true,
	"tmux-256color":         true,
	"rxvt-unicode-256color": true,
}

// IsConsole Determine whether w is one of stderr, stdout, stdin
func IsConsole(w io.Writer) bool {
	o, ok := w.(*os.File)
	if !ok {
		return false
	}

	fd := o.Fd()

	// fix: cannot use 'o == os.Stdout' to compare
	return fd == uintptr(syscall.Stdout) || fd == uintptr(syscall.Stdin) || fd == uintptr(syscall.Stderr)
}

// IsMSys msys(MINGW64) environment, does not necessarily support color
func IsMSys() bool {
	// like "MSYSTEM=MINGW64"
	if len(os.Getenv("MSYSTEM")) > 0 {
		return true
	}

	return false
}

// IsSupportColor check current console is support color.
//
// Supported:
// 	linux, mac, or windows's ConEmu, Cmder, putty, git-bash.exe
// Not support:
// 	windows cmd.exe, powerShell.exe
func IsSupportColor() bool {
	envTerm := os.Getenv("TERM")
	if strings.Contains(envTerm, "xterm") {
		return true
	}

	// it's special color term
	if _, ok := specialColorTerms[envTerm]; ok {
		return true
	}

	// like on ConEmu software, e.g "ConEmuANSI=ON"
	if os.Getenv("ConEmuANSI") == "ON" {
		return true
	}

	// like on ConEmu software, e.g "ANSICON=189x2000 (189x43)"
	if os.Getenv("ANSICON") != "" {
		return true
	}

	return false
}

// IsSupport256Color render
func IsSupport256Color() bool {
	// "TERM=xterm-256color"
	// "TERM=screen-256color"
	// "TERM=tmux-256color"
	// "TERM=rxvt-unicode-256color"
	return strings.Contains(os.Getenv("TERM"), "256color")
}

// IsSupportTrueColor render. IsSupportRGBColor
func IsSupportTrueColor() bool {
	// "COLORTERM=truecolor"
	return strings.Contains(os.Getenv("COLORTERM"), "truecolor")
}

// Hex2rgb alias of the HexToRgb()
func Hex2rgb(hex string) []int { return HexToRgb(hex) }

// HexToRGB alias of the HexToRgb()
func HexToRGB(hex string) []int { return HexToRgb(hex) }

// HexToRgb hex color string to RGB numbers
// Usage:
// 	rgb := HexToRgb("ccc") // rgb: [204 204 204]
// 	rgb := HexToRgb("aabbcc") // rgb: [170 187 204]
// 	rgb := HexToRgb("#aabbcc") // rgb: [170 187 204]
// 	rgb := HexToRgb("0xad99c0") // rgb: [170 187 204]
func HexToRgb(hex string) (rgb []int) {
	hex = strings.TrimSpace(hex)
	if hex == "" {
		return
	}

	// like from css. eg "#ccc" "#ad99c0"
	if hex[0] == '#' {
		hex = hex[1:]
	}

	hex = strings.ToLower(hex)
	switch len(hex) {
	case 3: // "ccc"
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	case 8: // "0xad99c0"
		hex = strings.TrimPrefix(hex, "0x")
	}

	// recheck
	if len(hex) != 6 {
		return
	}

	// convert string to int64
	if i64, err := strconv.ParseInt(hex, 16, 32); err == nil {
		color := int(i64)
		// parse int
		rgb = make([]int, 3)
		rgb[0] = color >> 16
		rgb[1] = (color & 0x00FF00) >> 8
		rgb[2] = color & 0x0000FF
	}

	return
}

// Rgb2hex alias of the RgbToHex()
func Rgb2hex(rgb []int) string { return RgbToHex(rgb) }

// RgbToHex convert RGB to hex code
// Usage:
//	hex := RgbToHex([]int{170, 187, 204}) // hex: "aabbcc"
func RgbToHex(rgb []int) string {
	hexNodes := make([]string, len(rgb))
	for _, v := range rgb {
		hexNodes = append(hexNodes, strconv.FormatInt(int64(v), 16))
	}

	return strings.Join(hexNodes, "")
}

/*************************************************************
 * helper methods for print
 *************************************************************/

// its Win system. linux windows darwin
// func isWindows() bool {
// 	return runtime.GOOS == "windows"
// }

func doPrint(code string, colors []Color, str string) {
	if isLikeInCmd {
		winPrint(str, colors...)
	} else {
		_, _ = fmt.Fprint(output, RenderString(code, str))
	}
}

func doPrintln(code string, colors []Color, args []interface{}) {
	str := formatArgsForPrintln(args)
	if isLikeInCmd {
		winPrintln(str, colors...)
	} else {
		_, _ = fmt.Fprintln(output, RenderString(code, str))
	}
}

// new implementation, support render full color code on pwsh.exe, cmd.exe
func doPrintV2(code, str string) {
	if isLikeInCmd {
		renderColorCodeOnCmd(func() {
			_, _ = fmt.Fprint(output, RenderString(code, str))
		})
	} else {
		_, _ = fmt.Fprint(output, RenderString(code, str))
	}
}

// new implementation, support render full color code on pwsh.exe, cmd.exe
func doPrintlnV2(code string, args []interface{}) {
	str := formatArgsForPrintln(args)
	if isLikeInCmd {
		renderColorCodeOnCmd(func() {
			_, _ = fmt.Fprintln(output, RenderString(code, str))
		})
	} else {
		_, _ = fmt.Fprintln(output, RenderString(code, str))
	}
}

func stringToArr(str, sep string) (arr []string) {
	str = strings.TrimSpace(str)
	if str == "" {
		return
	}

	ss := strings.Split(str, sep)
	for _, val := range ss {
		if val = strings.TrimSpace(val); val != "" {
			arr = append(arr, val)
		}
	}

	return
}

// if use Println, will add spaces for each arg
func formatArgsForPrintln(args []interface{}) (message string) {
	if ln := len(args); ln == 0 {
		message = ""
	} else if ln == 1 {
		message = fmt.Sprint(args[0])
	} else {
		message = fmt.Sprintln(args...)
		// clear last "\n"
		message = message[:len(message)-1]
	}
	return
}
