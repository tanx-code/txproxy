package main

import (
	"bytes"
	"io"
	"net"
	"os"
	"regexp"

	logging "github.com/op/go-logging"
	"github.com/urfave/cli"
)

var log = logging.MustGetLogger("txproxy")
var format = logging.MustStringFormatter(
	`%{color}%{time:15:04:05.000} %{level} %{shortfunc} %{shortfile}:%{color:reset} %{message}`,
)

func main() {
	// logger config
	backend1 := logging.NewLogBackend(os.Stdout, "", 0)
	backend2 := logging.NewLogBackend(os.Stderr, "", 0)
	backend2Formatter := logging.NewBackendFormatter(backend2, format)
	backend1Leveled := logging.AddModuleLevel(backend1)
	backend1Leveled.SetLevel(logging.ERROR, "")
	logging.SetBackend(backend1Leveled, backend2Formatter)
	app := cli.NewApp()
	app.Name = "txproxy"
	app.Usage = "a cool proxy"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "port, p",
			Value: "8080",
			Usage: "port to listen",
		},
	}
	app.Action = func(c *cli.Context) error {
		port := c.String("port")
		l, err := net.Listen("tcp", ":"+port)
		if err != nil {
			log.Critical(err)
		}
		log.Infof("txproxy listen on %s", port)
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Critical(err)
			}

			go handleConn(conn)
		}
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Critical(err)
	}
}

func parseHostPort(b []byte) (host string, port string) {
	log.Debug(string(b))
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
	log.Debugf("read %d bytes from client", len(partialData))
	log.Debugf("client request %s:%s", string(host), string(port))
	hostConn, err := net.Dial("tcp", host+":"+port)
	if err != nil {
		log.Panic()
	}
	if port == "443" {
		_, err := conn.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))
		if err != nil {
			log.Panic(err)
		}
	} else {
		_, err := hostConn.Write(partialData)
		if err != nil {
			log.Panic(err)
		}
	}
	go io.Copy(hostConn, conn)
	io.Copy(conn, hostConn)
}
