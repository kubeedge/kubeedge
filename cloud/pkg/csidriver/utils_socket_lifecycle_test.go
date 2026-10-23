/*
Copyright 2026 The KubeEdge Authors.

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
	"io"
	"net"
	"runtime/debug"
	"testing"
	"time"
)

func TestSendToKubeEdgeClosesConnection(t *testing.T) {
	// A GC finalizer must not hide a missing explicit Close in the client.
	previousGCPercent := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(previousGCPercent)

	for _, tc := range []struct {
		name      string
		readError bool
	}{
		{name: "success"},
		{name: "read error", readError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			deadline := time.Now().Add(5 * time.Second)
			if err := listener.SetDeadline(deadline); err != nil {
				t.Fatal(err)
			}

			const request = "request"
			peerResult := make(chan error, 1)
			go func() {
				peerResult <- func() error {
					conn, err := listener.AcceptTCP()
					if err != nil {
						return err
					}
					defer conn.Close()
					if err := conn.SetDeadline(deadline); err != nil {
						return err
					}
					got := make([]byte, len(request))
					if _, err := io.ReadFull(conn, got); err != nil {
						return err
					}
					if string(got) != request {
						return fmt.Errorf("request = %q, want %q", got, request)
					}
					if tc.readError {
						// EOF reaches the client's Read while the server's read
						// half stays open to observe the client's own Close.
						if err := conn.CloseWrite(); err != nil {
							return err
						}
					} else {
						// A one-byte response does not assume TCP packet framing.
						if _, err := conn.Write([]byte("r")); err != nil {
							return err
						}
					}

					var extra [1]byte
					n, err := conn.Read(extra[:])
					if n != 0 || err != io.EOF {
						return fmt.Errorf("client did not close after Send returned: read = (%d, %v), want (0, EOF)", n, err)
					}
					return nil
				}()
			}()

			response, sendErr := sendToKubeEdge(request, "tcp://"+listener.Addr().String())
			// Always collect the peer before reporting an assertion failure.
			// Its deadline also bounds the unmodified, leaking implementation.
			peerErr := <-peerResult
			if tc.readError {
				if response != "" || sendErr != io.EOF {
					t.Errorf("sendToKubeEdge = (%q, %v), want (%q, EOF)", response, sendErr, "")
				}
			} else if response != "r" || sendErr != nil {
				t.Errorf("sendToKubeEdge = (%q, %v), want (%q, nil)", response, sendErr, "r")
			}
			if peerErr != nil {
				t.Fatal(peerErr)
			}
		})
	}
}

func TestSendToKubeEdgeConnectFailure(t *testing.T) {
	const endpoint = "invalid://endpoint"
	response, err := sendToKubeEdge("request", endpoint)
	if response != "" || err == nil || err.Error() != "invalid endpoint: "+endpoint {
		t.Fatalf("sendToKubeEdge = (%q, %v), want empty response and original endpoint error", response, err)
	}
}
