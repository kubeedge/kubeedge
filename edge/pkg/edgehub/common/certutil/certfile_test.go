package certutil

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"testing"
)

func TestWriteKeyAndCert(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Error(err)
		return
	}
	type args struct {
		keyFile  string
		certFile string
		key      crypto.Signer
		cert     *x509.Certificate
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "TestWriteKeyAndCert(): Case 1: Test without key",
			args: args{
				keyFile:  "/tmp/key.pem",
				certFile: "/tmp/crt.pem",
				key:      nil,
				cert:     nil,
			},
			wantErr: fmt.Errorf("private key cannot be nil when writing to file"),
		},
		{
			name: "TestWriteKeyAndCert(): Case 2: Test without cert",
			args: args{
				keyFile:  "/tmp/key.pem",
				certFile: "/tmp/crt.pem",
				key:      key,
				cert:     nil,
			},
			wantErr: fmt.Errorf("certificate cannot be nil when writing to file"),
		},
		{
			name: "TestWriteKeyAndCert(): Case 1: Test with key and cert",
			args: args{
				keyFile:  "/tmp/key.pem",
				certFile: "/tmp/crt.pem",
				key:      key,
				cert:     &x509.Certificate{},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := WriteKeyAndCert(tt.args.keyFile, tt.args.certFile, tt.args.key, tt.args.cert); err != nil && err.Error() != tt.wantErr.Error() {
				t.Errorf("WriteKeyAndCert() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
