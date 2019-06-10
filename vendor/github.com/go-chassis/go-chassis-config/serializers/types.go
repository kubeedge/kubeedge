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

//Package serializers created  on 2017/6/22.
package serializers

import (
	"errors"
	"github.com/go-chassis/go-chassis-config/serializers/json"
)

const (
	//JsonEncoder is a variable of type string
	JsonEncoder = `application/json`
)

var availableSerializers map[string]Serializer

//Serializer is a interface which declares encode and decode methods
type Serializer interface {
	Encode(obj interface{}) ([]byte, error)
	Decode(data []byte, obj interface{}) error
}

var _ Serializer = json.JsonSerializer{}

func init() {
	availableSerializers = make(map[string]Serializer)
	availableSerializers[JsonEncoder] = json.JsonSerializer{}
}

// Encode is a convenience wrapper for encoding to a []byte from an Encoder
func Encode(serializersType string, obj interface{}) ([]byte, error) {
	serializer, ok := availableSerializers[serializersType]
	if !ok {
		errorMsg := "serializer" + serializersType + " not avaliable"
		return []byte{}, errors.New(errorMsg)
	}

	data, err := serializer.Encode(obj)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// Decode is a convenience wrapper for decoding data into an Object.
func Decode(serializersType string, data []byte, obj interface{}) error {
	serializer, ok := availableSerializers[serializersType]
	if !ok {
		errorMsg := "serializer" + serializersType + " not avaliable"
		return errors.New(errorMsg)
	}

	err := serializer.Decode(data, obj)
	return err
}
