/*
Copyright 2016 Under Armour, Inc.

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
