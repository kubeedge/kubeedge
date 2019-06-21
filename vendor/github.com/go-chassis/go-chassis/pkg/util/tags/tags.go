package utiltags

import (
	"sort"

	"github.com/go-chassis/go-chassis/core/common"
)

// Tags set tags and label
type Tags struct {
	KV    map[string]string
	Label string // format: version:1.0|app:mall|env:prod
}

// NewDefaultTag returns Tags with version and appID
func NewDefaultTag(version, appID string) Tags {
	return Tags{
		KV:    map[string]string{common.BuildinTagVersion: version, common.BuildinTagApp: appID},
		Label: common.BuildinTagApp + ":" + appID + "|" + common.BuildinTagVersion + ":" + version,
	}
}

// String returns label of tags
func (t Tags) String() string { return t.Label }

// AppID returns buildinTagApp of tags
func (t Tags) AppID() string { return t.KV[common.BuildinTagApp] }

// Version returns buildinTagVersion of tags
func (t Tags) Version() string { return t.KV[common.BuildinTagVersion] }

// IsSubsetOf returns if tags is labels
func (t Tags) IsSubsetOf(labels map[string]string) bool {
	for k, v := range t.KV {
		// TODO: remove buildinTag version
		if k == common.BuildinTagVersion && v == common.LatestVersion {
			continue
		}
		if labels[k] != v {
			return false
		}
	}
	return true
}

// LabelOfTags returns tags as string
func LabelOfTags(t map[string]string) (ret string) {
	ss := make([]string, 0, len(t))
	for k := range t {
		ss = append(ss, k)
	}
	sort.Sort(sort.StringSlice(ss))
	for i, s := range ss {
		ret += s + ":" + t[s]
		if i != len(ss)-1 {
			ret += "|"
		}
	}
	return
}
