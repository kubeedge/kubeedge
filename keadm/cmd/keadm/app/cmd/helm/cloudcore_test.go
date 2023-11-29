package helm

import (
	"fmt"
	"testing"

	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

func TestAppendDefaultSets(t *testing.T) {
	version := "v1.16.0"
	advertiseAddress := "127.0.0.1"
	componentsSize := len(setsKeyImageTags)
	opts := types.CloudInitUpdateBase{
		Sets: []string{},
	}
	appendDefaultSets(version, advertiseAddress, &opts)
	if len(opts.Sets) != componentsSize+1 {
		t.Fatalf("sets len want equal %d, but is %d", componentsSize, len(opts.Sets))
	}
	var idx int
	for idx = range setsKeyImageTags {
		want := setsKeyImageTags[idx] + "=" + version
		if opts.Sets[idx] != want {
			t.Fatalf("sets item[%d] is not %s", idx, want)
		}
	}
	want := "cloudCore.modules.cloudHub.advertiseAddress[0]=" + advertiseAddress
	if opts.Sets[idx+1] != want {
		t.Fatalf("sets item[%d] is not %s", idx+1, want)
	}
}

func TestGetValuesFile(t *testing.T) {
	cases := []struct {
		profile string
		want    string
	}{
		{profile: "version", want: "profiles/version.yaml"},
		{profile: "version.yaml", want: "profiles/version.yaml"},
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
