package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/inconshreveable/muxado"
)

var info, warn *log.Logger

func main() {
	var max_services int
	flag.IntVar(&max_services, "max", 50, "maximum number of services to forward on behalf of")
	flag.Parse()

	port, err := strconv.Atoi(flag.Arg(0))
	if flag.NArg() != 1 || err != nil {
		fmt.Println("operand missing: port to listen on")
		os.Exit(2)
	}

	info = log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds)
	warn = log.New(os.Stdout, "WARN: ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	// open up the back-facing port for services to connect
	socket, err := muxado.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		fmt.Println("couldn't create muxado socket: ", err.Error())
		return
	}
	defer socket.Close()
	info.Println("listening on port", port)

	// create our socket pool
	info.Println("serving for", max_services, "services")
	socket_pool := make(chan int, max_services)
	for i := 1; i <= max_services; i++ {
		socket_pool <- i + port
	}

	// handle new connections from services
	for {
		req, err := socket.Accept()
		if err != nil {
			warn.Println("error accepting service:", err.Error())
			break
		}

		go handleSession(req, socket_pool)
	}

	log.Panic("aborting...")
}

func handleSession(back_conn muxado.Session, socket_pool chan int) {
	defer back_conn.Close()

	// begin handshaking
	// the first (and only) stream opened by the client is the handshaking stream
	stream, err := back_conn.Accept()
	if err != nil {
		warn.Println("can't accept handshake stream:", err)
		return
	}

	// pull a socket from the pool
	// the connection will stall here if the server is full
	port := <-socket_pool
	address := "localhost" + ":" + strconv.Itoa(port)

	// setup a new forward-facing port for clients to connect to the connecting server
	front_conn, err := net.Listen("tcp", address)
	if err != nil {
		warn.Println("can't open socket:", err.Error())
		// don't put the socket back in the pool, maybe it's taken
		return
	}
	defer (func() {
		front_conn.Close()
		info.Println("closing port:", port)
		socket_pool <- port
	})()

	// send the service's forwarded address to the service
	byteArray := []byte(address)
	stream.Write(byteArray)
	stream.Close()

	info.Println("forwarding a service on:", address)

	// accept clients for the server
	for {
		client, err := front_conn.Accept()
		if err != nil {
			warn.Println("can't accept from:", address, ": ", err.Error())
			continue
		}
		defer client.Close()

		info.Println("accepted client for:", address)

		// "finish" the connection with a muxado stream
		server, err := back_conn.Open()
		if err != nil {
			warn.Println("can't open multiplexed stream:", err)
			break
		}

		// whichever goroutine reads from a stream is expected to close it
		// this could probably be improved in the future

		// forward server data to client
		go (func(server muxado.Stream, client net.Conn) {
			defer server.Close()
			io.Copy(server, client)
		})(server, client)

		// forward client data to server
		go (func(server muxado.Stream, client net.Conn) {
			defer client.Close()
			io.Copy(client, server)
		})(server, client)
	}

	warn.Println("aborting service:", address)
}
