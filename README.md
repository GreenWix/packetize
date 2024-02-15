## packetize

A simple library for creating packetized TCP server using Go.

## Installation

```
go get github.com/GreenWix/packetize
```

## Packet Format

Packet is a byte array with maximum length of 65,535 bytes.
Packets are sent in this format:

```<packet length : uint16> <packet data : bytes>```

## Example

```go
package main

import (
	"fmt"
	pkize "github.com/GreenWix/packetize"
	"net"
)

func main() {
	listener, _ := pkize.Listen("tcp4", "localhost:1337") // open listening socket
	for {
		conn, err := listener.Accept() // accept TCP connection from client and create a new socket
		if err != nil {
			continue
		}
		go handleClient(conn) // handle client connection in a separate goroutine
	}
}

func handleClient(conn net.Conn) {
	defer conn.Close() // close socket after done

	buf := make([]byte, 32) // buffer for reading data from client
	for {
		conn.Write([]byte("Hello world!")) // write to the client

		readLen, err := conn.Read(buf) // reading from the socket
		if err != nil {
			fmt.Println("error occured during client handling: ", err)
			break
		}

		fmt.Println(string(buf[:readLen]))
	}
}
```
