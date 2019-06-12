/*
 * Copyright 2017 Huawei Technologies Co., Ltd
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
//Created by on 2017/6/22.

//Package json is used for marshalling and unmarshalling
package json

import (
	jsonwrapper "encoding/json"
	"errors"
)

//JsonSerializer is a empty struct
type JsonSerializer struct{}

//Decode - Unmarshal unmarshaling data
func (js JsonSerializer) Decode(data []byte, v interface{}) error {
	var err error
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("Invalid request ")
		}

	}()

	err = jsonwrapper.Unmarshal(data, v)
	return err
}

//Encode - Marshal marshaling data
func (js JsonSerializer) Encode(v interface{}) ([]byte, error) {
	var (
		data []byte
		err  error
	)

	defer func() {
		if r := recover(); r != nil {
			err = errors.New("Invalid request ")
		}

	}()

	data, err = jsonwrapper.Marshal(v)

	return data, err
}
