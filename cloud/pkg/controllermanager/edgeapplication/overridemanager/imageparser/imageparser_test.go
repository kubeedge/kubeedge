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

package imageparser

import "testing"

func TestParse(t *testing.T) {
	const digest = "sha256:50d858e0985ecc7f60418aaf0cc5ab587f42c2570a884095a9e8ccacd0f6545c"
	tests := []struct {
		name       string
		image      string
		wantErr    bool
		hostname   string
		repository string
		tag        string
		digest     string
	}{
		{
			name:       "repository only",
			image:      "nginx",
			repository: "nginx",
		},
		{
			name:       "repository and tag",
			image:      "nginx:1.19.1",
			repository: "nginx",
			tag:        "1.19.1",
		},
		{
			name:       "hostname repository and tag",
			image:      "k8s.gcr.io/kube-apiserver:v1.0.0",
			hostname:   "k8s.gcr.io",
			repository: "kube-apiserver",
			tag:        "v1.0.0",
		},
		{
			name:       "hostname with port",
			image:      "fictional.registry.example:10443/karmada/karmada-controller-manager:v1.0.0",
			hostname:   "fictional.registry.example:10443",
			repository: "karmada/karmada-controller-manager",
			tag:        "v1.0.0",
		},
		{
			name:       "digest reference",
			image:      "nginx@" + digest,
			repository: "nginx",
			digest:     digest,
		},
		{
			name:    "invalid reference",
			image:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.image)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse(%q) error = %v, wantErr %v", tt.image, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got.Hostname() != tt.hostname {
				t.Errorf("Hostname() = %q, want %q", got.Hostname(), tt.hostname)
			}
			if got.Repository() != tt.repository {
				t.Errorf("Repository() = %q, want %q", got.Repository(), tt.repository)
			}
			if got.Tag() != tt.tag {
				t.Errorf("Tag() = %q, want %q", got.Tag(), tt.tag)
			}
			if got.Digest() != tt.digest {
				t.Errorf("Digest() = %q, want %q", got.Digest(), tt.digest)
			}
		})
	}
}

func TestSplitHostname(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantHostname string
		wantRemote   string
	}{
		{name: "no slash", input: "nginx", wantHostname: "", wantRemote: "nginx"},
		{name: "dotted hostname", input: "k8s.gcr.io/kube-apiserver", wantHostname: "k8s.gcr.io", wantRemote: "kube-apiserver"},
		{name: "hostname with port", input: "registry:5000/app", wantHostname: "registry:5000", wantRemote: "app"},
		{name: "localhost special case", input: "localhost/app", wantHostname: "localhost", wantRemote: "app"},
		{name: "no dot no colon treated as path", input: "library/nginx", wantHostname: "", wantRemote: "library/nginx"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHost, gotRemote := SplitHostname(tt.input)
			if gotHost != tt.wantHostname || gotRemote != tt.wantRemote {
				t.Errorf("SplitHostname(%q) = (%q, %q), want (%q, %q)",
					tt.input, gotHost, gotRemote, tt.wantHostname, tt.wantRemote)
			}
		})
	}
}

func TestComponentsAccessors(t *testing.T) {
	c := &Components{}

	c.SetHostname("k8s.gcr.io")
	if c.Hostname() != "k8s.gcr.io" {
		t.Errorf("Hostname() = %q, want %q", c.Hostname(), "k8s.gcr.io")
	}

	c.SetRepository("kube-apiserver")
	if c.Repository() != "kube-apiserver" {
		t.Errorf("Repository() = %q, want %q", c.Repository(), "kube-apiserver")
	}
	if got := c.FullRepository(); got != "k8s.gcr.io/kube-apiserver" {
		t.Errorf("FullRepository() = %q, want %q", got, "k8s.gcr.io/kube-apiserver")
	}

	c.SetTag("v1.0.0")
	if c.Tag() != "v1.0.0" {
		t.Errorf("Tag() = %q, want %q", c.Tag(), "v1.0.0")
	}
	if got := c.String(); got != "k8s.gcr.io/kube-apiserver:v1.0.0" {
		t.Errorf("String() = %q, want %q", got, "k8s.gcr.io/kube-apiserver:v1.0.0")
	}
	if got := c.TagOrDigest(); got != "v1.0.0" {
		t.Errorf("TagOrDigest() = %q, want %q", got, "v1.0.0")
	}

	// RemoveTag then String should fall back to bare repository.
	c.RemoveTag()
	if c.Tag() != "" {
		t.Errorf("Tag() after RemoveTag = %q, want empty", c.Tag())
	}
	if got := c.String(); got != "k8s.gcr.io/kube-apiserver" {
		t.Errorf("String() after RemoveTag = %q, want %q", got, "k8s.gcr.io/kube-apiserver")
	}

	// RemoveHostname drops the registry prefix.
	c.RemoveHostname()
	if got := c.FullRepository(); got != "kube-apiserver" {
		t.Errorf("FullRepository() after RemoveHostname = %q, want %q", got, "kube-apiserver")
	}

	c.RemoveRepository()
	if c.Repository() != "" {
		t.Errorf("Repository() after RemoveRepository = %q, want empty", c.Repository())
	}
}

