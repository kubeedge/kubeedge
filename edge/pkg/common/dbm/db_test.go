package dbm

import (
	"errors"
	"os"
	"testing"
)

// TestDevice is a dummy struct that is used for model creation in orm.
type TestDevice struct {
	ID          string `orm:"column(id); size(64); pk"`
	Name        string `orm:"column(name); null; type(text)"`
	Description string `orm:"column(description); null; type(text)"`
	State       string `orm:"column(state); null; type(text)"`
	LastOnline  string `orm:"column(last_online); null; type(text)"`
}

// TestRegisterModel is function to test RegisterModel().
func TestRegisterModel(t *testing.T) {
	tests := []struct {
		name       string
		modulename string
		m          interface{}
	}{
		{
			//Failure Case
			name:       "TestRegisterModel-UnregisteredModule",
			modulename: "testmodule",
			m:          "",
		},
		{
			//Success Case
			name:       "TestRegisterModel-RegisteredModule",
			modulename: "twin",
			m:          new(TestDevice),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RegisterModel(test.modulename, test.m)
		})
	}
}

// TestIsNonUniqueNameError is function to test IsNonUniqueNameError().
func TestIsNonUniqueNameError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantBool bool
	}{
		{
			name:     "Suffix-are not unique",
			err:      errors.New("The fields are not unique"),
			wantBool: true,
		},
		{
			name:     "Contains-UNIQUE constraint failed",
			err:      errors.New("Failed-UNIQUE constraint failed"),
			wantBool: true,
		},
		{
			name:     "Contains-constraint failed",
			err:      errors.New("The input constraint failed"),
			wantBool: true,
		},
		{
			name:     "OtherError",
			err:      errors.New("Failed"),
			wantBool: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotBool := IsNonUniqueNameError(test.err)
			if gotBool != test.wantBool {
				t.Errorf("IsNonUniqueError() failed, Got = %v, Want = %v", gotBool, test.wantBool)
			}
		})
	}
}

// TestCleanUp() is functioj to test CleanUp().
func TestCleanup(t *testing.T) {
	t.Run("CleanUpTest", func(t *testing.T) {
		Cleanup()
		_, err := os.Stat(defaultDataSource)
		if os.IsExist(err) {
			t.Error("CleanUp failed ,File not removed")
		}
	})
}

// TestCleanDBFile is function to test cleanDBFile().
func TestCleanDBFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{
			// Checks for the negative scenario of CleanBDFile where an unknown file is passed. Positive scenario is handled in CleanUp().
			name:     "CleanDBFileTest",
			filename: "testfile",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cleanDBFile(test.filename)
			_, err := os.Stat(test.filename)
			if os.IsExist(err) {
				t.Error("CleanUp failed ,File not removed")
			}
		})
	}
}

