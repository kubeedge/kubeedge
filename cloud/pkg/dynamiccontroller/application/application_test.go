package application

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestApplicationReason(t *testing.T) {
	type args struct {
		set  error
		want error
	}
	normalErr := fmt.Errorf("a normal error include %w", errors.New("internal error"))
	app := newApplication(context.Background(), "", "", "", nil, nil)
	tests := []struct {
		name string
		args args
	}{
		{
			name: "TestApplicationReason(): Case 1: no reason",
			args: args{
				nil,
				nil,
			},
		},
		{
			name: "TestApplicationReason(): Case 2: apierrors.",
			args: args{
				apierrors.NewAlreadyExists(schema.GroupResource{Resource: "foos"}, "bar"),
				apierrors.NewAlreadyExists(schema.GroupResource{Resource: "foos"}, "bar"),
			},
		},
		{
			name: "TestApplicationReason(): Case 3: normal error should return a wrapped internal error",
			args: args{
				normalErr,
				apierrors.NewInternalError(normalErr),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app.setReason(tt.args.set)
			if reason := app.getReason(); !reflect.DeepEqual(reason, tt.args.want) {
				t.Errorf("TestApplicationReason error = %v  wantErr %v", reason, tt.args.want)
			}
		})
	}
}
