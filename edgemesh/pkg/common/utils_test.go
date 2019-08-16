package common

import "testing"

func TestSplitServiceKey(t *testing.T) {
	type args struct {
		key string
	}
	tests := []struct {
		name          string
		args          args
		wantName      string
		wantNamespace string
	}{
		{"normal serviceKey", args{"foo.bar"}, "foo", "bar"},
		{"default namespace", args{"foo"}, "foo", "default"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotNamespace := SplitServiceKey(tt.args.key)
			if gotName != tt.wantName {
				t.Errorf("SplitServiceKey() gotName = %v, want %v", gotName, tt.wantName)
			}
			if gotNamespace != tt.wantNamespace {
				t.Errorf("SplitServiceKey() gotNamespace = %v, want %v", gotNamespace, tt.wantNamespace)
			}
		})
	}
}
