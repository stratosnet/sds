//go:build !windows
// +build !windows

package rpc

// isPacketTooBig reports whether err indicates that a UDP packet didn't
// fit the receive buffer. There is no such error on
// non-Windows platforms.
func isPacketTooBig(err error) bool {
	return false
}
