package runtime

import (
	"net"
	"os"
	"time"
)

func SocketExists(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeSocket) != 0
}

func UnixSocketDialOK(path string, timeout time.Duration) bool {
	if path == "" {
		return false
	}
	conn, err := net.DialTimeout("unix", path, timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
