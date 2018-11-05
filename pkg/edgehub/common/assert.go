package hubclient

import (
	"testing"
)

// AssertNoError triggers testing error if the passed-in err is not nil.
func AssertNoError(t *testing.T, err error, errMsg string) {
	if err != nil {
		t.Errorf("%s, error: %s", errMsg, err.Error())
	}
}

// AssertTrue triggers testing error if the passed-in is true.
func AssertTrue(t *testing.T, value bool, errMsg string) {
	if !value {
		t.Errorf("error: %s", errMsg)
	}
}

// AssertStringEqual triggers testing error if the expect and actual string are not the same.
func AssertStringEqual(t *testing.T, expect, actual, errMsg string) {
	if expect != actual {
		t.Errorf("%s, expect: \"%s\", actual: \"%s\"", errMsg, expect, actual)
	}
}

// AssertIntEqual triggers testing error if the expect and actual int value are not the same.
func AssertIntEqual(t *testing.T, expect, actual, errMsg string) {
	if expect != actual {
		t.Errorf("%s, expect: \"%s\", actual: \"%s\"", errMsg, expect, actual)
	}
}
