package main

import (
	"./natgo"
	"encoding/json"
	"flag"
	"github.com/gorilla/websocket"
	"log"
	"net"
	"net/http"
	"time"
)

var remoteServer = "localhost:5007" //host:port
var remoteServerURL string          //host:port
var service = ""                    //targetHost:port
var serviceMap = make(map[string]string)

func init() {
	flag.StringVar(&remoteServer, "remoteServer", remoteServer, "remoteServer")
	flag.StringVar(&service, "service", service, "service")
	flag.Parse()
	remoteServerURL = "ws://" + remoteServer
	if service == "" {
		log.Println("No service defined.")
	}
}

func main() {
	register()
}

func register() {
	conn, err := dial(remoteServerURL + "/register")
	if err != nil {
		log.Println("Failed to dial")
		return
	}

	for {
		msg := natgo.Message{}
		log.Println("Begin read msg...")
		messageType, r, err := conn.NextReader()
		log.Println("message type:", messageType)

		if err != nil {
			log.Println("Failed to read:", err)
			conn.Close()
			return
		}
		if messageType == websocket.CloseMessage {
			log.Println("Received close msg")
			conn.Close()
			return
		}
		err = json.NewDecoder(r).Decode(&msg)
		if err != nil {
			log.Println(err)
			conn.Close()
			return
		}

		switch msg.Type {
		case natgo.CMD_REGISTER_CLIENT_RESPONSE:
			log.Println("Got Register response:" + msg.Message)
			servicePort := msg.Arguments["ServicePort"].(string)
			serviceMap[servicePort] = service

		case natgo.CMD_SERVER_START_SESSION_REQUEST:
			servicePort := msg.Arguments["ServicePort"].(string)
			log.Println("Get service port:", servicePort)
			log.Println("serviceMap:", serviceMap)
			targetAddr := serviceMap[servicePort]
			if targetAddr == "" {
				log.Println("Invalid target address")
			}
			go beginWork(remoteServerURL, targetAddr, msg.Arguments["SessionID"].(string))
			log.Println("Got start session request")
		default:
			log.Println("Unknown msg type:", msg.Type)
		}
	}
}

func dial(url string) (*websocket.Conn, error) {
	dialer := websocket.Dialer{Proxy: http.ProxyFromEnvironment,}

	headers := http.Header{}

	conn, _, err := dialer.Dial(url, headers)
	if err != nil {
		return nil, err
	}

	return conn, err
}

func beginWork(remoteAddr, targetAddr string, sessionId string) {
	log.Println("Connecting target host:", targetAddr)
	targetConn := connectToPort(targetAddr)
	if targetConn == nil {
		log.Println("Failed to connect to target addr")
		return
	}

	log.Println("Connecting to session remoteAddr ", remoteAddr)
	sessionConn, err := dial(remoteServerURL + "/exchange/" + sessionId)
	if err != nil {
		panic(err)
	}
	if sessionConn == nil {
		log.Println("Failed to connect to remote addr")
		return
	}

	log.Println("Begin transfer data ...")

	natgo.ConnectionExchangeWithWebsocket(targetConn, sessionConn)
}

func connectToPort(remoteAddr string) net.Conn {
	conn, err := net.DialTimeout("tcp", remoteAddr, 5*time.Second)
	if err != nil {
		log.Println("Can't connect to addr: ", err)
		return nil
	}
	log.Println("Connected:", conn.RemoteAddr())
	return conn
}
