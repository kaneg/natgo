package natgo

import (
    "fmt"
    "net"
    "errors"
    "encoding/binary"
    _"time"
)

const CMD_HEART_BEAT_REQUEST = 0
const CMD_HEART_BEAT_RESPONSE = 1
const CMD_REGISTER_CLIENT_REQUEST = 2
const CMD_REGISTER_CLIENT_RESPONSE = 3

const CMD_SERVER_START_SESSION_REQUEST = 4
const CMD_SERVER_START_SESSION_RESPONSE = 5

const CMD_CLIENT_REPLY_SESSION_REQUEST = 6
const CMD_CLIENT_REPLY_SESSION_RESPONSE = 7

func HeartBeatRequest(conn net.Conn) error {
    fmt.Println("ClientHeartBeatRequest")
    request := make([]byte, 1)
    request[0] = CMD_HEART_BEAT_REQUEST
    _, err := conn.Write(request)
    fmt.Println("Write to peer.")

    if (err != nil) {
        fmt.Println("Failed to write to peer.")
    }
    return err
}

func ClientRegisterRequest(conn net.Conn, service string) error {
    fmt.Println("ClientRegisterRequest")
    request := make([]byte, 1)
    request[0] = CMD_REGISTER_CLIENT_REQUEST
    serviceBytes := []byte(service)
    conn.Write(request)
    fmt.Println("Write to server:", service, serviceBytes)
    size, err := conn.Write(serviceBytes)
    fmt.Println("write result:", size, err)
    if (err != nil) {
        return err
    }
    response := make([]byte, 1)
    _, err = conn.Read(response)
    if (err != nil) {
        return err
    }
    if (response[0] != CMD_REGISTER_CLIENT_RESPONSE) {
        return errors.New("Invalid response")
    } else {
        fmt.Println("Get client register response")
    }
    return nil
}

func ClientReplySessionRequest(conn net.Conn, sessionId int32) error {
    fmt.Println("ClientReplySessionRequest, session:", sessionId)
    data := append([]byte{CMD_CLIENT_REPLY_SESSION_REQUEST}, Int32ToBytes(sessionId)...)

    conn.Write(data)
    response := make([]byte, 1)
    conn.Read(response)
    if (response[0] != CMD_CLIENT_REPLY_SESSION_RESPONSE) {
        return errors.New("Invalid response")
    } else {
        fmt.Println("Get client reply session response ")
    }
    return nil
}

func ServerStartSessionRequest(conn net.Conn, requestId int32, guestPort string) error {
    fmt.Println("ServerStartSessionRequest")
    cmd := []byte{CMD_SERVER_START_SESSION_REQUEST}
    var data = append(cmd, Int32ToBytes(requestId)...)
    data = append(data, []byte(guestPort)...)
    conn.Write(data)
    return nil
}

func transferData(src, dst net.Conn) {
    var buffer = make([]byte, 4096)
    for {
        size, err := src.Read(buffer)
        if err != nil {
            fmt.Println("error during transferData", err)
            src.Close()
            dst.Close()
            break
        }
        value := buffer[:size]

        _, err2 := dst.Write(value)
        if err2 != nil {
            fmt.Println("error during transferData", err2)
            src.Close()
            dst.Close()
            break
        }
    }
}

func ConnectionExchange(src, dst net.Conn) {
    go transferData(src, dst)
    go transferData(dst, src)
}

func Int64ToBytes(i int64) []byte {
    var buf = make([]byte, 8)
    binary.BigEndian.PutUint64(buf, uint64(i))
    return buf
}

func BytesToInt64(buf []byte) int64 {
    return int64(binary.BigEndian.Uint64(buf))
}

func Int32ToBytes(i int32) []byte {
    var buf = make([]byte, 4)
    binary.BigEndian.PutUint32(buf, uint32(i))
    return buf
}

func BytesToInt32(buf []byte) int32 {
    return int32(binary.BigEndian.Uint32(buf))
}
