package main

import (
	"flag"
	"github.com/gorilla/websocket"
	"github.com/kaneg/flaskgo"
	"github.com/kaneg/natgo/natgo"
	"github.com/phayes/freeport"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var port = 5007
var useFreePort bool
var servicePorts = ""
var waitingGuestConnectionPool = make(map[string]net.Conn)
var servicePortListener = make(map[int]*net.Listener)
var servicePool = make(map[int]*websocket.Conn)

func init() {
	flag.IntVar(&port, "port", port, "Listen Port")
	flag.BoolVar(&useFreePort, "useFreePort", useFreePort, "useFreePort")
	flag.StringVar(&servicePorts, "servicePorts", servicePorts, "servicePorts")
	flag.Parse()

	initServicePool()
}

func initServicePool() {
	if servicePorts != "" {
		log.Println("servicePorts:", servicePorts)
		ports := strings.Split(servicePorts, ",")
		for _, portStr := range ports {
			port, e := strconv.Atoi(portStr)
			if e == nil {
				servicePool[port] = nil
			}
		}
		log.Println("servicePortsPool:", servicePool)
	}
}

type NatGo struct {
	app *flaskgo.App
}

func getAvailablePort() int {
	if useFreePort {
		return freeport.GetPort()
	} else {
		for port, guest := range servicePool {
			if guest == nil {
				return port //todo: concurrent
			}
		}
		log.Println("No available.")
		return 0
	}
}

func (natGo *NatGo) register() {
	r := flaskgo.GetRequest()
	log.Println("Get a client register:", r.RemoteAddr)
	w := flaskgo.GetResponseWriter()
	conn := upgradeWebSocket(w, r)
	servicePort := getAvailablePort()
	log.Println("Allocate a available port:", servicePort)
	if servicePort == 0 {
		log.Println("No available to serve, discard register request.")
		expectedErr := &websocket.CloseError{Code: websocket.CloseNormalClosure, Text: "No available to serve, discard register request."}
		conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(expectedErr.Code, expectedErr.Text),
			time.Now().Add(3*time.Second))
		conn.Close()
		return
	}
	conn.WriteJSON(natgo.Message{Type: natgo.CMD_REGISTER_CLIENT_RESPONSE,
		Message:   "Register Allowed:" + strconv.Itoa(servicePort),
		Arguments: map[string]interface{}{"ServicePort": strconv.Itoa(servicePort)}})
	servicePool[servicePort] = conn
	go listenOnServicePort(servicePort)
}

func heartBeat() {
	tick := time.Tick(time.Second * 5)
	for range tick {
		for servicePort, conn := range servicePool {
			if conn != nil {
				err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(3*time.Second))
				if err != nil {
					conn.Close()
					releaseService(servicePort)
				}
			}
		}
	}
}

func releaseService(servicePort int) {
	log.Println("Release service port:", servicePort)
	servicePool[servicePort] = nil //return back the service port
}

func (natGo *NatGo) exchange(sessionId string) {
	guestConn := waitingGuestConnectionPool[sessionId]
	log.Println("Get guest connection from pool:", guestConn)
	delete(waitingGuestConnectionPool, sessionId)
	if guestConn != nil {
		w := flaskgo.GetResponseWriter()
		r := flaskgo.GetRequest()
		conn := upgradeWebSocket(w, r)
		natgo.ConnectionExchangeWithWebsocket(guestConn, conn)
	}
}
func upgradeWebSocket(writer http.ResponseWriter, request *http.Request) *websocket.Conn {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  2048,
		WriteBufferSize: 2048,
	}
	conn, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		panic(nil)
	}
	return conn
}

func initRoute(natGo *NatGo) {
	natGo.app.AddRoute("/register", natGo.register)
	natGo.app.AddRoute("/exchange/<sessionId>", natGo.exchange)
}

func listenOnServicePort(servicePort int) {
	if servicePortListener[servicePort] != nil {
		log.Println("Port is already listening")
		return
	}
	log.Println("Start Guest thread on port ", servicePort)
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(servicePort))
	if err != nil {
		log.Println(err)
		panic(err)
	}
	servicePortListener[servicePort] = &ln
	log.Println("Listen on port:", servicePort, ln.Addr())
	defer ln.Close()

	for {
		guestConn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			guestConn.Close()
		} else {
			log.Println("Got a connection from guest:", guestConn.RemoteAddr())
			natConn := servicePool[servicePort]
			if natConn != nil {
				connectNATAndGuest(natConn, guestConn, servicePort)
			} else {
				log.Println("No client register for this port, close guest.")
				guestConn.Close()
			}
		}
	}
}
func connectNATAndGuest(natConn *websocket.Conn, guestConn net.Conn, servicePort int) {
	sessionId := natgo.NewUUID()
	msg := natgo.Message{Type: natgo.CMD_SERVER_START_SESSION_REQUEST,
		Arguments: map[string]interface{}{"SessionID": sessionId, "ServicePort": strconv.Itoa(servicePort)}}
	log.Println("Send start session request")
	err := natConn.WriteJSON(msg)
	if err == nil {
		waitingGuestConnectionPool[sessionId] = guestConn
		go timeoutForGuestConnection(sessionId)
	} else {
		log.Println("Release service port.")
		natConn.Close()
		releaseService(servicePort)
		log.Print("Get invalid response:")
	}
}
func timeoutForGuestConnection(sessionId string) {
	time.Sleep(5 * time.Second)
	conn, ok := waitingGuestConnectionPool[sessionId]
	if ok {
		log.Println("Timeout to wait for connection from client, close the guest.")
		conn.Close()
		delete(waitingGuestConnectionPool, sessionId)
	}
}

func main() {
	app := flaskgo.CreateApp()
	natGo := NatGo{&app}
	initRoute(&natGo)
	log.Println("Started")
	go heartBeat()
	app.Run(":" + strconv.Itoa(port))
}
