package util

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func TestPrintByLine(t *testing.T) {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	errs := []error{
		errors.New("This is error 1. "),
		errors.New("This is error 2. "),
		errors.New("This is error 3. "),
	}
	PrintByLine(os.Stderr, errs)

	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	w.Close()
	os.Stderr = old
	out := <-outC

	// The stderr should have "error: [\n" firstly, then repeat "  {each item}\n" for three times, and "]\n" in the end.
	const head = "error: [\n"
	var errLine1 = fmt.Sprintf("  %s\n", errs[0])
	var errLine2 = fmt.Sprintf("  %s\n", errs[1])
	var errLine3 = fmt.Sprintf("  %s\n", errs[2])
	const tail = "]\n"

	indexHead := strings.Index(out, head)
	if indexHead != 0 {
		t.Error("The func format the msg unexpected.")
		return
	}

	indexErr1 := strings.Index(out, errLine1)
	if indexErr1 != len(head) {
		t.Error("The func format the msg unexpected.")
		return
	}

	indexErr2 := strings.Index(out, errLine2)
	if indexErr2 != len(head)+len(errLine1) {
		t.Error("The func format the msg unexpected.")
		return
	}

	indexErr3 := strings.Index(out, errLine3)
	if indexErr3 != len(head)+len(errLine1)+len(errLine2) {
		t.Error("The func format the msg unexpected.")
		return
	}

	indexTail := strings.Index(out, tail)
	if indexTail != len(head)+len(errLine1)+len(errLine2)+len(errLine3) {
		t.Error("The func format the msg unexpected.")
		return
	}
}
