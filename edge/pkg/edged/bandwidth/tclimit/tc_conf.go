/*
Copyright 2024 The KubeEdge Authors.

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

package tclimit

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"

	"github.com/kubeedge/kubeedge/edge/pkg/edged/bandwidth/consts"
)

const (
	BandwidthRegexCheck = "^[1-9]\\d*(M|Mi|G|Gi)$"
	NumRegex            = "[1-9]\\d*"
	MB                  = "M"
	GB                  = "G"
	KB                  = "Kb" // KBytes
	B                   = "b"
	burstDivisor        = 100
	bytesPerKilobyte    = 1000
)

type TokenBucketFilterConf struct {
	netlinkDeviceName string
	// bit
	rate uint64
	// bit
	burst uint64
}

func parseIngressTrafficControlConf(pod *v1.Pod, networkDeviceName string) (*TokenBucketFilterConf, error) {
	annotations := pod.Annotations
	// MB/GB
	value, ok := annotations[consts.AnnotationIngressBandwidth]
	if !ok {
		return nil, errors.Errorf("pod `%s` ingress bandwidth annotation not found", pod.Name)
	}
	res, err := parseBandwidthParam(networkDeviceName, value)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func parseEgressTrafficControlConf(pod *v1.Pod, networkDeviceName string) (*TokenBucketFilterConf, error) {
	annotations := pod.Annotations
	// MB/GB
	value, ok := annotations[consts.AnnotationEgressBandwidth]
	if !ok {
		return nil, errors.Errorf("pod `%s` egress bandwidth annotation not found", pod.Name)
	}
	res, err := parseBandwidthParam(networkDeviceName, value)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func parseBandwidthParam(networkDeviceName, bandwidthValue string) (*TokenBucketFilterConf, error) {
	rate, burst, err := parseRate(bandwidthValue)
	if err != nil {
		return nil, err
	}
	res := &TokenBucketFilterConf{
		netlinkDeviceName: networkDeviceName,
		rate:              rate,
		burst:             burst,
	}
	if err := validateRateAndBurst(res.rate, res.burst); err != nil {
		return nil, err
	}

	return res, nil
}

func parseRate(value string) (rate, burst uint64, err error) {
	compile := regexp.MustCompile(BandwidthRegexCheck)
	if !compile.MatchString(value) {
		// no value matched
		return 0, 0, errors.Errorf("bandwidth limit annotation value regex match failed,value:%s", value)
	}
	numMatch := regexp.MustCompile(NumRegex)
	nums := numMatch.FindAllString(value, -1)
	if len(nums) != 1 {
		return 0, 0, errors.Errorf("bandwidth limit annotation rate value regex match failed,value:%s", value)
	}
	rateNum := nums[0]
	rateUnit := strings.Split(value, rateNum)[1]
	rateValue, err := strconv.ParseUint(rateNum, 10, 32)
	// refer to the bandwidth source code to process the remaining parameters.
	rate, burst = dealTCParam(rateUnit, rateValue)
	return rate, burst, err
}

// convert processing parameters into bits and calculate burst
func dealTCParam(unit string, rate uint64) (adjustedRate, adjustedBurst uint64) {
	switch unit {
	case KB:
		rate *= burstDivisor
	case MB:
		rate *= bytesPerKilobyte * bytesPerKilobyte
	case GB:
		rate *= bytesPerKilobyte * bytesPerKilobyte * bytesPerKilobyte
	}
	// calculate the optimal burst buffer data area size through rate. The faster the network bandwidth,
	//	the larger the value (rate is 10Mbit, burst must be >10kb)
	burst := rate / burstDivisor
	return rate, burst
}

func validateRateAndBurst(rate, burst uint64) error {
	switch {
	case burst == 0 && rate != 0:
		return fmt.Errorf("if rate is set, burst must also be set")
	case rate == 0 && burst != 0:
		return fmt.Errorf("if burst is set, rate must also be set")
	case burst/8 >= math.MaxUint32:
		return fmt.Errorf("burst cannot be more than 4GB")
	}

	return nil
}
