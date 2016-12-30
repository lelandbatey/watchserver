package main

import (
	"log"
	"net"
	"os"

	"github.com/fsnotify/fsnotify"
	flag "github.com/spf13/pflag"
)

// Creates a TCP server which watches a particular file system path provided by
// the user via command line argument. That server accepts any connection on
// and whenever it receives any file system event for the directory it's
// watching, it writes a single null byte across all connections.
func main() {

	var host = flag.String("host", "0.0.0.0", "Host to attempt to connect to")
	var port = flag.String("port", "6754", "Port to attempt to make connection on")
	flag.Parse()

	// Boilerplate for keeping track of client connections
	cons := []net.Conn{}
	_addcon := func(c net.Conn) {
		cons = append(cons, c)
	}
	_rmcon := func(c net.Conn) {
		idx := -1
		// Find index of item in collection
		for i, v := range cons {
			if c == v {
				idx = i
			}
		}
		// If item not found, ignore
		if idx == -1 {
			return
		}
		// Remove item from collection
		cons = append(cons[:idx], cons[idx+1:]...)
	}
	rm := make(chan net.Conn)
	add := make(chan net.Conn)
	go func() {
		for {
			select {
			case c := <-add:
				_addcon(c)
			case c := <-rm:
				_rmcon(c)
			}
		}
	}()

	addr := *host + ":" + *port
	server, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}

	// Listen for new connections
	go func() {
		for {
			c, err := server.Accept()
			if c == nil || err != nil {
				log.Printf("Failed to accept connection %v for reason %q\n", c, err)
				continue
			}
			log.Printf("Recieved a new connection: %v\n", c.RemoteAddr())
			add <- c
		}
	}()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	// Listen for fs events forever
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				badcons := []net.Conn{}
				// If we got an event, write a null byte across each connection
				for _, c := range cons {
					bw, err := c.Write([]byte{'\x00'})
					// If we couldn't do either of those, then the connection
					// is bad and we should remove that connection from our
					// connection list
					if err != nil || bw != 1 {
						badcons = append(badcons, c)
						log.Println("Marking connection as bad:", c.LocalAddr())
					} else {
						log.Printf("Sent message to connection: %v\n", c.LocalAddr())
					}
				}
				// Remove the bad connections
				for _, bc := range badcons {
					bc.Close()
					rm <- bc
					//rmcon(bc)
				}
				log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("modified file:", event.Name)
				}
			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	<-done
}
