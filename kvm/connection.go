package kvm

import (
	"fmt"
	"net"

	libvirt "github.com/digitalocean/go-libvirt"
)

// UnixDialer implements libvirt.Dialer for Unix socket connections
type UnixDialer struct {
	Path string
}

// Dial establishes a Unix socket connection
func (d *UnixDialer) Dial() (net.Conn, error) {
	return net.Dial("unix", d.Path)
}

// Connect establishes a connection to libvirt and returns the client
func Connect(socketPath string) (*libvirt.Libvirt, error) {
	dialer := &UnixDialer{Path: socketPath}
	vir := libvirt.NewWithDialer(dialer)
	if err := vir.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to libvirt daemon: %w", err)
	}
	return vir, nil
}

// Disconnect safely closes a libvirt connection
func Disconnect(vir *libvirt.Libvirt) error {
	if vir != nil {
		return vir.Disconnect()
	}
	return nil
}
