package util

import (
	"errors"
	"fmt"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"strings"
	"testing"
)

func TestSpliceErrors(t *testing.T) {
	err1 := errors.New("this is error 1")
	err2 := errors.New("this is error 2")
	err3 := errors.New("this is error 3")

	const head = "[\n"
	var line1 = fmt.Sprintf("  %s\n", err1)
	var line2 = fmt.Sprintf("  %s\n", err2)
	var line3 = fmt.Sprintf("  %s\n", err3)
	const tail = "]\n"

	errList := field.ErrorList{}
	errList = append(errList,
		field.InternalError(field.NewPath("test path 1"), err1),
		field.InternalError(field.NewPath("test path 2"), err2),
		field.InternalError(field.NewPath("test path 3"), err3),
	)

	// case 1: none error
	if SpliceErrors("") != "" ||
		SpliceErrors("test text") != "" ||
		SpliceErrors([]int{1}) != "" ||
		SpliceErrors([]int{1, 2, 3}) != "" {
		t.Error("the func format the none error unexpected")
		return
	}

	// case 2: single error
	singleOutput := SpliceErrors(err1)
	if singleOutput != err1.Error() {
		t.Error("the func format the single error unexpected")
		return
	}

	// case 3-1: single element error slice
	sliceOutput := SpliceErrors([]error{err1})
	if sliceOutput != err1.Error() {
		t.Error("the func format the single element error slice unexpected")
		return
	}

	// case 3-2: multiple elements error slice
	sliceOutput = SpliceErrors([]error{err1, err2, err3})
	if strings.Index(sliceOutput, head) != 0 ||
		strings.Index(sliceOutput, line1) != len(head) ||
		strings.Index(sliceOutput, line2) != len(head+line1) ||
		strings.Index(sliceOutput, line3) != len(head+line1+line2) ||
		strings.Index(sliceOutput, tail) != len(head+line1+line2+line3) {
		t.Error("the func format the multiple elements error slice unexpected")
		return
	}

	// case 4: single complex error
	if SpliceErrors(errList[0]) != errList[0].Error() {
		t.Error("the func format the single complex error unexpected")
		return
	}

	// case 5-1: single element complex error slice
	if SpliceErrors(field.ErrorList{errList[0]}) != errList[0].Error() {
		t.Error("the func format the single element complex error slice unexpected")
		return
	}

	// case 5-2: multiple element complex error slice
	cpx1 := fmt.Sprintf("  %s\n", errList[0].Error())
	cpx2 := fmt.Sprintf("  %s\n", errList[1].Error())
	cpx3 := fmt.Sprintf("  %s\n", errList[2].Error())
	complexOutput := SpliceErrors(errList)
	if strings.Index(complexOutput, head) != 0 ||
		strings.Index(complexOutput, cpx1) != len(head) ||
		strings.Index(complexOutput, cpx2) != len(head+cpx1) ||
		strings.Index(complexOutput, cpx3) != len(head+cpx1+cpx2) ||
		strings.Index(complexOutput, tail) != len(head+cpx1+cpx2+cpx3) {
		t.Error("the func format the multiple element complex error slice unexpected")
		return
	}
}
