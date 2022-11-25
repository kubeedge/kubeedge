/*
Copyright 2022 The KubeEdge Authors.

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

package csidriver

import (
	"fmt"
	"net"
	"os"
	"reflect"
	"testing"
)

func TestNewUnixDomainSocket(t *testing.T) {
	tests := []struct {
		name       string
		filename   string
		buffersize []int
		want       *UnixDomainSocket
	}{
		{
			name:       "base",
			filename:   "default",
			buffersize: []int{1024},
			want: &UnixDomainSocket{
				filename:   "default",
				buffersize: 1024,
			},
		},
		{
			name:     "buffersize is nil",
			filename: "default",
			want: &UnixDomainSocket{
				filename:   "default",
				buffersize: 10480,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewUnixDomainSocket(tt.filename, tt.buffersize...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewUnixDomainSocket() = %v, want %v", got, tt.want)
			}
		})
	}
}

func newLocalListener() (net.Listener, error) {
	f, err := os.CreateTemp("", "unix.sock")
	if err != nil {
		panic(err)
	}
	addr := f.Name()
	f.Close()
	os.Remove(addr)
	return net.Listen("unix", addr)
}

func TestUnixDomainSocket_Connect(t *testing.T) {
	ln, err := newLocalListener()
	if err == nil {
		defer ln.Close()
	}
	want, err := net.Dial("unix", ln.Addr().String())
	if err == nil {
		defer want.Close()
	}

	tests := []struct {
		name       string
		filename   string
		buffersize int
		want       net.Conn
		wantErr    bool
	}{
		{
			name:     "base",
			filename: "default",
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "address not exist and dial error",
			filename: "unix://" + "not-exist",
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "address exist and dial successfully",
			filename: "unix://" + ln.Addr().String(),
			want:     want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			us := &UnixDomainSocket{
				filename:   tt.filename,
				buffersize: tt.buffersize,
			}
			got, err := us.Connect()
			if (err != nil) != tt.wantErr {
				t.Errorf("UnixDomainSocket.Connect() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				defer got.Close()
			}
			if !reflect.DeepEqual(got, tt.want) {
				if tt.want != nil { // real conn
					gotAddr, wantAddr := got.RemoteAddr().String(), tt.want.RemoteAddr().String()
					if gotAddr != wantAddr {
						t.Errorf("UnixDomainSocket.Connect() = %v, want %v", gotAddr, wantAddr)
					}
				} else {
					t.Errorf("UnixDomainSocket.Connect() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

type fakeConn struct {
	net.Conn
}

func (c *fakeConn) Write(b []byte) (n int, err error) {
	if len(b) == 0 {
		return 0, fmt.Errorf("raise an error")
	}
	return len(b), nil
}

func (c *fakeConn) Read(b []byte) (n int, err error) {
	if len(b) == 0 {
		return 0, fmt.Errorf("raise an error")
	}
	return len(b), nil
}

func TestUnixDomainSocket_Send(t *testing.T) {
	tests := []struct {
		name       string
		filename   string
		buffersize int
		c          net.Conn
		context    string
		want       string
		wantErr    bool
	}{
		{
			name:    "base",
			c:       &fakeConn{},
			want:    "",
			wantErr: true,
		},
		{
			name:    "write ok",
			c:       &fakeConn{},
			context: "default",
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			us := &UnixDomainSocket{
				filename:   tt.filename,
				buffersize: tt.buffersize,
			}
			got, err := us.Send(tt.c, tt.context)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnixDomainSocket.Send() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("UnixDomainSocket.Send() = %v, want %v", got, tt.want)
			}
		})
	}
}
