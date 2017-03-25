package main

import (
    "fmt"
    "net"
    "./natgo"
    "sync"
    "os"
    "time"
    "strings"
    "errors"
)

var serviceMap = make(map[string]string)
var mgrChannelLock = sync.Mutex{}

var isServerAlive = true

func main() {
    fmt.Println("Start NAT client")
    if (len(os.Args) < 3) {
        fmt.Println("Usage: natgo-client <remoteHost:port>  <servicePort:targetHost:port> [servicePort:targetHost:port] ")
        return
    }
    remoteAddr := os.Args[1]
    targetAddrs := os.Args[2:]
    var services = ""
    for _, targetAddr := range targetAddrs {
        i := strings.Index(targetAddr, ":")
        service := targetAddr[0:i]
        targetAddr = targetAddr[i + 1:]
        serviceMap[service] = targetAddr
        services += service
        services += ","
    }
    services = strings.Trim(services, ",")
    fmt.Println(services)

    fmt.Println("Connect to control address  ", remoteAddr)
    wg := sync.WaitGroup{}
    wg.Add(1)
    for {
        connRemote, err := net.DialTimeout("tcp", remoteAddr, 10 * time.Second)
        if err != nil {
            fmt.Println("Can't connect to remote: ", err)
            time.Sleep(10 * time.Second)
            continue
        }

        connRemote.SetReadDeadline(time.Now().Add(5 * time.Second))

        err = natgo.ClientRegisterRequest(connRemote, services)
        if (err != nil) {
            fmt.Println("Can not register to server.")
            continue
        }
        connRemote.SetReadDeadline(time.Time{})

        go heartBeat(connRemote)
        for {

            sessionId, service, err := serverStartSessionResponse(connRemote)
            if (err == nil) {
                targetAddr := serviceMap[service]
                work(remoteAddr, targetAddr, sessionId)
            } else {
                fmt.Println("Failed to start session with server.")
                connRemote.Close()
                break
            }
        }
    }

    wg.Wait()
}

func heartBeat(conn net.Conn) {
    for {
        time.Sleep(30 * time.Second)
        fmt.Println("Begin heart beat")
        fmt.Println("Get mgr conn lock")
        mgrChannelLock.Lock()
        isServerAlive = false
        _, err := conn.Write([]byte{natgo.CMD_HEART_BEAT_REQUEST})
        if (err != nil) {
            fmt.Println("Failed to send heartbeat request to server, close the connection.", err)
            conn.Close()
            mgrChannelLock.Unlock()
            return
        }

        time.Sleep(2 * time.Second) //waiting for response

        if (isServerAlive) {
            fmt.Println("Get heartbeat response from server")
        } else {
            fmt.Println("Failed to get heartbeat response, close the connection.", err)
            conn.Close()
            mgrChannelLock.Unlock()
            return
        }
        fmt.Println("Release mgr lock")
        mgrChannelLock.Unlock()
        fmt.Println("Done heartbeat")
    }
}

func work(remoteAddr, targetAddr string, sessionId int32) {
    fmt.Println("Connecting target host ", targetAddr)
    targetConn := connectPort(targetAddr)
    if targetConn == nil {
        fmt.Println("Failed to connect to target addr")
        return
    }

    fmt.Println("Connecting to session remoteAddr ", remoteAddr)
    sessionConn := connectPort(remoteAddr)
    if (sessionConn == nil) {
        fmt.Println("Failed to connect to remote addr")
        return
    }
    natgo.ClientReplySessionRequest(sessionConn, sessionId)

    fmt.Println("Begin transfer data ...")

    natgo.ConnectionExchange(targetConn, sessionConn)
}

func connectPort(remoteAddr string) net.Conn {
    conn, err := net.DialTimeout("tcp", remoteAddr, 5 * time.Second)
    if (err != nil) {
        fmt.Println("Can't connect to addr: ", err)
        return nil
    }
    fmt.Println("Connectted:", conn.RemoteAddr())
    return conn
}

func serverStartSessionResponse(conn net.Conn) (int32, string, error) {
    fmt.Println("ServerStartSessionResponse")
    for {
        request := make([]byte, 20)
        fmt.Println("Begin read cmd...")
        _, err := conn.Read(request)
        if (err != nil) {
            return 0, "", err
        }
        fmt.Println("Read cmd:", request)
        if (request[0] == natgo.CMD_HEART_BEAT_RESPONSE) {
            fmt.Println("Get heart beat cmd.")
            isServerAlive = true
            continue
        } else if (request[0] != natgo.CMD_SERVER_START_SESSION_REQUEST) {
            fmt.Println("Invalid cmd")
            return 0, "", errors.New("Invalid cmd from server")
        }
        requestId := natgo.BytesToInt32(request[1:5])
        fmt.Println("request id:", requestId)

        serviceBuffer := request[5:]

        service := string(serviceBuffer)
        service = strings.Trim(service, "\x00")
        fmt.Println("Client got service:", service)

        response := make([]byte, 1)
        response[0] = natgo.CMD_SERVER_START_SESSION_RESPONSE
        fmt.Println("Sending response ", response)
        mgrChannelLock.Lock()
        conn.Write(response)
        mgrChannelLock.Unlock()
        fmt.Println("Sent CMD_SERVER_START_SESSION_RESPONSE response")
        return requestId, service, nil
    }
}
