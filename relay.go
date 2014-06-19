package main

import (
	"github.com/inconshreveable/muxado"
	"flag"
	"fmt"
	"net"
	"io"
	"os"
	"strconv"
)

func main() {
	var max_services int
	flag.IntVar(&max_services, "max", 50, "maximum number of services to forward on behalf of")
	flag.Parse()

	port, err := strconv.Atoi(flag.Arg(0))
    if flag.NArg() != 1 || err != nil {
        fmt.Println("operand missing: port to listen on")
        os.Exit(2)
    }

	// open up the back-facing port for services to connect
	socket, err := muxado.Listen("tcp", ":" + strconv.Itoa(port))
	if err != nil {
		fmt.Println("Error creating socket: ", err.Error())
		return
	}
	defer socket.Close()
	fmt.Println("Listening on port", port)

	// create our socket pool
	fmt.Println("Serving for", max_services, "services")
	socket_pool := make(chan int, max_services)
	for i := 1; i <= max_services; i++ {
		socket_pool <- i + port
	}

	// handle new connections from services
	for {
		req, err := socket.Accept()
		if err != nil {
			fmt.Println("Error accepting server: ", err.Error())
			continue
		}

		go handleSession(req, socket_pool)
	}
}

func handleSession(back_conn muxado.Session, socket_pool chan int) {
	defer back_conn.Close()

	// begin handshaking
	// the first (and only) stream opened by the client is the handshaking stream
	stream, err := back_conn.Accept()
	if err != nil {
		fmt.Println(err)
		return
	}

	// pull a socket from the pool
	// the connection will stall here if the server is full
	port := <- socket_pool
	address := "localhost"+":"+strconv.Itoa(port)

	// setup a new forward-facing port for clients to connect to the connecting server
	front_conn, err := net.Listen("tcp", address)
	if err != nil {
	    fmt.Println("Error setting up forward-facing port:", err.Error())
	    // don't put the socket back in the pool, maybe it's taken
	    return
	}
	defer (func () {
		front_conn.Close()
		fmt.Println("No longer forwarding on", port)
		socket_pool <- port
	})()

	// send the service's forwarded address to the service
	byteArray := []byte(address)
	stream.Write(byteArray)
	stream.Close()

	fmt.Println("Now forwarding", address)

	// accept clients for the server
	for {
	    client, err := front_conn.Accept()
	    if err != nil {
	        fmt.Println("Error accepting from ", address, ": ", err.Error())
	        continue
	    }

	    fmt.Println("Accepted client for ", address)

	    // "finish" the connection with a muxado stream
	    server, err := back_conn.Open()
    	if err != nil {
    		fmt.Println("Error opening multiplexed stream:", err)
			continue
		}

		// whichever goroutine reads from a stream is expected to close it
		// this could probably be improved in the future

	    // forward server data to client
	    go (func (server muxado.Stream, client net.Conn) {
			defer server.Close()
			io.Copy(server, client)
			fmt.Println("no longer server->client")
		})(server, client)

	    // forward client data to server
	    go (func (server muxado.Stream, client net.Conn) {
	    	defer client.Close()
	    	io.Copy(client, server)
			fmt.Println("no longer client->server")
		})(server, client)
	}
}

