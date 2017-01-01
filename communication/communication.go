package communication

import (
	"log"
	"net"
	"reflect"
	"time"
)

type Connection struct {
	alive        bool
	conn         net.Conn
	Errs         chan error
	Notification chan []byte
}

func (c *Connection) Alive() bool {
	return c.alive
}

func New(addr string) (*Connection, error) {
	con, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	rv := Connection{
		alive:        true,
		conn:         con,
		Notification: make(chan []byte, 10),
		Errs:         make(chan error),
	}
	go watchConnection(&rv)
	return &rv, nil
}

func watchConnection(con *Connection) {
	buf := make([]byte, 1)
	for {
		con.conn.SetDeadline(time.Now().Add(15 * time.Second))
		_, err := con.conn.Read(buf)
		if err != nil {
			log.Printf("There was an error: %v", err)
			con.Errs <- err
			err = con.conn.Close()
			if err != nil {
				log.Printf("There was an error while closing the connection: %v", err)
				con.Errs <- err
			}
			con.alive = false
			return
		}
		if reflect.DeepEqual(buf, []byte{'\x00'}) {
			con.Notification <- buf
		}
	}
}
