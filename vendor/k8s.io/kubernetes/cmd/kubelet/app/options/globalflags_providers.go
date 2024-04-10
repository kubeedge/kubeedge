//go:build !providerless
// +build !providerless

/*
Copyright 2020 The Kubernetes Authors.

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

package options

import (
	"github.com/spf13/pflag"
)

func addLegacyCloudProviderCredentialProviderFlags(global, local *pflag.FlagSet) {
	// TODO(#58034): This is not a static file, so it's not quite as straightforward as --google-json-key.
	// We need to figure out how ACR users can dynamically provide pull credentials before we can deprecate this.
	pflagRegister(global, local, "azure-container-registry-config")
	local.MarkDeprecated("azure-container-registry-config", "Use --image-credential-provider-config and --image-credential-provider-bin-dir to setup acr credential provider instead. Will be removed in a future release.")
}
