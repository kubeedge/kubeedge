package reader

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"sync"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/socket/wrapper/packer"
)

// PackageReader package reader
type PackageReader struct {
	scanner *bufio.Scanner
	lock    sync.Mutex
	packer  *packer.Packer
}

// NewPackageReader new package reader
func NewPackageReader(obj interface{}, buffSize int) *PackageReader {
	if buffSize <= 0 {
		klog.Errorf("bad buffer size %d", buffSize)
		return nil
	}

	conn, ok := obj.(net.Conn)
	if !ok {
		klog.Errorf("bad conn obj")
		return nil
	}

	p := packer.NewPacker()
	splitFunc := func(data []byte, eof bool) (advance int, token []byte, err error) {
		if !eof && p.Validate(data) {
			length := p.GetMessageLen(data)
			packageLen := int(length) + packer.MessageOffset
			if packageLen <= len(data) {
				return packageLen, data[packer.MessageOffset:packageLen], nil
			}
		}
		return
	}
	scanner := bufio.NewScanner(conn)
	scanner.Split(splitFunc)
	scanner.Buffer(make([]byte, buffSize), buffSize)

	return &PackageReader{
		scanner: scanner,
		packer:  p,
	}
}

// Read read
func (reader *PackageReader) Read() ([]byte, error) {
	reader.lock.Lock()
	defer reader.lock.Unlock()

	gotPackage := reader.scanner.Scan()
	if !gotPackage || reader.scanner.Err() != nil {
		return nil, fmt.Errorf("failed to scann package, error: %+v", reader.scanner.Err())
	}
	return reader.scanner.Bytes(), nil
}

// ReadJSON read json
func (reader *PackageReader) ReadJSON(obj interface{}) error {
	reader.lock.Lock()
	defer reader.lock.Unlock()

	gotPackage := reader.scanner.Scan()
	if !gotPackage || reader.scanner.Err() != nil {
		return fmt.Errorf("failed to scann package, error: %+v", reader.scanner.Err())
	}
	return json.NewDecoder(bytes.NewReader(reader.scanner.Bytes())).Decode(obj)
}
