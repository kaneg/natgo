package natgo

import (
	"crypto/rand"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net"
)

type Message struct {
	Type      int
	Message   string
	Arguments map[string]interface{}
}

func NewUUID() (string) {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		panic(err)
	}

	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
}

func ConnectionExchangeWithWebsocket(src net.Conn, dst *websocket.Conn) {
	go transferDataWebToNet(dst, src)
	go transferDataNetToWeb(src, dst)
}

func transferDataNetToWeb(src net.Conn, dst *websocket.Conn) {
	var buffer = make([]byte, 32*1024)
	for {
		size, err := src.Read(buffer)
		if err == nil {
			value := buffer[:size]
			err = dst.WriteMessage(websocket.BinaryMessage, value)

		}

		if err != nil {
			log.Println("Error during transferDataNetToWeb", err)
			break
		}
	}
	src.Close()
	dst.Close()
}
func transferDataWebToNet(src *websocket.Conn, dst net.Conn) {
	for {
		messageType, r, err := src.NextReader()

		if err == nil {
			if messageType == websocket.BinaryMessage {
				_, err = io.Copy(dst, r)
			}
		}

		if err != nil {
			fmt.Println("Error during transferDataWebToNet", err)
			break
		}
	}

	src.Close()
	dst.Close()
}
