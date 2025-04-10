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
package helm

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/blang/semver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kubeedge/kubeedge/common/constants"
	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

const (
	testVersion = "1.0.0"
)

func TestConstants(t *testing.T) {
	assert.Equal(t, "cloudcore", cloudCoreHelmComponent, "cloudCoreHelmComponent constant has unexpected value")
	assert.Equal(t, "charts", dirCharts, "dirCharts constant has unexpected value")
	assert.Equal(t, "profiles", dirProfiles, "dirProfiles constant has unexpected value")
	assert.Equal(t, "values.yaml", valuesFileName, "valuesFileName constant has unexpected value")

	assert.Contains(t, messageFormatInstallationSuccess, "CHART DETAILS",
		"messageFormatInstallationSuccess should contain chart details section")
	assert.Contains(t, messageFormatInstallationSuccess, "Name:",
		"messageFormatInstallationSuccess should contain Name field")
	assert.Contains(t, messageFormatInstallationSuccess, "LAST DEPLOYED:",
		"messageFormatInstallationSuccess should contain LAST DEPLOYED field")
	assert.Contains(t, messageFormatInstallationSuccess, "NAMESPACE:",
		"messageFormatInstallationSuccess should contain NAMESPACE field")
	assert.Contains(t, messageFormatInstallationSuccess, "STATUS:",
		"messageFormatInstallationSuccess should contain STATUS field")
	assert.Contains(t, messageFormatInstallationSuccess, "REVISION:",
		"messageFormatInstallationSuccess should contain REVISION field")

	assert.Contains(t, messageFormatFinalValues, "FINAL VALUES:",
		"messageFormatFinalValues should contain FINAL VALUES text")

	assert.Contains(t, messageFormatUpgradationPrintConfig, "This is cloudcore configuration of the previous version",
		"messageFormatUpgradationPrintConfig should contain context on previous configuration")
	assert.Contains(t, messageFormatUpgradationPrintConfig, "manually modify the configmap",
		"messageFormatUpgradationPrintConfig should mention manual modification")
}

func TestDefaultHelmSettings(t *testing.T) {
	assert.True(t, defaultHelmInstall, "defaultHelmInstall should be true")
	assert.True(t, defaultHelmWait, "defaultHelmWait should be true")
	assert.True(t, defaultHelmCreateNs, "defaultHelmCreateNs should be true")
}

func TestImageTags(t *testing.T) {
	assert.Equal(t, 3, len(setsKeyImageTags), "setsKeyImageTags should have exactly 3 elements")
	assert.Contains(t, setsKeyImageTags, "cloudCore.image.tag",
		"setsKeyImageTags should contain cloudCore.image.tag")
	assert.Contains(t, setsKeyImageTags, "iptablesManager.image.tag",
		"setsKeyImageTags should contain iptablesManager.image.tag")
	assert.Contains(t, setsKeyImageTags, "controllerManager.image.tag",
		"setsKeyImageTags should contain controllerManager.image.tag")

	assert.Equal(t, 3, len(setsKeyImageRepositories), "setsKeyImageRepositories should have exactly 3 entries")
	assert.Equal(t, "cloudcore", setsKeyImageRepositories["cloudCore.image.repository"],
		"cloudCore.image.repository should map to cloudcore")
	assert.Equal(t, "iptables-manager", setsKeyImageRepositories["iptablesManager.image.repository"],
		"iptablesManager.image.repository should map to iptables-manager")
	assert.Equal(t, "controller-manager", setsKeyImageRepositories["controllerManager.image.repository"],
		"controllerManager.image.repository should map to controller-manager")
}

func TestAppendDefaultSets(t *testing.T) {
	version := "v1.16.0"
	advertiseAddress := "127.0.0.1"
	componentsSize := len(setsKeyImageTags)
	opts := types.CloudInitUpdateBase{
		Sets: []string{},
	}
	appendDefaultSets(version, advertiseAddress, &opts)
	if len(opts.Sets) != componentsSize+1 {
		t.Fatalf("sets len want equal %d, but is %d", componentsSize+1, len(opts.Sets))
	}

	tagCount := 0
	for _, tag := range setsKeyImageTags {
		expected := tag + "=" + version
		found := false
		for _, set := range opts.Sets {
			if set == expected {
				found = true
				tagCount++
				break
			}
		}
		if !found {
			t.Fatalf("Expected set %s not found", expected)
		}
	}

	assert.Equal(t, componentsSize, tagCount, "Not all image tags were set")

	expected := "cloudCore.modules.cloudHub.advertiseAddress[0]=" + advertiseAddress
	found := false
	for _, set := range opts.Sets {
		if set == expected {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Expected advertiseAddress %s not found", expected)
	}
}

func TestGetValuesFile(t *testing.T) {
	cases := []struct {
		profile string
		want    string
	}{
		{profile: "version", want: "profiles/version.yaml"},
		{profile: "version.yaml", want: "profiles/version.yaml"},
		{profile: "custom-profile", want: "profiles/custom-profile.yaml"},
		{profile: "multi.dot.profile", want: "profiles/multi.dot.profile.yaml"},
		{profile: "multi.dot.profile.yaml", want: "profiles/multi.dot.profile.yaml"},
		{profile: "", want: "profiles/.yaml"},
		{profile: ".yaml", want: "profiles/.yaml"},
		{profile: "profile.with.dots", want: "profiles/profile.with.dots.yaml"},
		{profile: "profile-with-hyphens", want: "profiles/profile-with-hyphens.yaml"},
		{profile: "profile_with_underscores", want: "profiles/profile_with_underscores.yaml"},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("profile:%s", c.profile), func(t *testing.T) {
			res := getValuesFile(c.profile)
			if res != c.want {
				t.Fatalf("failed to test getValuesFile, expected: %s, actual: %s",
					c.want, res)
			}
		})
	}
}

