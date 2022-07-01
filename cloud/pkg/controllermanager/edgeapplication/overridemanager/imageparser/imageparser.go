/*
CHANGELOG
KubeEdge Authors:
- This File is drived from github.com/karmada-io/karmada/pkg/util/imageparser/parser.go
*/

package imageparser

import (
	"strings"

	"github.com/distribution/distribution/v3/reference"
)

// Components make up a whole image.
// Basically we presume an image can be made of `[domain][:port][path]<name>[:tag][@sha256:digest]`, ie:
// fictional.registry.example:10443/karmada/karmada-controller-manager:v1.0.0 or
// fictional.registry.example:10443/karmada/karmada-controller-manager@sha256:50d858e0985ecc7f60418aaf0cc5ab587f42c2570a884095a9e8ccacd0f6545c
type Components struct {
	// hostname is the prefix of referencing image, like "k8s.gcr.io".
	// It may optionally be followed by a port number, like "fictional.registry.example:10443".
	hostname string
	// repository is the short name of referencing image, like "karmada-controller-manager".
	// It may be made up of slash-separated components, like "karmada/karmada-controller-manager".
	repository string
	// tag is the tag of referencing image. ie:
	// - latest
	// - v1.19.1
	tag string
	// digest is the digest of referencing image. ie:
	// - sha256:50d858e0985ecc7f60418aaf0cc5ab587f42c2570a884095a9e8ccacd0f6545c
	digest string
}

// Hostname returns the hostname, as well known as image registry.
func (c *Components) Hostname() string {
	return c.hostname
}

// SetHostname sets the hostname.
func (c *Components) SetHostname(hostname string) {
	c.hostname = hostname
}

// RemoveHostname removes the hostname.
func (c *Components) RemoveHostname() {
	c.hostname = ""
}

// Repository returns the repository without hostname.
func (c *Components) Repository() string {
	return c.repository
}

// SetRepository sets the repository.
func (c *Components) SetRepository(repository string) {
	c.repository = repository
}

// RemoveRepository removes the repository.
func (c *Components) RemoveRepository() {
	c.repository = ""
}

// FullRepository returns the whole repository, including hostname if not empty.
func (c *Components) FullRepository() string {
	if c.hostname == "" {
		return c.repository
	}

	return c.hostname + "/" + c.repository
}

// String returns the full name of the image, including repository and tag(or digest).
func (c Components) String() string {
	if c.tag != "" {
		return c.FullRepository() + ":" + c.tag
	} else if c.digest != "" {
		return c.FullRepository() + "@" + c.digest
	}

	return c.FullRepository()
}

// Tag returns the tag.
func (c *Components) Tag() string {
	return c.tag
}

// SetTag sets the tag.
func (c *Components) SetTag(tag string) {
	c.tag = tag
}

// RemoveTag removes the tag.
func (c *Components) RemoveTag() {
	c.tag = ""
}

// Digest returns the digest.
func (c *Components) Digest() string {
	return c.digest
}

// SetDigest sets the digest.
func (c *Components) SetDigest(digest string) {
	c.digest = digest
}

// RemoveDigest removes the digest.
func (c *Components) RemoveDigest() {
	c.digest = ""
}

// TagOrDigest returns image tag if not empty otherwise returns image digest.
func (c *Components) TagOrDigest() string {
	if c.tag != "" {
		return c.tag
	}

	return c.digest
}

// SetTagOrDigest sets image tag or digest by matching the input.
func (c *Components) SetTagOrDigest(input string) {
	if anchoredTagRegexp.MatchString(input) {
		c.SetTag(input)
		c.RemoveDigest()
		return
	}

	if anchoredDigestRegexp.MatchString(input) {
		c.SetDigest(input)
		c.RemoveTag()
	}
}

// RemoveTagOrDigest removes tag or digest.
// Since tag and digest don't co-exist, so remove tag if tag not empty, otherwise remove digest.
func (c *Components) RemoveTagOrDigest() {
	if c.tag != "" {
		c.tag = ""
	} else if c.digest != "" {
		c.digest = ""
	}
}

// Parse returns a Components of the given image.
func Parse(image string) (*Components, error) {
	ref, err := reference.Parse(image)
	if err != nil {
		return nil, err
	}

	comp := &Components{}
	if named, ok := ref.(reference.Named); ok {
		comp.hostname, comp.repository = SplitHostname(named.Name())
	}

	if tagged, ok := ref.(reference.Tagged); ok {
		comp.tag = tagged.Tag()
	} else if digested, ok := ref.(reference.Digested); ok {
		comp.digest = digested.Digest().String()
	}

	return comp, nil
}

// SplitHostname splits a repository name(ie: k8s.gcr.io/kube-apiserver) to hostname(k8s.gcr.io) and remotename(kube-apiserver) string.
func SplitHostname(name string) (hostname, remoteName string) {
	i := strings.IndexRune(name, '/')
	if i == -1 || (!strings.ContainsAny(name[:i], ".:") && name[:i] != "localhost") {
		hostname, remoteName = "", name
	} else {
		hostname, remoteName = name[:i], name[i+1:]
	}
	return
}