func TestComponentsDigest(t *testing.T) {
	const digest = "sha256:50d858e0985ecc7f60418aaf0cc5ab587f42c2570a884095a9e8ccacd0f6545c"
	c := &Components{}
	c.SetRepository("nginx")
	c.SetDigest(digest)

	if c.Digest() != digest {
		t.Errorf("Digest() = %q, want %q", c.Digest(), digest)
	}
	if got := c.String(); got != "nginx@"+digest {
		t.Errorf("String() = %q, want %q", got, "nginx@"+digest)
	}
	if got := c.TagOrDigest(); got != digest {
		t.Errorf("TagOrDigest() = %q, want %q", got, digest)
	}

	c.RemoveDigest()
	if c.Digest() != "" {
		t.Errorf("Digest() after RemoveDigest = %q, want empty", c.Digest())
	}
	if got := c.String(); got != "nginx" {
		t.Errorf("String() after RemoveDigest = %q, want %q", got, "nginx")
	}
}

func TestSetTagOrDigest(t *testing.T) {
	const digest = "sha256:50d858e0985ecc7f60418aaf0cc5ab587f42c2570a884095a9e8ccacd0f6545c"

	// A tag-like input sets the tag and clears any digest.
	c := &Components{digest: digest}
	c.SetTagOrDigest("v1.2.3")
	if c.Tag() != "v1.2.3" {
		t.Errorf("Tag() = %q, want %q", c.Tag(), "v1.2.3")
	}
	if c.Digest() != "" {
		t.Errorf("Digest() = %q, want empty after setting tag", c.Digest())
	}

	// A digest-like input sets the digest and clears any tag.
	c = &Components{tag: "v1.2.3"}
	c.SetTagOrDigest(digest)
	if c.Digest() != digest {
		t.Errorf("Digest() = %q, want %q", c.Digest(), digest)
	}
	if c.Tag() != "" {
		t.Errorf("Tag() = %q, want empty after setting digest", c.Tag())
	}

	// An input matching neither a tag nor a digest leaves both fields untouched.
	c = &Components{tag: "v1.2.3", digest: digest}
	c.SetTagOrDigest("invalid_format@")
	if c.Tag() != "v1.2.3" {
		t.Errorf("Tag() = %q, want %q after invalid input", c.Tag(), "v1.2.3")
	}
	if c.Digest() != digest {
		t.Errorf("Digest() = %q, want %q after invalid input", c.Digest(), digest)
	}
}

func TestRemoveTagOrDigest(t *testing.T) {
	// Tag takes precedence when present.
	c := &Components{tag: "v1", digest: "sha256:abc"}
	c.RemoveTagOrDigest()
	if c.Tag() != "" {
		t.Errorf("Tag() = %q, want empty", c.Tag())
	}
	if c.Digest() != "sha256:abc" {
		t.Errorf("Digest() = %q, want untouched", c.Digest())
	}

	// Falls back to removing digest when no tag.
	c = &Components{digest: "sha256:abc"}
	c.RemoveTagOrDigest()
	if c.Digest() != "" {
		t.Errorf("Digest() = %q, want empty", c.Digest())
	}
}
