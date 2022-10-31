package upgrade

import (
	"testing"

	"github.com/kubeedge/beehive/pkg/core/model"
)

func TestFilter(t *testing.T) {
	uh := &upgradeHandler{}

	tests := []struct {
		msg  *model.Message
		want bool
		name string
	}{
		{
			msg: &model.Message{
				Router: model.MessageRoute{Group: "nodeupgradejobcontroller"},
			},
			want: true,
			name: "Node upgrade job controlled",
		},
		{
			msg: &model.Message{
				Router: model.MessageRoute{Group: "devicecontroller"},
			},
			want: false,
			name: "Device controller",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := uh.Filter(tt.msg); got != tt.want {
				t.Errorf("upgradeHandler.Filter() retuned unexpected result. got = %v, want = %v", got, tt.want)
			}
		})
	}
}