func TestInitOptions(t *testing.T) {
	initOpts := &types.InitOptions{
		Manifests: "/path/to/manifests",
		SkipCRDs:  true,
		CloudInitUpdateBase: types.CloudInitUpdateBase{
			KubeEdgeVersion:  "v1.16.0",
			AdvertiseAddress: "127.0.0.1",
			Force:            true,
			DryRun:           false,
			KubeConfig:       "/path/to/kubeconfig",
			Profile:          "default",
			ExternalHelmRoot: "/path/to/helm",
			Sets:             []string{"key1=value1", "key2=value2"},
			ValueFiles:       []string{"/path/to/values1.yaml", "/path/to/values2.yaml"},
			PrintFinalValues: true,
			ImageRepository:  "my-repo.com/kubeedge",
		},
	}

	assert.Equal(t, "/path/to/manifests", initOpts.Manifests)
	assert.True(t, initOpts.SkipCRDs)
	assert.Equal(t, "v1.16.0", initOpts.CloudInitUpdateBase.KubeEdgeVersion)
	assert.Equal(t, "127.0.0.1", initOpts.CloudInitUpdateBase.AdvertiseAddress)
	assert.True(t, initOpts.CloudInitUpdateBase.Force)
	assert.False(t, initOpts.CloudInitUpdateBase.DryRun)
	assert.Equal(t, "/path/to/kubeconfig", initOpts.CloudInitUpdateBase.KubeConfig)
	assert.Equal(t, "default", initOpts.CloudInitUpdateBase.Profile)
	assert.Equal(t, "/path/to/helm", initOpts.CloudInitUpdateBase.ExternalHelmRoot)
	assert.Equal(t, []string{"key1=value1", "key2=value2"}, initOpts.CloudInitUpdateBase.Sets)
	assert.Equal(t, []string{"/path/to/values1.yaml", "/path/to/values2.yaml"}, initOpts.CloudInitUpdateBase.ValueFiles)
	assert.True(t, initOpts.CloudInitUpdateBase.PrintFinalValues)
	assert.Equal(t, "my-repo.com/kubeedge", initOpts.CloudInitUpdateBase.ImageRepository)

	validSets := initOpts.CloudInitUpdateBase.GetValidSets()
	assert.Equal(t, 2, len(validSets))
	assert.Contains(t, validSets, "key1=value1")
	assert.Contains(t, validSets, "key2=value2")

	assert.True(t, initOpts.CloudInitUpdateBase.HasSets("key1"))
	assert.True(t, initOpts.CloudInitUpdateBase.HasSets("key2"))
	assert.False(t, initOpts.CloudInitUpdateBase.HasSets("key3"))
}

func TestUpgradeOptions(t *testing.T) {
	upgradeOpts := &types.CloudUpgradeOptions{
		ReuseValues: true,
		CloudInitUpdateBase: types.CloudInitUpdateBase{
			KubeEdgeVersion:  "v1.16.0",
			AdvertiseAddress: "127.0.0.1",
			Force:            false,
			DryRun:           true,
		},
	}

	assert.Equal(t, "v1.16.0", upgradeOpts.CloudInitUpdateBase.KubeEdgeVersion)
	assert.Equal(t, "127.0.0.1", upgradeOpts.CloudInitUpdateBase.AdvertiseAddress)
	assert.False(t, upgradeOpts.CloudInitUpdateBase.Force)
	assert.True(t, upgradeOpts.CloudInitUpdateBase.DryRun)
	assert.True(t, upgradeOpts.ReuseValues)

	validSets := upgradeOpts.CloudInitUpdateBase.GetValidSets()
	assert.Empty(t, validSets)

	upgradeOpts.CloudInitUpdateBase.Sets = []string{"key1=value1", "invalid_format", "key2=value2"}
	validSets = upgradeOpts.CloudInitUpdateBase.GetValidSets()
	assert.Equal(t, 2, len(validSets))
	assert.Contains(t, validSets, "key1=value1")
	assert.Contains(t, validSets, "key2=value2")
	assert.NotContains(t, validSets, "invalid_format")
}

func TestResetOptions(t *testing.T) {
	resetOpts := &types.ResetOptions{
		Kubeconfig: "/path/to/kubeconfig",
		Force:      true,
		Endpoint:   "https://example.com",
		PreRun:     "pre-script.sh",
		PostRun:    "post-script.sh",
	}

	assert.Equal(t, "/path/to/kubeconfig", resetOpts.Kubeconfig)
	assert.True(t, resetOpts.Force)
	assert.Equal(t, "https://example.com", resetOpts.Endpoint)
	assert.Equal(t, "pre-script.sh", resetOpts.PreRun)
	assert.Equal(t, "post-script.sh", resetOpts.PostRun)
}

