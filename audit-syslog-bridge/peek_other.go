//go:build !linux

package main

import "net"

// peek is a stub on non-Linux platforms.
func peek(net.Conn) peekState {
	return peekUnknown
}
