package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/lelandbatey/blink"
	flag "github.com/spf13/pflag"
)

func blinker(c chan bool) func() {
	return func() {
		for {
			<-c
			blink.Do(500 * time.Millisecond)
			time.Sleep(500 * time.Millisecond)
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
	//addr := "127.0.0.1:6754"
	for {
		c, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err != nil {
			//panic(err)
			log.Printf("Failed to connect to address: %q", addr)
			time.Sleep(10 * time.Second)
			continue
		}
		log.Printf("Successfully connected to address: %q", addr)
		buf := make([]byte, 1)
		for {
			err = readConn(addr, c, blinkchan)
			n, err := c.Read(buf)
			if err != nil {
				log.Printf("Reading from connection failed: %v", err)
				break
			}
			fmt.Printf("Read '%v' bytes from connection\n", n)
			blinkchan <- true
		}
	}
}
