/*
   To allow net.UDPcon to have various Nice kernel flags used

*/

package cadent

import (
	reuse "github.com/jbenet/go-reuseport"
	"net"
)

// for TCP conns
func GetReuseListener(protocal string, addr string) (net.Listener, error) {
	return reuse.Listen(protocal, addr)

}

// for UDP conns
func GetReusePacketListener(protocal string, addr string) (net.PacketConn, error) {
	return reuse.ListenPacket(protocal, addr)
}
