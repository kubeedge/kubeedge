package translator

import (
	"reflect"
	"testing"

	"github.com/kubeedge/beehive/pkg/core/model"
)

func TestMessageTranslator_Encode(t1 *testing.T) {
	type args struct {
		msg interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			args: args{
				msg: &model.Message{
					Header: model.MessageHeader{
						ID: "1",
					},
					Content: "msg",
				},
			},
			want: []byte("\n\x03\n\x011\x12\x00\x1a\x03msg"),
		},
		{
			args: args{
				msg: &model.Message{
					Header: model.MessageHeader{
						ID: "1",
					},
					Content: []byte("msg"),
				},
			},
			want: []byte("\n\x03\n\x011\x12\x00\x1a\x03msg"),
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &MessageTranslator{}
			got, err := t.Encode(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t1.Errorf("Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("Encode() got = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMessageTranslator_Decode(t1 *testing.T) {
	type args struct {
		raw []byte
		msg interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		wantMsg interface{}
	}{
		{
			args: args{
				raw: []byte("\n\x03\n\x011\x12\x00\x1a\x03msg"),
				msg: &model.Message{},
			},
			wantMsg: &model.Message{
				Header: model.MessageHeader{
					ID: "1",
				},
				Content: []byte("msg"),
			},
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &MessageTranslator{}
			if err := t.Decode(tt.args.raw, tt.args.msg); (err != nil) != tt.wantErr {
				t1.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.args.msg, tt.wantMsg) {
				t1.Errorf("Encode() \n got = %#v, \nwant = %#v", tt.args.msg, tt.wantMsg)
			}
		})
	}
}
