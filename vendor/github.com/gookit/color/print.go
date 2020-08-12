package color

import (
	"fmt"
	"io"
	"log"
)

/*************************************************************
 * print methods(will auto parse color tags)
 *************************************************************/

// Print render color tag and print messages
func Print(a ...interface{}) {
	Fprint(output, a...)
}

// Printf format and print messages
func Printf(format string, a ...interface{}) {
	Fprintf(output, format, a...)
}

// Println messages with new line
func Println(a ...interface{}) {
	Fprintln(output, a...)
}

// Fprint print rendered messages to writer
// Notice: will ignore print error
func Fprint(w io.Writer, a ...interface{}) {
	if isLikeInCmd {
		renderColorCodeOnCmd(func() {
			_, _ = fmt.Fprint(w, Render(a...))
		})
	} else {
		_, _ = fmt.Fprint(w, Render(a...))
	}
}

// Fprintf print format and rendered messages to writer.
// Notice: will ignore print error
func Fprintf(w io.Writer, format string, a ...interface{}) {
	str := fmt.Sprintf(format, a...)
	if isLikeInCmd {
		renderColorCodeOnCmd(func() {
			_, _ = fmt.Fprint(w, ReplaceTag(str))
		})
	} else {
		_, _ = fmt.Fprint(w, ReplaceTag(str))
	}
}

// Fprintln print rendered messages line to writer
// Notice: will ignore print error
func Fprintln(w io.Writer, a ...interface{}) {
	str := formatArgsForPrintln(a)
	if isLikeInCmd {
		renderColorCodeOnCmd(func() {
			_, _ = fmt.Fprintln(w, ReplaceTag(str))
		})
	} else {
		_, _ = fmt.Fprintln(w, ReplaceTag(str))
	}
}

// Lprint passes colored messages to a log.Logger for printing.
// Notice: should be goroutine safe
func Lprint(l *log.Logger, a ...interface{}) {
	if isLikeInCmd {
		renderColorCodeOnCmd(func() {
			l.Print(Render(a...))
		})
	} else {
		l.Print(Render(a...))
	}
}

// Render parse color tags, return rendered string.
// Usage:
//	text := Render("<info>hello</> <cyan>world</>!")
//	fmt.Println(text)
func Render(a ...interface{}) string {
	if len(a) == 0 {
		return ""
	}

	return ReplaceTag(fmt.Sprint(a...))
}

// Sprint parse color tags, return rendered string
func Sprint(args ...interface{}) string {
	return Render(args...)
}

// Sprintf format and return rendered string
func Sprintf(format string, a ...interface{}) string {
	return ReplaceTag(fmt.Sprintf(format, a...))
}

// String alias of the ReplaceTag
func String(s string) string {
	return ReplaceTag(s)
}

// Text alias of the ReplaceTag
func Text(s string) string {
	return ReplaceTag(s)
}