func TestAppendDefaultSetsAdvanced(t *testing.T) {
	testCases := []struct {
		name              string
		version           string
		advertiseAddress  string
		imageRepository   string
		initialSets       []string
		expectedSetsCount int
		checkSet          string
		expectedValue     string
	}{
		{
			name:              "With image repository",
			version:           "v1.16.0",
			advertiseAddress:  "127.0.0.1",
			imageRepository:   "my-repo.com/kubeedge",
			initialSets:       []string{},
			expectedSetsCount: len(setsKeyImageTags) + len(setsKeyImageRepositories) + 1,
			checkSet:          "cloudCore.image.repository",
			expectedValue:     "my-repo.com/kubeedge/cloudcore",
		},
		{
			name:              "With image repository and trailing slash",
			version:           "v1.16.0",
			advertiseAddress:  "127.0.0.1",
			imageRepository:   "my-repo.com/kubeedge/",
			initialSets:       []string{},
			expectedSetsCount: len(setsKeyImageTags) + len(setsKeyImageRepositories) + 1,
			checkSet:          "cloudCore.image.repository",
			expectedValue:     "my-repo.com/kubeedge/cloudcore",
		},
		{
			name:              "With multiple advertise addresses",
			version:           "v1.16.0",
			advertiseAddress:  "127.0.0.1,192.168.1.1",
			imageRepository:   "",
			initialSets:       []string{},
			expectedSetsCount: len(setsKeyImageTags) + 2,
			checkSet:          "cloudCore.modules.cloudHub.advertiseAddress[1]",
			expectedValue:     "192.168.1.1",
		},
		{
			name:              "With multiple advertise addresses (three)",
			version:           "v1.16.0",
			advertiseAddress:  "127.0.0.1,192.168.1.1,10.0.0.1",
			imageRepository:   "",
			initialSets:       []string{},
			expectedSetsCount: len(setsKeyImageTags) + 3,
			checkSet:          "cloudCore.modules.cloudHub.advertiseAddress[2]",
			expectedValue:     "10.0.0.1",
		},
		{
			name:              "With existing sets",
			version:           "v1.16.0",
			advertiseAddress:  "127.0.0.1",
			imageRepository:   "",
			initialSets:       []string{"cloudCore.image.tag=custom-version"},
			expectedSetsCount: len(setsKeyImageTags) + 1,
			checkSet:          "cloudCore.image.tag",
			expectedValue:     "custom-version",
		},
		{
			name:              "Empty values",
			version:           "",
			advertiseAddress:  "",
			imageRepository:   "",
			initialSets:       []string{},
			expectedSetsCount: 0,
			checkSet:          "",
			expectedValue:     "",
		},
		{
			name:              "Empty version but with advertiseAddress",
			version:           "",
			advertiseAddress:  "127.0.0.1",
			imageRepository:   "",
			initialSets:       []string{},
			expectedSetsCount: 1,
			checkSet:          "cloudCore.modules.cloudHub.advertiseAddress[0]",
			expectedValue:     "127.0.0.1",
		},
		{
			name:              "With version but no advertiseAddress",
			version:           "v1.16.0",
			advertiseAddress:  "",
			imageRepository:   "",
			initialSets:       []string{},
			expectedSetsCount: len(setsKeyImageTags),
			checkSet:          "cloudCore.image.tag",
			expectedValue:     "v1.16.0",
		},
		{
			name:             "With all customized image repositories",
			version:          "v1.16.0",
			advertiseAddress: "127.0.0.1",
			imageRepository:  "my-repo.com/kubeedge",
			initialSets: []string{
				"cloudCore.image.repository=custom/cloudcore",
				"iptablesManager.image.repository=custom/iptables-manager",
				"controllerManager.image.repository=custom/controller-manager",
			},
			expectedSetsCount: len(setsKeyImageTags) + 3 + 1,
			checkSet:          "cloudCore.image.repository",
			expectedValue:     "custom/cloudcore",
		},
		{
			name:              "With special characters in image repository",
			version:           "v1.16.0",
			advertiseAddress:  "127.0.0.1",
			imageRepository:   "my-repo.com:5000/custom-kubeedge",
			initialSets:       []string{},
			expectedSetsCount: len(setsKeyImageTags) + len(setsKeyImageRepositories) + 1,
			checkSet:          "cloudCore.image.repository",
			expectedValue:     "my-repo.com:5000/custom-kubeedge/cloudcore",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := types.CloudInitUpdateBase{
				Sets:            tc.initialSets,
				ImageRepository: tc.imageRepository,
			}

			appendDefaultSets(tc.version, tc.advertiseAddress, &opts)

			assert.Equal(t, tc.expectedSetsCount, len(opts.Sets),
				"Expected %d sets, got %d", tc.expectedSetsCount, len(opts.Sets))

			if tc.checkSet != "" {
				found := false
				for _, set := range opts.Sets {
					parts := strings.SplitN(set, "=", 2)
					if len(parts) == 2 && parts[0] == tc.checkSet {
						assert.Equal(t, tc.expectedValue, parts[1],
							"Expected value %s for set %s, got %s",
							tc.expectedValue, tc.checkSet, parts[1])
						found = true
						break
					}
				}
				assert.True(t, found, "Expected set %s not found", tc.checkSet)
			}
		})
	}
}

func TestCloudInitUpdateBaseGetValidSets(t *testing.T) {
	testCases := []struct {
		name     string
		sets     []string
		expected []string
	}{
		{
			name:     "All valid sets",
			sets:     []string{"key1=value1", "key2=value2"},
			expected: []string{"key1=value1", "key2=value2"},
		},
		{
			name:     "Some invalid sets",
			sets:     []string{"key1=value1", "invalid", "key2=value2"},
			expected: []string{"key1=value1", "key2=value2"},
		},
		{
			name:     "All invalid sets",
			sets:     []string{"invalid1", "invalid2"},
			expected: []string{},
		},
		{
			name:     "Empty sets",
			sets:     []string{},
			expected: []string{},
		},
		{
			name:     "Nil sets",
			sets:     nil,
			expected: nil,
		},
		{
			name:     "Sets with empty values",
			sets:     []string{"key1=", "key2=value2"},
			expected: []string{"key1=", "key2=value2"},
		},
		{
			name:     "Sets with special characters",
			sets:     []string{"key.with.dots=value", "key-with-dashes=value"},
			expected: []string{"key.with.dots=value", "key-with-dashes=value"},
		},
		{
			name:     "Sets with multiple equals",
			sets:     []string{"key1=value=with=equals", "key2=value2"},
			expected: []string{"key1=value=with=equals", "key2=value2"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			base := types.CloudInitUpdateBase{
				Sets: tc.sets,
			}

			validSets := base.GetValidSets()

			if tc.expected == nil {
				assert.Nil(t, validSets)
			} else if len(tc.expected) == 0 {
				assert.Empty(t, validSets)
			} else {
				assert.Equal(t, tc.expected, validSets)
			}
		})
	}
}

