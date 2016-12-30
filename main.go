package main

import (
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/rjeczalik/notify"
	flag "github.com/spf13/pflag"
)

func isDir(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.IsDir()
}

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
			// Create a goroutine which sends heartbeat informantion. If
			// heartbeat fails, close the connection and stop trying to send
			// heartbeat so the connection will be cleaned up by the fs notify
			// loop.
			go func() {
				for {
					bw, err := c.Write([]byte{'\x01'})
					if err != nil || bw != 1 {
						c.Close()
						return
					}
					time.Sleep(10 * time.Second)
				}
			}()
			add <- c
		}
	}()

	event_chan := make(chan notify.EventInfo, 1)
	path, err := filepath.Abs(os.Args[1])
	if err != nil {
		log.Fatal("Could not find absolute path for %q: %v", os.Args[1], err)
	}
	if isDir(path) {
		path = path + "/..."
	}
	if err = notify.Watch(path, event_chan, notify.All); err != nil {
		log.Fatalf("Could not register notify on location %q: %v", os.Args[1], err)
	}

	done := make(chan bool)
	// Listen for fs events forever
	go func() {
		for {
			select {
			case event := <-event_chan:
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
					}
				}
				// Remove the bad connections
				for _, bc := range badcons {
					bc.Close()
					rm <- bc
				}
				log.Println("event:", event)
			}
		}
	}()

	<-done
}
