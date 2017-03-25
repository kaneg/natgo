package main

import (
    "net"
    "sync"
    "fmt"
    "./natgo"
    "os"
    "strings"
    "time"
    "math/rand"
)

var natConnectionPool = make(map[string]net.Conn)
var waitingGuestConnectionPool = make(map[int32]net.Conn)

var mgrChannelLock = sync.Mutex{}

func main() {
    fmt.Println("Start NAT server")
    if (len(os.Args) < 3) {
        fmt.Println("Usage: natgo-server <managerPort>  <servicePort> [servicePort] ...")
        return
    }

    go listenForNAT(os.Args[1])
    listenForGuests(os.Args[2:])
    wg := sync.WaitGroup{}
    wg.Add(1)
    wg.Wait()
}
func listenForNAT(localPort string) {
    fmt.Println("Start NAT thread on port " + localPort)
    ln, err := net.Listen("tcp", ":" + localPort)
    if err != nil {
        fmt.Println(err)
        panic(err)
    }
    fmt.Println("Listen on port:" + localPort, ln.Addr())

    for {
        conn, err := ln.Accept()
        if err != nil {
            fmt.Println(err)
            conn.Close()
        } else {
            fmt.Println("Got a new connection from nat:", conn.RemoteAddr())
            handleClient(conn)
        }
    }
}

func handleClient(conn net.Conn) error {
    fmt.Println("handleClient")
    request := make([]byte, 1)
    conn.Read(request)

    cmd := request[0]
    if (cmd == natgo.CMD_REGISTER_CLIENT_REQUEST) {
        fmt.Println("Get register request from client.")
        portBuffer := make([]byte, 1024)
        conn.Read(portBuffer)
        portStr := string(portBuffer)
        portStr = strings.Trim(portStr, "\x00")
        fmt.Println("Port service from client:", portStr)
        response := make([]byte, 1)
        response[0] = natgo.CMD_REGISTER_CLIENT_RESPONSE
        conn.Write(response)
        fmt.Println("Send client register response")
        services := strings.Split(portStr, ",")
        for _, v := range services {
            oldConn := natConnectionPool[v]
            if oldConn != nil {
                oldConn.Close()
                oldConn = nil
            }
            natConnectionPool[v] = conn
        }
        //go heartbeat(conn)
        go processMgrChannel(conn)
    } else if (cmd == natgo.CMD_CLIENT_REPLY_SESSION_REQUEST) {
        fmt.Println("Got a session connection from client:", conn)
        request = make([]byte, 4)
        conn.Read(request)
        sessionId := natgo.BytesToInt32(request)
        fmt.Println("Get session id from client:", sessionId)
        response := make([]byte, 1)
        response[0] = natgo.CMD_CLIENT_REPLY_SESSION_RESPONSE
        conn.Write(response)
        fmt.Println("Send client reply session response")

        guestConn := waitingGuestConnectionPool[sessionId]
        fmt.Println("Get guest connection from pool:", guestConn)
        delete(waitingGuestConnectionPool, sessionId)
        if (guestConn != nil) {
            natgo.ConnectionExchange(conn, guestConn)
        }
    }

    return nil
}

func processMgrChannel(conn net.Conn) {
    for {
        buffer := make([]byte, 1)
        _, err := conn.Read(buffer)
        if (err != nil) {
            fmt.Println("Failed to read from client, ", err)
            conn.Close()
            return
        }
        cmd := buffer[0]
        if (cmd == natgo.CMD_HEART_BEAT_REQUEST) {
            fmt.Println("Get heartbeat request from client")
            rsp := []byte{natgo.CMD_HEART_BEAT_RESPONSE}
            mgrChannelLock.Lock()
            fmt.Println("Response heartbeat to client")
            _, err = conn.Write(rsp)
            mgrChannelLock.Unlock()
            if (err != nil) {
                fmt.Println("Failed to write client")
                conn.Close()
                return
            }
        } else if (cmd == natgo.CMD_SERVER_START_SESSION_RESPONSE) {
            //do nothing for now
            fmt.Println("Get response from client for start session")
        }
    }
}

func listenForGuests(guestPorts []string) {
    for _, port := range guestPorts {
        go listenForGuest(port)
    }
}

func listenForGuest(guestPort string) {
    fmt.Println("Start Guest thread on port " + guestPort)
    ln, err := net.Listen("tcp", ":" + guestPort)
    if err != nil {
        fmt.Println(err)
        panic(err)
    }
    fmt.Println("Listen on port:" + guestPort, ln.Addr())
    defer ln.Close()

    for {
        guestConn, err := ln.Accept()
        if err != nil {
            fmt.Println(err)
            guestConn.Close()
        } else {
            fmt.Println("Got a connection from guest:", guestConn.RemoteAddr())
            natConn := natConnectionPool[guestPort]
            if natConn != nil {
                connectNATAndGuest(natConn, guestConn, guestPort)
            } else {
                fmt.Println("No client register for this port, close guest.")
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
    if (err == nil) {
        fmt.Println("Get correct response, begin start session")
        waitingGuestConnectionPool[sessionId] = guestConn
        go timeoutForGuestConnection(sessionId)
    } else {
        fmt.Print("Get invalid response:")
    }
}

func timeoutForGuestConnection(sessionId int32) {
    time.Sleep(5 * time.Second)
    conn, ok := waitingGuestConnectionPool[sessionId]
    if (ok) {
        fmt.Println("Timeout to wait for connection from client, close the guest.")
        conn.Close()
        delete(waitingGuestConnectionPool, sessionId)
    }
}