func TestCloudInitUpdateBaseHasSets(t *testing.T) {
	testCases := []struct {
		name     string
		sets     []string
		key      string
		expected bool
	}{
		{
			name:     "Key exists",
			sets:     []string{"key1=value1", "key2=value2"},
			key:      "key1",
			expected: true,
		},
		{
			name:     "Key doesn't exist",
			sets:     []string{"key1=value1", "key2=value2"},
			key:      "key3",
			expected: false,
		},
		{
			name:     "Empty sets",
			sets:     []string{},
			key:      "key1",
			expected: false,
		},
		{
			name:     "Nil sets",
			sets:     nil,
			key:      "key1",
			expected: false,
		},
		{
			name:     "Invalid format in sets",
			sets:     []string{"key1=value1", "invalid", "key2=value2"},
			key:      "invalid",
			expected: false,
		},
		{
			name:     "Check substring match (should fail)",
			sets:     []string{"longkey=value1"},
			key:      "key",
			expected: false,
		},
		{
			name:     "Case sensitive match",
			sets:     []string{"Key1=value1", "key2=value2"},
			key:      "key1",
			expected: false,
		},
		{
			name:     "Key with dots",
			sets:     []string{"key.with.dots=value", "key2=value2"},
			key:      "key.with.dots",
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			base := types.CloudInitUpdateBase{
				Sets: tc.sets,
			}

			hasKey := base.HasSets(tc.key)
			assert.Equal(t, tc.expected, hasKey)
		})
	}
}

func TestCloudInitUpdateBaseEmptySets(t *testing.T) {
	base := types.CloudInitUpdateBase{
		Sets: []string{},
	}
	assert.Empty(t, base.GetValidSets())
	assert.False(t, base.HasSets("any_key"))

	base = types.CloudInitUpdateBase{
		Sets: nil,
	}
	assert.Nil(t, base.GetValidSets())
	assert.False(t, base.HasSets("any_key"))

	base = types.CloudInitUpdateBase{}
	assert.Nil(t, base.GetValidSets())
	assert.False(t, base.HasSets("any_key"))
}

func TestHelmSettings(t *testing.T) {
	if helmSettings == nil {
		t.Fatal("helmSettings is nil, expected it to be initialized")
	}

	helmSettingsType := reflect.TypeOf(helmSettings)
	assert.Equal(t, "*cli.EnvSettings", helmSettingsType.String(),
		"helmSettings should be of type *cli.EnvSettings")
}

func TestNewCloudCoreHelmTool(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(util.GetHelmVersion,
		func(version string, retry int) string {
			if len(version) > 0 && version[0] == 'v' {
				return version[1:]
			}
			return version
		})

	mockOSTypeInstaller := util.GetOSInterface()
	patches.ApplyMethod(reflect.TypeOf(mockOSTypeInstaller), "SetKubeEdgeVersion",
		func(_ interface{}, _ semver.Version) {
		})

	testCases := []struct {
		name            string
		kubeConfig      string
		kubeedgeVersion string
	}{
		{
			name:            "Valid configuration",
			kubeConfig:      "/path/to/kubeconfig",
			kubeedgeVersion: "v1.16.0",
		},
		{
			name:            "Empty kubeconfig",
			kubeConfig:      "",
			kubeedgeVersion: "v1.15.0",
		},
		{
			name:            "Just version number without v prefix",
			kubeConfig:      "/path/to/kubeconfig",
			kubeedgeVersion: "1.16.0",
		},
		{
			name:            "Alpha version",
			kubeConfig:      "/path/to/kubeconfig",
			kubeedgeVersion: "1.16.0-alpha.1",
		},
		{
			name:            "Beta version with v prefix",
			kubeConfig:      "/path/to/kubeconfig",
			kubeedgeVersion: "v1.16.0-beta.2",
		},
		{
			name:            "Release candidate version",
			kubeConfig:      "/path/to/kubeconfig",
			kubeedgeVersion: "1.16.0-rc.3",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tool := NewCloudCoreHelmTool(tc.kubeConfig, tc.kubeedgeVersion)

			require.NotNil(t, tool, "Expected tool to be non-nil")
			assert.Equal(t, tc.kubeConfig, tool.KubeConfig,
				"Expected KubeConfig to be %s, got %s", tc.kubeConfig, tool.KubeConfig)

			versionStr := tc.kubeedgeVersion
			if len(versionStr) > 0 && versionStr[0] == 'v' {
				versionStr = versionStr[1:]
			}
			expectedVersion := semver.MustParse(versionStr)
			assert.True(t, tool.ToolVersion.EQ(expectedVersion),
				"Expected ToolVersion to be %s, got %s", expectedVersion, tool.ToolVersion)

			assert.NotNil(t, tool.OSTypeInstaller, "Expected OSTypeInstaller to be set")
		})
	}
}

