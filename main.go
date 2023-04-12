package main

import (
	"io"
	"log"
	"net"
	"strconv"
)

func main() {
	listener, err := net.Listen("tcp", ":51080")
	if err != nil {
		log.Fatalf("Error listening on port 51080: %v", err)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 1024)

	// handshake
	_, err := conn.Read(buf)
	if err != nil {
		log.Printf("Error reading handshake: %v", err)
		return
	}
	if buf[0] != 5 { // SOCKS5 only
		log.Printf("Unsupported SOCKS version: %d", buf[0])
		return
	}
	conn.Write([]byte{5, 0}) // response

	// request
	_, err = conn.Read(buf)
	if err != nil {
		log.Printf("Error reading request: %v", err)
		return
	}
	if buf[0] != 5 || buf[1] != 1 { // SOCKS5, connect only
		log.Printf("Unsupported command: %v", buf[1])
		return
	}
	switch buf[3] {
	case 1: // IPv4 address
		addr := net.IPv4(buf[4], buf[5], buf[6], buf[7])
		port := uint16(buf[8])<<8 + uint16(buf[9])
		log.Printf("Connecting to %v:%v", addr, port)
		remote, err := net.Dial("tcp", net.JoinHostPort(addr.String(), strconv.Itoa(int(port))))
		if err != nil {
			log.Printf("Error connecting to %v:%v: %v", addr, port, err)
			return
		}
		defer remote.Close()
		conn.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0}) // response
		go io.Copy(remote, conn)
		io.Copy(conn, remote)
	case 3: // domain name
		addrlen := int(buf[4])
		addr := string(buf[5 : 5+addrlen])
		port := uint16(buf[5+addrlen])<<8 + uint16(buf[5+addrlen+1])
		log.Printf("Connecting to %v:%v", addr, port)
		remote, err := net.Dial("tcp", net.JoinHostPort(addr, strconv.Itoa(int(port))))
		if err != nil {
			log.Printf("Error connecting to %v:%v: %v", addr, port, err)
			return
		}
		defer remote.Close()
		conn.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0}) // response
		go io.Copy(remote, conn)
		io.Copy(conn, remote)
	case 4: // IPv6 address
		log.Printf("Unsupported IPv6 address: %v", buf[3])
		return
	default:
		log.Printf("Unsupported address type: %v", buf[3])
		return
	}
}
