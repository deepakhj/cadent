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

package gossip

import (
	. "github.com/smartystreets/goconvey/convey"
	"net"
	"sync"
	"testing"
	"time"
)

/*** NOTE:

you will need to


# Setup loopback
for ((i=2;i<256;i++))
do
    sudo ifconfig lo0 alias 127.0.0.$i up
done

on OSX

*/

var bindLock sync.Mutex
var bindNum byte = 10

func getBindAddr() net.IP {
	bindLock.Lock()
	defer bindLock.Unlock()

	result := net.IPv4(127, 0, 0, bindNum)
	bindNum++
	if bindNum > 255 {
		bindNum = 10
	}

	return result
}

func TestGossiping(t *testing.T) {

	// set up 3 gossipers
	b1 := getBindAddr()
	Start("lan", 12312, b1.String(), b1.String())

	b2 := getBindAddr()
	m2, err := Start("lan", 12312, b2.String(), b2.String())
	t.Logf("JOIN: %v", err)
	m2.Join(b1.String())

	b3 := getBindAddr()
	m3, err := Start("lan", 12312, b3.String(), b3.String())
	t.Logf("JOIN: %v", err)

	m3.Join(b1.String())

	time.Sleep(time.Second * 5) // wait for things to settle

	Convey("We should gossip", t, func() {

		So(Get().List.NumMembers(), ShouldEqual, 3)

	})

}
