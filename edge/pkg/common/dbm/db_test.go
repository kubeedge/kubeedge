/*
Copyright 2019 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
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
		name           string
		moduleName     string
		modelStructure interface{}
	}{
		{
			//Failure Case
			name:           "TestRegisterModel-UnregisteredModule",
			moduleName:     "testmodule",
			modelStructure: "",
		},
		{
			//Success Case
			name:           "TestRegisterModel-RegisteredModule",
			moduleName:     "twin",
			modelStructure: new(TestDevice),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RegisterModel(test.moduleName, test.modelStructure)
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

// TestCleanUp() is function to test CleanUp().
func TestCleanup(t *testing.T) {
	t.Run("CleanUpTest", func(t *testing.T) {
		Cleanup()
		_, err := os.Stat(dataSource)
		if os.IsExist(err) {
			t.Errorf("CleanUp failed ,File not removed")
		}
	})
}

// TestCleanDBFile is function to test cleanDBFile().
func TestCleanDBFile(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
	}{
		{
			// Checks for the negative scenario of CleanBDFile where an unknown file is passed. Positive scenario is handled in CleanUp().
			name:     "CleanDBFileTest",
			fileName: "testfile",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cleanDBFile(test.fileName)
			_, err := os.Stat(test.fileName)
			if os.IsExist(err) {
				t.Errorf("CleanUp failed ,file exist")
			}
		})
	}
}
