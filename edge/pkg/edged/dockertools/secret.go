/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

@CHANGELOG
KubeEdge Authors: To create mini-kubelet for edge deployment scenario,
This file is derived from K8S Kubelet code with pruned structures and interfaces
and changed most of the realization.
1. Functions from k8s.io\kubernetes\pkg\credentialprovider\config.go related to DockerConfigEntry
   are re-used here
2. Function from k8s.io\kubernetes\pkg\credentialprovider\secrets\secrets.go is derived here
*/

package dockertools

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"k8s.io/api/core/v1"
)

// DockerConfig represents the config file used by the docker CLI.
// This config that represents the credentials that should be used
// when pulling images from specific image repositories.
type DockerConfig map[string]DockerConfigEntry

//DockerConfigEntry defines
type DockerConfigEntry struct {
	Username string
	Password string
	Email    string
}

//DockerConfigJSON is json struct for docker configuration
type DockerConfigJSON struct {
	Auths DockerConfig `json:"auths"`
	// +optional
	HTTPHeaders map[string]string `json:"HttpHeaders,omitempty"`
}

type dockerConfigEntryWithAuth struct {
	// +optional
	Username string `json:"username,omitempty"`
	// +optional
	Password string `json:"password,omitempty"`
	// +optional
	Email string `json:"email,omitempty"`
	// +optional
	Auth string `json:"auth,omitempty"`
}

//UnmarshalJSON is to unmarshal json object
func (ident *DockerConfigEntry) UnmarshalJSON(data []byte) error {
	var tmp dockerConfigEntryWithAuth
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}

	ident.Username = tmp.Username
	ident.Password = tmp.Password
	ident.Email = tmp.Email

	if len(tmp.Auth) == 0 {
		return nil
	}

	ident.Username, ident.Password, err = decodeDockerConfigFieldAuth(tmp.Auth)
	return err
}

//MarshalJSON is to encode byte array to json
func (ident DockerConfigEntry) MarshalJSON() ([]byte, error) {
	toEncode := dockerConfigEntryWithAuth{ident.Username, ident.Password, ident.Email, ""}
	toEncode.Auth = encodeDockerConfigFieldAuth(ident.Username, ident.Password)

	return json.Marshal(toEncode)
}

// decodeDockerConfigFieldAuth deserializes the "auth" field from dockercfg into a
// username and a password. The format of the auth field is base64(<username>:<password>).
func decodeDockerConfigFieldAuth(field string) (username, password string, err error) {
	decoded, err := base64.StdEncoding.DecodeString(field)
	if err != nil {
		return
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		err = fmt.Errorf("unable to parse auth field")
		return
	}

	username = parts[0]
	password = parts[1]

	return
}

func encodeDockerConfigFieldAuth(username, password string) string {
	fieldValue := username + ":" + password

	return base64.StdEncoding.EncodeToString([]byte(fieldValue))
}

func getDockerConfigEntryFromSecret(passedSecrets []v1.Secret) ([]DockerConfigEntry, error) {
	dockerConfigEntrys := []DockerConfigEntry{}
	for _, passedSecret := range passedSecrets {
		if dockerConfigJSONBytes, dockerConfigJSONExists := passedSecret.Data[v1.DockerConfigJsonKey]; passedSecret.Type == v1.SecretTypeDockerConfigJson && dockerConfigJSONExists && (len(dockerConfigJSONBytes) > 0) {
			dockerConfigJSON := DockerConfigJSON{}
			if err := json.Unmarshal(dockerConfigJSONBytes, &dockerConfigJSON); err != nil {
				return nil, err
			}
			for _, entry := range dockerConfigJSON.Auths {
				dockerConfigEntrys = append(dockerConfigEntrys, entry)
			}
		}
	}
	return dockerConfigEntrys, nil
}