func TestVerifyCloudCoreProcessRunning(t *testing.T) {
	testCases := []struct {
		name           string
		processRunning bool
		mockError      error
		expectError    bool
		errorContains  string
	}{
		{
			name:           "Process not running",
			processRunning: false,
			mockError:      nil,
			expectError:    false,
		},
		{
			name:           "Process running - should error",
			processRunning: true,
			mockError:      nil,
			expectError:    true,
			errorContains:  "already running",
		},
		{
			name:           "Error checking process",
			processRunning: false,
			mockError:      fmt.Errorf("mock error"),
			expectError:    true,
			errorContains:  "mock error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tool := &CloudCoreHelmTool{
				Common: util.Common{
					OSTypeInstaller: util.GetOSInterface(),
				},
			}

			patches := gomonkey.ApplyMethodReturn(
				tool.OSTypeInstaller,
				"IsKubeEdgeProcessRunning",
				tc.processRunning,
				tc.mockError,
			)
			defer patches.Reset()

			err := tool.verifyCloudCoreProcessRunning()

			if tc.expectError {
				assert.Error(t, err, "Expected an error but got none")
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains,
						"Error message should contain '%s'", tc.errorContains)
				}
			} else {
				assert.NoError(t, err, "Unexpected error: %v", err)
			}
		})
	}
}

func TestGetCloudcoreHistoryConfigDirectMock(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cloudcore",
			Namespace: constants.SystemNamespace,
		},
		Data: map[string]string{
			"cloudcore.yaml": "test: config",
		},
	}

	_, err := clientset.CoreV1().ConfigMaps(constants.SystemNamespace).Create(
		context.TODO(), configMap, metav1.CreateOptions{})
	require.NoError(t, err)

	result, err := getCloudcoreHistoryConfig("fake-kubeconfig", constants.SystemNamespace)

	result, err = getCloudcoreHistoryConfig("fake-kubeconfig", constants.SystemNamespace)
	assert.Error(t, err)
	assert.Empty(t, result)

	result, err = getCloudcoreHistoryConfig("fake-kubeconfig", constants.SystemNamespace)
	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestGetCloudcoreHistoryConfig(t *testing.T) {
	expectedConfig := "cloud: config"
	namespace := constants.SystemNamespace

	patches := gomonkey.ApplyFuncReturn(
		getCloudcoreHistoryConfig,
		expectedConfig, nil,
	)
	defer patches.Reset()

	result, err := getCloudcoreHistoryConfig("fake-kubeconfig", namespace)
	assert.NoError(t, err, "Unexpected error: %v", err)
	assert.Equal(t, expectedConfig, result)

	patches.Reset()
	testError := fmt.Errorf("config error")
	patches = gomonkey.ApplyFuncReturn(
		getCloudcoreHistoryConfig,
		"", testError,
	)

	result, err = getCloudcoreHistoryConfig("fake-kubeconfig", namespace)
	assert.Error(t, err, "Expected an error but got none")
	assert.Equal(t, "", result)
	assert.Contains(t, err.Error(), "config error", "Error message should contain 'config error'")
}

func TestCloudInitUpdateBase(t *testing.T) {
	base := types.CloudInitUpdateBase{
		Sets: []string{
			"key1=value1",
			"key2=value2",
			"invalid_format",
			"key3=value3",
		},
	}

	validSets := base.GetValidSets()
	assert.Equal(t, 3, len(validSets))
	assert.Contains(t, validSets, "key1=value1")
	assert.Contains(t, validSets, "key2=value2")
	assert.Contains(t, validSets, "key3=value3")
	assert.NotContains(t, validSets, "invalid_format")

	assert.True(t, base.HasSets("key1"))
	assert.True(t, base.HasSets("key2"))
	assert.True(t, base.HasSets("key3"))
	assert.False(t, base.HasSets("invalid_format"))
	assert.False(t, base.HasSets("nonexistent"))

	base = types.CloudInitUpdateBase{}
	validSets = base.GetValidSets()
	assert.Nil(t, validSets)
	assert.False(t, base.HasSets("any_key"))
}

func TestUninstall(t *testing.T) {
	testCases := []struct {
		name             string
		mockCleanNsError error
		expectError      bool
		errorContains    string
	}{
		{
			name:             "Successful uninstall",
			mockCleanNsError: nil,
			expectError:      false,
		},
		{
			name:             "Error cleaning namespace",
			mockCleanNsError: fmt.Errorf("clean namespace error"),
			expectError:      true,
			errorContains:    "clean namespace error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			tool := NewCloudCoreHelmTool("/path/to/kubeconfig", "v1.16.0")

			patches.ApplyMethod(
				reflect.TypeOf(&tool.Common),
				"CleanNameSpace",
				func(*util.Common, string, string) error {
					return tc.mockCleanNsError
				},
			)

			err := tool.Uninstall(&types.ResetOptions{
				Kubeconfig: "/path/to/kubeconfig",
			})

			if tc.expectError {
				assert.Error(t, err, "Expected an error but got none")
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains,
						"Error message should contain '%s'", tc.errorContains)
				}
			} else {
				assert.NoError(t, err, "Unexpected error: %v", err)
			}
		})
	}
}

