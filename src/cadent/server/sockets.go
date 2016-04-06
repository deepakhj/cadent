/*
   To allow net.UDPcon to have various Nice kernel flags used

*/

package cadent

import (
	"bufio"
	"errors"
	reuse "github.com/jbenet/go-reuseport"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

func ReusePortID() int {
	switch runtime.GOOS {

	case "darwin", "freebsd", "netbsd", "openbsd":
		return syscall.SO_REUSEPORT

	default:
		return 0x0F
	}
}

func FastOpenId() int {
	switch runtime.GOOS {

	case "darwin", "freebsd", "netbsd", "openbsd":
		return 0

	default:
		return 0x16
	}
}

// getSockaddr parses protocol and address and returns implementor syscall.Sockaddr: syscall.SockaddrInet4 or syscall.SockaddrInet6.
func getSockaddr(proto string, addr string) (sa syscall.Sockaddr, soType int, err error) {
	var (
		addr4 [4]byte
		addr6 [16]byte
		ip    *net.TCPAddr
		uip   *net.UDPAddr
		iip   *net.IPAddr
	)

	switch proto {
	default:
		return nil, -1, errors.New("unsupported protocal for socket")
	case "tcp", "tcp4":
		ip, err = net.ResolveTCPAddr(proto, addr)
		if err != nil {
			return nil, -1, err
		}
		if ip.IP != nil {
			copy(addr4[:], ip.IP[12:16]) // copy last 4 bytes of slice to array
		}
		return &syscall.SockaddrInet4{Port: ip.Port, Addr: addr4}, syscall.AF_INET, nil
	case "tcp6":
		ip, err = net.ResolveTCPAddr(proto, addr)
		if err != nil {
			return nil, -1, err
		}
		if ip.IP != nil {
			copy(addr6[:], ip.IP) // copy all bytes of slice to array
		}
		return &syscall.SockaddrInet6{Port: ip.Port, Addr: addr6}, syscall.AF_INET6, nil
	case "udp", "udp4":
		uip, err = net.ResolveUDPAddr(proto, addr)
		if err != nil {
			return nil, -1, err
		}
		if uip.IP != nil {
			copy(addr4[:], uip.IP[12:16]) // copy last 4 bytes of slice to array
		}
		return &syscall.SockaddrInet4{Port: uip.Port, Addr: addr4}, syscall.AF_INET, nil

	case "udp6":
		uip, err = net.ResolveUDPAddr(proto, addr)
		if err != nil {
			return nil, -1, err
		}
		if uip.IP != nil {
			copy(addr4[:], uip.IP[12:16]) // copy last 4 bytes of slice to array
		}
		return &syscall.SockaddrInet6{Port: uip.Port, Addr: addr6}, syscall.AF_INET6, nil

	case "ip", "ip4":
		iip, err = net.ResolveIPAddr(proto, addr)
		if err != nil {
			return nil, -1, err
		}
		if uip.IP != nil {
			copy(addr4[:], iip.IP[12:16]) // copy last 4 bytes of slice to array
		}
		return &syscall.SockaddrInet4{Addr: addr4}, syscall.AF_INET, nil

	case "ip6":
		iip, err = net.ResolveIPAddr(proto, addr)
		if err != nil {
			return nil, -1, err
		}
		if iip.IP != nil {
			copy(addr4[:], iip.IP[12:16]) // copy last 4 bytes of slice to array
		}
		return &syscall.SockaddrInet6{Addr: addr6}, syscall.AF_INET6, nil

	}

}

func maxListenerBacklog() int {
	var (
		n   uint32
		err error
	)
	switch runtime.GOOS {

	case "darwin", "freebsd":
		n, err = syscall.SysctlUint32("kern.ipc.somaxconn")
	case "netbsd":
	// NOTE: NetBSD has no somaxconn-like kernel state so far
	case "openbsd":
		n, err = syscall.SysctlUint32("kern.somaxconn")
	//linux generic
	default:
		fd, err := os.Open("/proc/sys/net/core/somaxconn")
		if err != nil {
			return syscall.SOMAXCONN
		}
		defer fd.Close()
		rd := bufio.NewReader(fd)
		line, err := rd.ReadString('\n')
		if err != nil {
			return syscall.SOMAXCONN
		}
		f := strings.Fields(line)
		if len(f) < 1 {
			return syscall.SOMAXCONN
		}
		n, err := strconv.Atoi(f[0])
		if err != nil || n == 0 {
			return syscall.SOMAXCONN
		}
		// Linux stores the backlog in a uint16.
		// Truncate number to avoid wrapping.
		// See issue 5030.
		if n > 1<<16-1 {
			n = 1<<16 - 1
		}
		return n
	}
	if n == 0 || err != nil {
		return syscall.SOMAXCONN
	}
	// FreeBSD stores the backlog in a uint16, as does Linux.
	// Assume the other BSDs do too. Truncate number to avoid wrapping.
	// See issue 5030.
	if n > 1<<16-1 {
		n = 1<<16 - 1
	}
	return int(n)
}

// NewReusablePortListener returns net.FileListener for a socket with SO_REUSEPORT option.
func NewSocketListener(proto string, addr string, reuse bool, fastopen bool) (l net.Listener, err error) {
	var (
		soType, fd int
		file       *os.File
		sockaddr   syscall.Sockaddr
	)

	if sockaddr, soType, err = getSockaddr(proto, addr); err != nil {
		return nil, err
	}

	switch proto {
	case "tcp", "tcp4", "tcp6":
		if fd, err = syscall.Socket(soType, syscall.SOCK_STREAM, syscall.IPPROTO_TCP); err != nil {
			return nil, err
		}
		// may or may not be supported
		if FastOpenId() > 0 && fastopen {
			err = syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, FastOpenId(), maxListenerBacklog())
		}
	case "udp", "udp4", "udp6":
		if fd, err = syscall.Socket(soType, syscall.SOCK_DGRAM, syscall.IPPROTO_UDP); err != nil {
			return nil, err
		}
	}

	defer func() {
		if err != nil {
			syscall.Close(fd)
		}
	}()

	if reuse {
		if err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
			return nil, err
		}
		if err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, ReusePortID(), 1); err != nil {
			return nil, err
		}
	}

	if err = syscall.Bind(fd, sockaddr); err != nil {
		return nil, err
	}

	if err = syscall.Listen(fd, maxListenerBacklog()); err != nil {
		return nil, err
	}

	// File Name get be nil
	file = os.NewFile(uintptr(fd), "cadent-listener-"+addr)
	if l, err = net.FileListener(file); err != nil {
		return nil, err
	}

	if err = file.Close(); err != nil {
		return nil, err
	}

	return l, err
}

func SetUDPReuse(conn *net.UDPConn) error {
	f, err := conn.File()

	if err != nil {
		return err
	}

	fd := int(f.Fd())

	if err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		return err
	}

	if err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, ReusePortID(), 1); err != nil {
		return err
	}

	if err = syscall.SetNonblock(fd, true); err != nil {
		return err
	}
	return nil
}

type ReuseConn struct {
	listener net.Listener
}

func GetReuseListener(protocal string, addr string) (net.Listener, error) {
	return reuse.Listen(protocal, addr)

}

func GetReusePacketListener(protocal string, addr string) (net.PacketConn, error) {
	return reuse.ListenPacket(protocal, addr)
}
