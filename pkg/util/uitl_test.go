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
	err1 := errors.New("This is error 1. ")
	err2 := errors.New("This is error 2. ")
	err3 := errors.New("This is error 3. ")

	// case 1: slice stderr
	const sliceHead = "error: [\n"
	var sliceLine1 = fmt.Sprintf("  %s\n", err1)
	var sliceLine2 = fmt.Sprintf("  %s\n", err2)
	var sliceLine3 = fmt.Sprintf("  %s\n", err3)
	const sliceTail = "]\n"

	slice := []error{err1, err2, err3}
	outSlice := helpGenStderrString(func() {
		PrintByLine(os.Stderr, slice)
	})

	if strings.Index(outSlice, sliceHead) != 0 ||
		strings.Index(outSlice, sliceLine1) != len(sliceHead) ||
		strings.Index(outSlice, sliceLine2) != len(sliceHead+sliceLine1) ||
		strings.Index(outSlice, sliceLine3) != len(sliceHead+sliceLine1+sliceLine2) ||
		strings.Index(outSlice, sliceTail) != len(sliceHead+sliceLine1+sliceLine2+sliceLine3) {
		t.Error("The func format the slice errors unexpected.")
		return
	}

	// case 2: map stdout
	m := map[int]error{1: err1, 2: err2}
	outMap := helpGenStdoutString(func() {
		PrintByLine(os.Stdout, m)
	})
	mapHead := "[\n"
	var mapMiddle []string
	for k, v := range m {
		mapMiddle = append(mapMiddle, fmt.Sprintf("  %v: %v\n", k, v))
	}
	mapTail := "]\n"
	if strings.Index(outMap, mapHead) != 0 ||
		strings.Index(outMap, mapTail) != len(mapHead+mapMiddle[0]+mapMiddle[1]) ||
		(strings.Index(outMap, mapMiddle[0]) != len(mapHead) && strings.Index(outMap, mapMiddle[0]) != len(mapHead+mapMiddle[1])) {
		t.Error("The func format the map errors unexpected.")
		return
	}

	// case 3: error stderr
	outError := helpGenStderrString(func() {
		PrintByLine(os.Stderr, err1)
	})
	if outError != fmt.Sprintf("error: %v\n", err1) {
		t.Error("The func format the single error unexpected.")
		return
	}
}

func helpGenStderrString(f func()) string {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	f()

	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	w.Close()
	os.Stderr = old
	out := <-outC

	return out
}

func helpGenStdoutString(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	w.Close()
	os.Stdout = old
	out := <-outC

	return out
}
