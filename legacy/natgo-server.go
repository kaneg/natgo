package legacy

import (
    "net"
    "sync"
    "../natgo"
    "os"
    "strings"
    "time"
    "math/rand"
    "log"
)

var natConnectionPool = make(map[string]net.Conn)
var waitingGuestConnectionPool = make(map[int32]net.Conn)

var mgrChannelLock = sync.Mutex{}

func main() {
    log.Println("Start NAT server")
    if len(os.Args) < 3 {
        log.Println("Usage: natgo-server <managerPort>  <servicePort> [servicePort] ...")
        return
    }

    go listenForNAT(os.Args[1])
    listenForGuests(os.Args[2:])
    wg := sync.WaitGroup{}
    wg.Add(1)
    wg.Wait()
}
func listenForNAT(localPort string) {
    log.Println("Start NAT thread on port " + localPort)
    ln, err := net.Listen("tcp", ":" + localPort)
    if err != nil {
        log.Println(err)
        panic(err)
    }
    log.Println("Listen on port:" + localPort, ln.Addr())

    for {
        conn, err := ln.Accept()
        if err != nil {
            log.Println(err)
            conn.Close()
        } else {
            log.Println("Got a new connection from nat:", conn.RemoteAddr())
            go handleClient(conn)
        }
    }
}

func handleClient(conn net.Conn) error {
    log.Println("handleClient")
    request := make([]byte, 1)
    conn.Read(request)

    cmd := request[0]
    if cmd == natgo.CMD_REGISTER_CLIENT_REQUEST {
        log.Println("Get register request from client.")
        portBuffer := make([]byte, 1024)
        conn.Read(portBuffer)
        portStr := string(portBuffer)
        portStr = strings.Trim(portStr, "\x00")
        log.Println("Port service from client:", portStr)
        response := make([]byte, 1)
        response[0] = natgo.CMD_REGISTER_CLIENT_RESPONSE
        conn.Write(response)
        log.Println("Send client register response")
        services := strings.Split(portStr, ",")
        for _, v := range services {
            oldConn := natConnectionPool[v]
            if oldConn != nil {
                oldConn.Close()
                oldConn = nil
            }
            natConnectionPool[v] = conn
        }
        go processMgrChannel(conn)
    } else if cmd == natgo.CMD_CLIENT_REPLY_SESSION_REQUEST {
        log.Println("Got a session connection from client:", conn)
        request = make([]byte, 4)
        conn.Read(request)
        sessionId := natgo.BytesToInt32(request)
        log.Println("Get session id from client:", sessionId)
        response := make([]byte, 1)
        response[0] = natgo.CMD_CLIENT_REPLY_SESSION_RESPONSE
        conn.Write(response)
        log.Println("Send client reply session response")

        guestConn := waitingGuestConnectionPool[sessionId]
        log.Println("Get guest connection from pool:", guestConn)
        delete(waitingGuestConnectionPool, sessionId)
        if guestConn != nil {
            natgo.ConnectionExchange(conn, guestConn)
        }
    }

    return nil
}

func processMgrChannel(conn net.Conn) {
    for {
        buffer := make([]byte, 1)
        _, err := conn.Read(buffer)
        if err != nil {
            log.Println("Failed to read from client, ", err)
            conn.Close()
            return
        }
        cmd := buffer[0]
        if cmd == natgo.CMD_HEART_BEAT_REQUEST {
            log.Println("Get heartbeat request from client")
            rsp := []byte{natgo.CMD_HEART_BEAT_RESPONSE}
            mgrChannelLock.Lock()
            log.Println("Response heartbeat to client")
            _, err = conn.Write(rsp)
            mgrChannelLock.Unlock()
            if err != nil {
                log.Println("Failed to write client")
                conn.Close()
                return
            }
        } else if cmd == natgo.CMD_SERVER_START_SESSION_RESPONSE {
            //do nothing for now
            log.Println("Get response from client for start session")
        }
    }
}

func listenForGuests(guestPorts []string) {
    for _, port := range guestPorts {
        go listenForGuest(port)
    }
}

func listenForGuest(guestPort string) {
    log.Println("Start Guest thread on port " + guestPort)
    ln, err := net.Listen("tcp", ":" + guestPort)
    if err != nil {
        log.Println(err)
        panic(err)
    }
    log.Println("Listen on port:" + guestPort, ln.Addr())
    defer ln.Close()

    for {
        guestConn, err := ln.Accept()
        if err != nil {
            log.Println(err)
            guestConn.Close()
        } else {
            log.Println("Got a connection from guest:", guestConn.RemoteAddr())
            natConn := natConnectionPool[guestPort]
            if natConn != nil {
                connectNATAndGuest(natConn, guestConn, guestPort)
            } else {
                log.Println("No client register for this port, close guest.")
                guestConn.Close()
            }
        }
    }
}

func connectNATAndGuest(natConn net.Conn, guestConn net.Conn, guestPort string) {
    var sessionId int32 = rand.Int31()
    mgrChannelLock.Lock()
    err := natgo.ServerStartSessionRequest(natConn, sessionId, guestPort)
    mgrChannelLock.Unlock()
    if err == nil {
        log.Println("Get correct response, begin start session")
        waitingGuestConnectionPool[sessionId] = guestConn
        go timeoutForGuestConnection(sessionId)
    } else {
        log.Print("Get invalid response:")
    }
}

func timeoutForGuestConnection(sessionId int32) {
    time.Sleep(5 * time.Second)
    conn, ok := waitingGuestConnectionPool[sessionId]
    if ok {
        log.Println("Timeout to wait for connection from client, close the guest.")
        conn.Close()
        delete(waitingGuestConnectionPool, sessionId)
    }
}