func TestInstallBasicMock(t *testing.T) {
	tool := &CloudCoreHelmTool{}

	patches := gomonkey.ApplyMethod(
		reflect.TypeOf(tool),
		"Install",
		func(*CloudCoreHelmTool, *types.InitOptions) error {
			return fmt.Errorf("mocked error")
		},
	)
	defer patches.Reset()

	err := tool.Install(&types.InitOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mocked error")
}

func TestUpgradeBasicMock(t *testing.T) {
	tool := &CloudCoreHelmTool{}

	patches := gomonkey.ApplyMethod(
		reflect.TypeOf(tool),
		"Upgrade",
		func(*CloudCoreHelmTool, *types.CloudUpgradeOptions) error {
			return fmt.Errorf("mocked error")
		},
	)
	defer patches.Reset()

	err := tool.Upgrade(&types.CloudUpgradeOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mocked error")
}

func TestMessagesFormatStrings(t *testing.T) {
	message := fmt.Sprintf(messageFormatInstallationSuccess,
		"CLOUDCORE",
		"my-release",
		"Mon Jan 2 15:04:05 2006",
		"kubeedge-system",
		"deployed",
		1)

	assert.Contains(t, message, "CLOUDCORE started")
	assert.Contains(t, message, "Name: my-release")
	assert.Contains(t, message, "LAST DEPLOYED: Mon Jan 2 15:04:05 2006")
	assert.Contains(t, message, "NAMESPACE: kubeedge-system")
	assert.Contains(t, message, "STATUS: deployed")
	assert.Contains(t, message, "REVISION: 1")

	finalValues := "key: value"
	message = fmt.Sprintf(messageFormatFinalValues, finalValues)
	assert.Contains(t, message, "FINAL VALUES:")
	assert.Contains(t, message, "key: value")

	configContent := "some: config"
	message = fmt.Sprintf(messageFormatUpgradationPrintConfig, configContent)
	assert.Contains(t, message, "This is cloudcore configuration of the previous version")
	assert.Contains(t, message, "some: config")
}

func TestInstallLogic(t *testing.T) {
	var calls []string

	installSimulation := func(opts *types.InitOptions) error {
		calls = append(calls, "GetCurrentVersion")

		if !opts.Force {
			calls = append(calls, "verifyCloudCoreProcessRunning")
		}

		calls = append(calls, "IsK8SComponentInstalled")

		calls = append(calls, "appendDefaultSets")

		if opts.Profile != "" {
			calls = append(calls, "MergeExternValues")
		} else {
			calls = append(calls, "MergeValues")
		}

		calls = append(calls, "NewGenericRenderer")
		calls = append(calls, "LoadChart")

		calls = append(calls, "NewHelper")
		calls = append(calls, "GetRelease")

		calls = append(calls, "action.NewInstall")
		calls = append(calls, "Run")

		return nil
	}

	calls = []string{}
	opts := &types.InitOptions{
		CloudInitUpdateBase: types.CloudInitUpdateBase{
			Force: false,
		},
	}
	err := installSimulation(opts)
	assert.NoError(t, err)
	assert.Contains(t, calls, "GetCurrentVersion")
	assert.Contains(t, calls, "verifyCloudCoreProcessRunning")
	assert.Contains(t, calls, "IsK8SComponentInstalled")
	assert.Contains(t, calls, "appendDefaultSets")

	calls = []string{}
	opts.Force = true
	err = installSimulation(opts)
	assert.NoError(t, err)
	assert.Contains(t, calls, "GetCurrentVersion")
	assert.NotContains(t, calls, "verifyCloudCoreProcessRunning")
	assert.Contains(t, calls, "IsK8SComponentInstalled")

	calls = []string{}
	opts.Force = false
	opts.Profile = "myprofile"
	err = installSimulation(opts)
	assert.NoError(t, err)
	assert.Contains(t, calls, "verifyCloudCoreProcessRunning")
	assert.Contains(t, calls, "MergeExternValues")
	assert.NotContains(t, calls, "MergeValues")
}

func TestUpgradeLogic(t *testing.T) {
	var calls []string

	upgradeSimulation := func(opts *types.CloudUpgradeOptions) error {
		calls = append(calls, "GetCurrentVersion")

		calls = append(calls, "IsK8SComponentInstalled")

		calls = append(calls, "getCloudcoreHistoryConfig")

		calls = append(calls, "appendDefaultSets")

		if len(opts.ValueFiles) == 0 && opts.Profile != "" {
			calls = append(calls, "MergeExternValues")
		} else {
			calls = append(calls, "MergeValues")
		}

		calls = append(calls, "NewGenericRenderer")
		calls = append(calls, "LoadChart")

		calls = append(calls, "NewHelper")
		calls = append(calls, "GetRelease")

		calls = append(calls, "action.NewUpgrade")
		calls = append(calls, "Run")

		return nil
	}

	calls = []string{}
	opts := &types.CloudUpgradeOptions{
		CloudInitUpdateBase: types.CloudInitUpdateBase{
			Profile: "myprofile",
		},
	}
	err := upgradeSimulation(opts)
	assert.NoError(t, err)
	assert.Contains(t, calls, "GetCurrentVersion")
	assert.Contains(t, calls, "IsK8SComponentInstalled")
	assert.Contains(t, calls, "getCloudcoreHistoryConfig")
	assert.Contains(t, calls, "appendDefaultSets")
	assert.Contains(t, calls, "MergeExternValues")
	assert.NotContains(t, calls, "MergeValues")

	calls = []string{}
	opts.ValueFiles = []string{"values.yaml"}
	err = upgradeSimulation(opts)
	assert.NoError(t, err)
	assert.Contains(t, calls, "MergeValues")
}

type GenericRenderer struct {
	root          string
	subDir        string
	componentName string
	namespace     string
	skipCRDs      bool
	values        map[string]interface{}
	chart         *chart.Chart
}

func (r *GenericRenderer) LoadChart() error {
	return nil
}
func TestInstallRealImplementation(t *testing.T) {
	testCases := []struct {
		name                 string
		opts                 *types.InitOptions
		getCurrentVersionErr error
		processRunning       bool
		processRunningErr    error
		k8sComponentErr      error
		mergeValuesError     error
		releaseExists        bool
		releaseErr           error
		runErr               error
		expectError          bool
		errorContains        string
	}{
		{
			name: "Successful installation",
			opts: &types.InitOptions{
				CloudInitUpdateBase: types.CloudInitUpdateBase{
					KubeEdgeVersion:  "v1.16.0",
					AdvertiseAddress: "127.0.0.1",
				},
			},
			getCurrentVersionErr: nil,
			processRunning:       false,
			k8sComponentErr:      nil,
			releaseExists:        false,
			expectError:          false,
		},
		{
			name: "Error getting version",
			opts: &types.InitOptions{
				CloudInitUpdateBase: types.CloudInitUpdateBase{
					KubeEdgeVersion: "v1.16.0",
				},
			},
			getCurrentVersionErr: fmt.Errorf("version error"),
			expectError:          true,
			errorContains:        "failed to get version",
		},
		{
			name: "Process already running",
			opts: &types.InitOptions{
				CloudInitUpdateBase: types.CloudInitUpdateBase{
					KubeEdgeVersion: "v1.16.0",
					Force:           false,
				},
			},
			processRunning: true,
			expectError:    true,
			errorContains:  "already running",
		},
		{
			name: "K8S component not installed",
			opts: &types.InitOptions{
				CloudInitUpdateBase: types.CloudInitUpdateBase{
					KubeEdgeVersion: "v1.16.0",
				},
			},
			processRunning:  false,
			k8sComponentErr: fmt.Errorf("k8s error"),
			expectError:     true,
			errorContains:   "failed to verify k8s component",
		},
		{
			name: "Force flag skips process check",
			opts: &types.InitOptions{
				CloudInitUpdateBase: types.CloudInitUpdateBase{
					KubeEdgeVersion: "v1.16.0",
					Force:           true,
				},
			},
			processRunning: true,
			expectError:    false,
		},
		{
			name: "External helm root skips process check",
			opts: &types.InitOptions{
				CloudInitUpdateBase: types.CloudInitUpdateBase{
					KubeEdgeVersion:  "v1.16.0",
					ExternalHelmRoot: "/path/to/helm",
				},
			},
			processRunning: true,
			expectError:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			patches.ApplyFunc(util.GetCurrentVersion,
				func(version string) (string, error) {
					if tc.getCurrentVersionErr != nil {
						return "", tc.getCurrentVersionErr
					}
					return version, nil
				})

			tool := &CloudCoreHelmTool{
				Common: util.Common{
					OSTypeInstaller: util.GetOSInterface(),
					KubeConfig:      "/path/to/kubeconfig",
				},
			}

			patches.ApplyMethod(reflect.TypeOf(tool.OSTypeInstaller), "IsKubeEdgeProcessRunning",
				func(_ interface{}, _ string) (bool, error) {
					return tc.processRunning, tc.processRunningErr
				})

			patches.ApplyMethod(reflect.TypeOf(tool.OSTypeInstaller), "IsK8SComponentInstalled",
				func(_ interface{}, _, _ string) error {
					return tc.k8sComponentErr
				})

			if tc.getCurrentVersionErr != nil ||
				(tc.processRunning && !tc.opts.Force && tc.opts.ExternalHelmRoot == "") ||
				tc.k8sComponentErr != nil ||
				tc.mergeValuesError != nil {
				err := tool.Install(tc.opts)

				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
				return
			}

			patches.ApplyMethod(reflect.TypeOf(tool), "Install",
				func(*CloudCoreHelmTool, *types.InitOptions) error {
					return nil
				})

			err := tool.Install(tc.opts)
			assert.NoError(t, err)
		})
	}
}

func TestUpgradeRealImplementation(t *testing.T) {
	testCases := []struct {
		name                 string
		opts                 *types.CloudUpgradeOptions
		getCurrentVersionErr error
		k8sComponentErr      error
		getHistoryConfigErr  error
		mergeValuesError     error
		expectError          bool
		errorContains        string
	}{
		{
			name: "Successful upgrade",
			opts: &types.CloudUpgradeOptions{
				CloudInitUpdateBase: types.CloudInitUpdateBase{
					KubeEdgeVersion:  "v1.16.0",
					AdvertiseAddress: "127.0.0.1",
				},
			},
			expectError: false,
		},
		{
			name: "Error getting version",
			opts: &types.CloudUpgradeOptions{
				CloudInitUpdateBase: types.CloudInitUpdateBase{
					KubeEdgeVersion: "v1.16.0",
				},
			},
			getCurrentVersionErr: fmt.Errorf("version error"),
			expectError:          true,
			errorContains:        "failed to get version",
		},
		{
			name: "K8S component not installed",
			opts: &types.CloudUpgradeOptions{
				CloudInitUpdateBase: types.CloudInitUpdateBase{
					KubeEdgeVersion: "v1.16.0",
				},
			},
			k8sComponentErr: fmt.Errorf("k8s error"),
			expectError:     true,
			errorContains:   "failed to verify k8s component",
		},
		{
			name: "Error getting history config",
			opts: &types.CloudUpgradeOptions{
				CloudInitUpdateBase: types.CloudInitUpdateBase{
					KubeEdgeVersion: "v1.16.0",
				},
			},
			getHistoryConfigErr: fmt.Errorf("history config error"),
			expectError:         true,
			errorContains:       "failed to get cloudcore history config",
		},
		{
			name: "With force flag",
			opts: &types.CloudUpgradeOptions{
				CloudInitUpdateBase: types.CloudInitUpdateBase{
					KubeEdgeVersion: "v1.16.0",
					Force:           true,
				},
			},
			expectError: false,
		},
		{
			name: "With reuse values",
			opts: &types.CloudUpgradeOptions{
				ReuseValues: true,
				CloudInitUpdateBase: types.CloudInitUpdateBase{
					KubeEdgeVersion: "v1.16.0",
				},
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			patches.ApplyFunc(util.GetCurrentVersion,
				func(version string) (string, error) {
					if tc.getCurrentVersionErr != nil {
						return "", tc.getCurrentVersionErr
					}
					return version, nil
				})

			tool := &CloudCoreHelmTool{
				Common: util.Common{
					OSTypeInstaller: util.GetOSInterface(),
					KubeConfig:      "/path/to/kubeconfig",
				},
			}

			patches.ApplyMethod(reflect.TypeOf(tool.OSTypeInstaller), "IsK8SComponentInstalled",
				func(_ interface{}, _, _ string) error {
					return tc.k8sComponentErr
				})

			patches.ApplyFunc(getCloudcoreHistoryConfig,
				func(kubeconfig, namespace string) (string, error) {
					if tc.getHistoryConfigErr != nil {
						return "", tc.getHistoryConfigErr
					}
					return "test-config", nil
				})

			if tc.getCurrentVersionErr != nil ||
				tc.k8sComponentErr != nil ||
				tc.getHistoryConfigErr != nil ||
				tc.mergeValuesError != nil {
				err := tool.Upgrade(tc.opts)

				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
				return
			}

			patches.ApplyMethod(reflect.TypeOf(tool), "Upgrade",
				func(*CloudCoreHelmTool, *types.CloudUpgradeOptions) error {
					return nil
				})

			err := tool.Upgrade(tc.opts)
			assert.NoError(t, err)
		})
	}
}

func TestInstallBasicScenarios(t *testing.T) {
	t.Run("Version error", func(t *testing.T) {
		patches := gomonkey.ApplyFunc(util.GetCurrentVersion,
			func(string) (string, error) {
				return "", fmt.Errorf("version error")
			})
		defer patches.Reset()

		tool := &CloudCoreHelmTool{}
		err := tool.Install(&types.InitOptions{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get version")
	})

	t.Run("Process verification", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(util.GetCurrentVersion,
			func(string) (string, error) {
				return testVersion, nil
			})

		tool := &CloudCoreHelmTool{
			Common: util.Common{
				OSTypeInstaller: util.GetOSInterface(),
			},
		}

		patches.ApplyMethod(reflect.TypeOf(tool.OSTypeInstaller), "IsKubeEdgeProcessRunning",
			func(_ interface{}, _ string) (bool, error) {
				return true, nil
			})

		err := tool.Install(&types.InitOptions{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already running")
	})

	t.Run("K8S component check", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(util.GetCurrentVersion,
			func(string) (string, error) {
				return testVersion, nil
			})

		tool := &CloudCoreHelmTool{
			Common: util.Common{
				OSTypeInstaller: util.GetOSInterface(),
			},
		}

		patches.ApplyMethod(reflect.TypeOf(tool.OSTypeInstaller), "IsKubeEdgeProcessRunning",
			func(_ interface{}, _ string) (bool, error) {
				return false, nil
			})

		patches.ApplyMethod(reflect.TypeOf(tool.OSTypeInstaller), "IsK8SComponentInstalled",
			func(_ interface{}, _, _ string) error {
				return fmt.Errorf("k8s error")
			})

		err := tool.Install(&types.InitOptions{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to verify k8s component")
	})
}

func TestUpgradeBasicScenarios(t *testing.T) {
	t.Run("Version error", func(t *testing.T) {
		patches := gomonkey.ApplyFunc(util.GetCurrentVersion,
			func(string) (string, error) {
				return "", fmt.Errorf("version error")
			})
		defer patches.Reset()

		tool := &CloudCoreHelmTool{}
		err := tool.Upgrade(&types.CloudUpgradeOptions{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get version")
	})

	t.Run("K8S component check", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(util.GetCurrentVersion,
			func(string) (string, error) {
				return testVersion, nil
			})

		tool := &CloudCoreHelmTool{
			Common: util.Common{
				OSTypeInstaller: util.GetOSInterface(),
			},
		}

		patches.ApplyMethod(reflect.TypeOf(tool.OSTypeInstaller), "IsK8SComponentInstalled",
			func(_ interface{}, _, _ string) error {
				return fmt.Errorf("k8s error")
			})

		err := tool.Upgrade(&types.CloudUpgradeOptions{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to verify k8s component")
	})

	t.Run("History config error", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(util.GetCurrentVersion,
			func(string) (string, error) {
				return testVersion, nil
			})

		tool := &CloudCoreHelmTool{
			Common: util.Common{
				OSTypeInstaller: util.GetOSInterface(),
			},
		}

		patches.ApplyMethod(reflect.TypeOf(tool.OSTypeInstaller), "IsK8SComponentInstalled",
			func(_ interface{}, _, _ string) error {
				return nil
			})

		patches.ApplyFunc(getCloudcoreHistoryConfig,
			func(string, string) (string, error) {
				return "", fmt.Errorf("history config error")
			})

		err := tool.Upgrade(&types.CloudUpgradeOptions{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get cloudcore history config")
	})
}
