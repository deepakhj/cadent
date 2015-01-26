package main

import (
	"net"
    "time"
    "log"
)

func main() {

    testConnections()
}


func testConnections() {

    // async this guy

    servers := [...]string{"google.com:80", "localhost:234", "localhost:456"}
    ok := true
    
    testUp := func(server string, out chan bool){
        log.Printf("Checking %s", server)
        _, err := net.DialTimeout("tcp", server, 5 * time.Second)
        if err != nil {
            log.Printf("Checking %s Error: %s",server, err)
            out <- false
        }
        out <- true
    }

    gotResponse :=  func(serv string, in <-chan bool){
		message := <-in
        log.Printf("Got %s-%v", serv, message)
        time.Sleep(1 * time.Second)
    }

    for ok {
        for _, serv := range servers {
            log.Printf("ON %s", serv)
            channel := make(chan bool)
            defer close(channel)
            go testUp(serv, channel)
            go gotResponse(serv, channel)
        }
            time.Sleep(20 * time.Second)
	}

}