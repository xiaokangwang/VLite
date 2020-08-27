package water

import "os"

func DropTunInterface(fd os.File) *Interface {
	return &Interface{isTAP: false, name: "/dev/net/tun", fd: int(fd.Fd()), ReadWriteCloser: &fd}
}
