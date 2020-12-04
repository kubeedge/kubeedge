package debug

import (
	"testing"

	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

func TestExecuteCollect(t *testing.T) {
	type args struct {
		collectOptions *types.CollectOptions
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "collectionOptionsNoArgs",
			args: args{collectOptions: &types.CollectOptions{
				Config:     "",
				OutputPath: "",
				Detail:     false,
				LogPath:    "",
			}},
			wantErr: true,
		},
		{
			name: "collectionOptionsWithCustomeArgs",
			args: args{collectOptions: &types.CollectOptions{
				Config:     "/etc/kubeedge/config/edgecore.yaml",
				OutputPath: "/etc/kubeedge/",
				Detail:     true,
				LogPath:    "/etc/kubeedge/",
			}},
			wantErr: false,
		},
		{
			name: "fakeConfigFile",
			args: args{collectOptions: &types.CollectOptions{
				Config:     "/etc/kubeedge/config/fakeConfig.yaml",
			}},
			wantErr: true,
		},
		{
			name: "fakeOutputPath",
			args: args{collectOptions: &types.CollectOptions{
				OutputPath: "/etc/kubeedge/fakeOutputPath/",
			}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ExecuteCollect(tt.args.collectOptions); (err != nil) != tt.wantErr {
				t.Errorf("ExecuteCollect() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
