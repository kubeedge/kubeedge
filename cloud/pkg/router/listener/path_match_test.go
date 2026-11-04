package listener

import (
	"testing"
)

func noopHandle(interface{}) (interface{}, error) {
	return nil, nil
}

func TestMatchedPathOnlyMatchesWholeSegments(t *testing.T) {
	rh := &RestHandler{}
	rh.AddListener("/default/api", Handle(noopHandle))

	cases := []struct {
		uri  string
		want string
	}{
		{"/node1/default/api", "/default/api"},
		{"/node1/default/api/", "/default/api"},
		{"/node1/default/api/users/1", "/default/api"},
		{"/node1/default/api?limit=1", "/default/api"},
		{"/node1/default/apiv2", ""},
		{"/node1/default/api-internal", ""},
		{"/node1/default/apikeys/secret", ""},
		{"/node1/other/default/api", ""},
		{"/node1/default", ""},
		{"/node1", ""},
	}
	for _, c := range cases {
		t.Run(c.uri, func(t *testing.T) {
			got, exist := rh.matchedPath(c.uri)
			if c.want == "" {
				if exist {
					t.Fatalf("matchedPath(%q) = %q, want no match", c.uri, got)
				}
				return
			}
			if !exist {
				t.Fatalf("matchedPath(%q) found no match, want %q", c.uri, c.want)
			}
			if got != c.want {
				t.Fatalf("matchedPath(%q) = %q, want %q", c.uri, got, c.want)
			}
		})
	}
}

func TestMatchedPathIsIndependentOfRegistrationOrder(t *testing.T) {
	orders := [][]string{
		{"/default/api", "/default/apiv2"},
		{"/default/apiv2", "/default/api"},
	}
	for _, order := range orders {
		rh := &RestHandler{}
		for _, key := range order {
			rh.AddListener(key, Handle(noopHandle))
		}
		for uri, want := range map[string]string{
			"/node1/default/api":       "/default/api",
			"/node1/default/apiv2":     "/default/apiv2",
			"/node1/default/apiv2/sub": "/default/apiv2",
			"/node1/default/api/sub":   "/default/api",
		} {
			got, exist := rh.matchedPath(uri)
			if !exist || got != want {
				t.Fatalf("registered %v: matchedPath(%q) = %q exist=%v, want %q", order, uri, got, exist, want)
			}
		}
	}
}

func TestMatchedPathSupportsNamespaceWideAndParamRules(t *testing.T) {
	rh := &RestHandler{}
	rh.AddListener("/default/", Handle(noopHandle))
	rh.AddListener("/other/a/{id}", Handle(noopHandle))
	rh.AddListener("/other/api.v1", Handle(noopHandle))

	cases := []struct {
		uri  string
		want string
	}{
		{"/node1/default", ""},
		{"/node1/default/", "/default/"},
		{"/node1/default/anything", "/default/"},
		{"/node1/default/anything/deeper", "/default/"},
		{"/node1/other/a/5", "/other/a/{id}"},
		{"/node1/other/a/5/sub", "/other/a/{id}"},
		{"/node1/other/a", ""},
		{"/node1/other/api.v1", "/other/api.v1"},
		{"/node1/other/apiXv1", ""},
	}
	for _, c := range cases {
		t.Run(c.uri, func(t *testing.T) {
			got, exist := rh.matchedPath(c.uri)
			if c.want == "" {
				if exist {
					t.Fatalf("matchedPath(%q) = %q, want no match", c.uri, got)
				}
				return
			}
			if !exist || got != c.want {
				t.Fatalf("matchedPath(%q) = %q exist=%v, want %q", c.uri, got, exist, c.want)
			}
		})
	}
}
