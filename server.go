package main

import (
	"bytes"
	"io"
	"log"
	"net"
	"regexp"
)

func main() {
	l, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Panic(err)
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Panic(err)
		}

		go handleConn(conn)
	}
}

func parseHostPort(b []byte) (host string, port string) {
	log.Println(string(b))
	p := regexp.MustCompile(`Host: (.*?):?(\d+)?\r\n`)
	res := p.FindSubmatch(b)
	if len(res) == 3 {
		host, port = string(res[1]), string(res[2])
		if port == "" {
			port = "80"
		}
	} else {
		log.Panic("can't parse host or port")
	}
	return
}

func read(conn net.Conn) []byte {
	buffer := [1024]byte{}
	i := 0
	for {
		n, err := conn.Read(buffer[i:])
		if err != nil {
			log.Panic(err)
		}
		i += n
		if n == 0 || bytes.Count(buffer[:], []byte("\n")) >= 2 {
			break
		}
	}
	return buffer[:i]
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	partialData := read(conn)
	host, port := parseHostPort(partialData)
	log.Printf("read %d bytes from client", len(partialData))
	log.Printf("client request %s:%s", string(host), string(port))
	hostConn, err := net.Dial("tcp", host+":"+port)
	if err != nil {
		log.Panic()
	}
	if port == "443" {
		conn.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))
	} else {
		hostConn.Write(partialData)
	}
	go io.Copy(hostConn, conn)
	io.Copy(conn, hostConn)
}
