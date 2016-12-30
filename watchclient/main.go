package main

import (
	"log"
	"net"
	"time"

	"github.com/lelandbatey/blink"
	"github.com/lelandbatey/watchserver/communication"
	flag "github.com/spf13/pflag"
)

func blinker(c chan bool) func() {
	return func() {
		for {
			<-c
			blink.Do(200 * time.Millisecond)
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func readConn(addr string, c net.Conn, blinkchan chan bool) error {
	buf := make([]byte, 1)
	for {
		n, err := c.Read(buf)
		if err != nil {
			return err
		}
		log.Printf("Read '%v' bytes from connection\n", n)
		blinkchan <- true
	}
}

func main() {
	var host = flag.String("host", "127.0.0.1", "Host to attempt to connect to")
	var port = flag.String("port", "6754", "Port to attempt to make connection on")
	flag.Parse()

	blinkchan := make(chan bool, 10)
	go blinker(blinkchan)()
	addr := *host + ":" + *port
	for {
		log.Printf("Attempting to connect to address %q...", addr)
		con, err := communication.New(addr)
		if err != nil {
			log.Printf("Failed to connect to address: %q", addr)
			time.Sleep(10 * time.Second)
			continue
		}
		log.Printf("Successfully connected to address: %q", addr)
		for {
			if !con.Alive() {
				log.Printf("Connection died, restarting connection")
				break
			}
			select {
			case <-con.Notification:
				blinkchan <- true
			case err = <-con.Errs:
				log.Printf("Error '%v'", err)
			}
		}
	}
}
