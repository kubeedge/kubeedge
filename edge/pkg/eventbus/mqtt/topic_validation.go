/*
Copyright 2026 The KubeEdge Authors.

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

package mqtt

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	// maxTopicLength is the maximum allowed length of an MQTT topic name
	// per the MQTT specification (65,535 bytes UTF-8 encoded).
	maxTopicLength = 65535
)

// ValidateMQTTTopicName validates an MQTT topic name according to the MQTT
// specification and security best practices. It returns an error if the topic
// is invalid.
//
// The following checks are performed:
//   - Topic must not be empty
//   - Topic must not exceed 65,535 bytes (UTF-8 encoded)
//   - Topic must not contain the null character (U+0000)
//   - Topic must not contain MQTT wildcard characters ('+' or '#')
//   - Topic must not contain control characters (U+0001–U+001F, U+007F–U+009F)
//   - Topic must be valid UTF-8
func ValidateMQTTTopicName(topic string) error {
	if topic == "" {
		return fmt.Errorf("MQTT topic name must not be empty")
	}

	if len(topic) > maxTopicLength {
		return fmt.Errorf("MQTT topic name exceeds maximum length of %d bytes: got %d bytes", maxTopicLength, len(topic))
	}

	if !utf8.ValidString(topic) {
		return fmt.Errorf("MQTT topic name contains invalid UTF-8 encoding")
	}

	if strings.ContainsRune(topic, '\u0000') {
		return fmt.Errorf("MQTT topic name must not contain null character (U+0000)")
	}

	if strings.ContainsAny(topic, "+#") {
		return fmt.Errorf("MQTT topic name must not contain wildcard characters ('+' or '#'): %q", topic)
	}

	for _, r := range topic {
		if (r >= '\u0001' && r <= '\u001F') || (r >= '\u007F' && r <= '\u009F') {
			return fmt.Errorf("MQTT topic name must not contain control characters: found U+%04X in %q", r, topic)
		}
	}

	return nil
}
