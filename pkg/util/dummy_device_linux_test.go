/*
Copyright 2025 The KubeEdge Authors.

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

package util

import (
	"errors"
	"net"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

func TestNewDummyDeviceManager(t *testing.T) {
	manager := NewDummyDeviceManager()
	assert.NotNil(t, manager, "Expected manager to be created successfully")
}

type mockLink struct {
	netlink.LinkAttrs
}

func (m *mockLink) Attrs() *netlink.LinkAttrs {
	return &m.LinkAttrs
}

func (m *mockLink) Type() string {
	return "mock"
}

func TestEnsureDummyDevice(t *testing.T) {
	manager := &DummyDeviceManager{}

	t.Run("Device already exists", func(t *testing.T) {
		patches := gomonkey.ApplyFunc((*netlink.Handle).LinkByName,
			func(_ *netlink.Handle, name string) (netlink.Link, error) {
				return &netlink.Dummy{
					LinkAttrs: netlink.LinkAttrs{Name: name},
				}, nil
			})
		defer patches.Reset()

		exists, err := manager.EnsureDummyDevice("dummy0")
		assert.True(t, exists, "Expected device to exist")
		assert.NoError(t, err, "Expected no error")
	})

	t.Run("Device doesn't exist and creation fails", func(t *testing.T) {
		patches := gomonkey.ApplyFunc((*netlink.Handle).LinkByName,
			func(_ *netlink.Handle, name string) (netlink.Link, error) {
				return nil, errors.New("not found")
			})
		patches.ApplyFunc((*netlink.Handle).LinkAdd,
			func(_ *netlink.Handle, link netlink.Link) error {
				return errors.New("failed to create link")
			})
		defer patches.Reset()

		exists, err := manager.EnsureDummyDevice("dummy0")
		assert.False(t, exists, "Expected device not to exist")
		assert.Error(t, err, "Expected error on creation failure")
	})
}

func TestDeleteDummyDevice(t *testing.T) {
	manager := &DummyDeviceManager{}

	t.Run("Device exists and deleted successfully", func(t *testing.T) {
		patches := gomonkey.ApplyFunc((*netlink.Handle).LinkByName,
			func(_ *netlink.Handle, name string) (netlink.Link, error) {
				return &netlink.Dummy{
					LinkAttrs: netlink.LinkAttrs{Name: name},
				}, nil
			})
		patches.ApplyFunc((*netlink.Handle).LinkDel,
			func(_ *netlink.Handle, link netlink.Link) error {
				return nil
			})
		defer patches.Reset()

		err := manager.DeleteDummyDevice("dummy0")
		assert.NoError(t, err, "Expected no error on deletion")
	})

	t.Run("Device doesn't exist", func(t *testing.T) {
		patches := gomonkey.ApplyFunc((*netlink.Handle).LinkByName,
			func(_ *netlink.Handle, name string) (netlink.Link, error) {
				return nil, netlink.LinkNotFoundError{}
			})
		defer patches.Reset()

		err := manager.DeleteDummyDevice("dummy0")
		assert.NoError(t, err, "Expected no error when device doesn't exist")
	})

	t.Run("LinkByName returns error other than not found", func(t *testing.T) {
		patches := gomonkey.ApplyFunc((*netlink.Handle).LinkByName,
			func(_ *netlink.Handle, name string) (netlink.Link, error) {
				return nil, errors.New("other error")
			})
		defer patches.Reset()

		err := manager.DeleteDummyDevice("dummy0")
		assert.Error(t, err, "Expected error when LinkByName fails with non-not-found error")
		assert.Contains(t, err.Error(), "failed to delete a non-exist dummy device")
	})

	t.Run("Link exists but is not a dummy device", func(t *testing.T) {
		patches := gomonkey.ApplyFunc((*netlink.Handle).LinkByName,
			func(_ *netlink.Handle, name string) (netlink.Link, error) {
				return &mockLink{
					LinkAttrs: netlink.LinkAttrs{Name: name},
				}, nil
			})
		defer patches.Reset()

		err := manager.DeleteDummyDevice("dummy0")
		assert.Error(t, err, "Expected error when device is not a dummy")
		assert.Contains(t, err.Error(), "expect dummy device")
	})
}

func TestListBindAddress(t *testing.T) {
	manager := &DummyDeviceManager{}

	t.Run("Device exists and has addresses", func(t *testing.T) {
		patches := gomonkey.ApplyFunc((*netlink.Handle).LinkByName,
			func(_ *netlink.Handle, name string) (netlink.Link, error) {
				return &netlink.Dummy{
					LinkAttrs: netlink.LinkAttrs{Name: name},
				}, nil
			})
		patches.ApplyFunc((*netlink.Handle).AddrList,
			func(_ *netlink.Handle, link netlink.Link, family int) ([]netlink.Addr, error) {
				return []netlink.Addr{
					{IPNet: &net.IPNet{IP: net.ParseIP("192.168.1.1")}},
					{IPNet: &net.IPNet{IP: net.ParseIP("10.0.0.1")}},
				}, nil
			})
		defer patches.Reset()

		ips, err := manager.ListBindAddress("dummy0")
		assert.NoError(t, err, "Expected no error")
		assert.Equal(t, []string{"192.168.1.1", "10.0.0.1"}, ips, "Expected correct IPs")
	})

	t.Run("LinkByName fails", func(t *testing.T) {
		patches := gomonkey.ApplyFunc((*netlink.Handle).LinkByName,
			func(_ *netlink.Handle, name string) (netlink.Link, error) {
				return nil, errors.New("link error")
			})
		defer patches.Reset()

		ips, err := manager.ListBindAddress("dummy0")
		assert.Error(t, err, "Expected error when LinkByName fails")
		assert.Nil(t, ips, "Expected nil IPs")
		assert.Contains(t, err.Error(), "failed to get interface")
	})

	t.Run("AddrList fails", func(t *testing.T) {
		patches := gomonkey.ApplyFunc((*netlink.Handle).LinkByName,
			func(_ *netlink.Handle, name string) (netlink.Link, error) {
				return &netlink.Dummy{
					LinkAttrs: netlink.LinkAttrs{Name: name},
				}, nil
			})
		patches.ApplyFunc((*netlink.Handle).AddrList,
			func(_ *netlink.Handle, link netlink.Link, family int) ([]netlink.Addr, error) {
				return nil, errors.New("addr list error")
			})
		defer patches.Reset()

		ips, err := manager.ListBindAddress("dummy0")
		assert.Error(t, err, "Expected error when AddrList fails")
		assert.Nil(t, ips, "Expected nil IPs")
		assert.Contains(t, err.Error(), "failed to list bound address")
	})
}

func TestEnsureAddressBind(t *testing.T) {
	manager := &DummyDeviceManager{}

	t.Run("Device exists and address bound successfully", func(t *testing.T) {
		patches := gomonkey.ApplyFunc((*netlink.Handle).LinkByName,
			func(_ *netlink.Handle, name string) (netlink.Link, error) {
				return &netlink.Dummy{
					LinkAttrs: netlink.LinkAttrs{Name: name},
				}, nil
			})
		patches.ApplyFunc((*netlink.Handle).AddrAdd,
			func(_ *netlink.Handle, link netlink.Link, addr *netlink.Addr) error {
				return nil
			})
		defer patches.Reset()

		bound, err := manager.EnsureAddressBind("192.168.1.1", "dummy0")
		assert.NoError(t, err, "Expected no error")
		assert.False(t, bound, "Expected address to be newly bound")
	})

	t.Run("Device exists and address already bound", func(t *testing.T) {
		patches := gomonkey.ApplyFunc((*netlink.Handle).LinkByName,
			func(_ *netlink.Handle, name string) (netlink.Link, error) {
				return &netlink.Dummy{
					LinkAttrs: netlink.LinkAttrs{Name: name},
				}, nil
			})
		patches.ApplyFunc((*netlink.Handle).AddrAdd,
			func(_ *netlink.Handle, link netlink.Link, addr *netlink.Addr) error {
				return unix.EEXIST
			})
		defer patches.Reset()

		bound, err := manager.EnsureAddressBind("192.168.1.1", "dummy0")
		assert.NoError(t, err, "Expected no error")
		assert.True(t, bound, "Expected address to be already bound")
	})

	t.Run("LinkByName fails", func(t *testing.T) {
		patches := gomonkey.ApplyFunc((*netlink.Handle).LinkByName,
			func(_ *netlink.Handle, name string) (netlink.Link, error) {
				return nil, errors.New("link error")
			})
		defer patches.Reset()

		bound, err := manager.EnsureAddressBind("192.168.1.1", "dummy0")
		assert.Error(t, err, "Expected error when LinkByName fails")
		assert.False(t, bound, "Expected not bound")
		assert.Contains(t, err.Error(), "failed to get interface")
	})

	t.Run("Invalid IP address", func(t *testing.T) {
		patches := gomonkey.ApplyFunc((*netlink.Handle).LinkByName,
			func(_ *netlink.Handle, name string) (netlink.Link, error) {
				return &netlink.Dummy{
					LinkAttrs: netlink.LinkAttrs{Name: name},
				}, nil
			})
		defer patches.Reset()

		bound, err := manager.EnsureAddressBind("invalid-ip", "dummy0")
		assert.Error(t, err, "Expected error with invalid IP")
		assert.False(t, bound, "Expected not bound")
		assert.Contains(t, err.Error(), "failed to parse ip address")
	})

	t.Run("AddrAdd fails with error other than EEXIST", func(t *testing.T) {
		patches := gomonkey.ApplyFunc((*netlink.Handle).LinkByName,
			func(_ *netlink.Handle, name string) (netlink.Link, error) {
				return &netlink.Dummy{
					LinkAttrs: netlink.LinkAttrs{Name: name},
				}, nil
			})
		patches.ApplyFunc((*netlink.Handle).AddrAdd,
			func(_ *netlink.Handle, link netlink.Link, addr *netlink.Addr) error {
				return errors.New("addr add error")
			})
		defer patches.Reset()

		bound, err := manager.EnsureAddressBind("192.168.1.1", "dummy0")
		assert.Error(t, err, "Expected error when AddrAdd fails")
		assert.False(t, bound, "Expected not bound")
		assert.Contains(t, err.Error(), "failed to bind address")
	})
}
