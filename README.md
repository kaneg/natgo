# Natgo
A tool that can help nodes behinds NAT be accessed by outside.


# Usage

## Server
#### Usage

```
Usage: natgo-server <managerPort>  <servicePort> [servicePort] ...
```

#### Example:

```
$ natgo-server 5000 6022 6080
Start NAT server
Start Guest thread on port 6080
Listen on port:6080 [::]:6080
Start NAT thread on port 5000
Listen on port:5000 [::]:5000
Start Guest thread on port 6022
Listen on port:6022 [::]:6022
```
The serve will listen on management port 5000 and listen on service port 6022 and 6080. Then the client can map the port 6022 for its port 22 and 6080 for its port 80.

## Client

#### Usage
```
Usage: natgo-client <remoteHost:port>  <servicePort:targetHost:port> [servicePort:targetHost:port]
```

#### Example

```
$ natgo-client <server>:5000 6022:localhost:22 6080:localhost:80
```
The client is a node behind a NAT which can only connect the server. After connecting to server, the user can connect to the client's ssh port by connecting server's 6022 port and client's http port by server's 6080 port.

## Security

For now, the tool is only a port forwarder. It is not responsible for authentication of the client and server. The security should be considered by the application which providing the service.

# License

The Apache2.0 License