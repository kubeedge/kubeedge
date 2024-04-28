//go:build !linux

package metaserver

func setupDummyInterface() error {
	return nil
}
