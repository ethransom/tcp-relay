package main

import (
	"flag"
	"fmt"

	"github.com/inconshreveable/muxado"
	// "net"
	// "os"
	"io/ioutil"
)

func main() {
	flag.Parse()
	host := flag.Arg(0)
	port := flag.Arg(1)
	if host == "" || port == "" {
		panic("usage: ./echoserver [relay host] [relay port]")
	}

	// connect to the relay
	sess, err := muxado.Dial("tcp", host+":"+port)
	if err != nil {
		panic(err)
	}
	defer (func() {
		sess.Close()
		fmt.Println("Disconnected...")
	})()

	fmt.Println("Connected. Waiting for handshake...")

	// before we can run smoothly, we must handshake with the server
	// the server will send our new address down the first session
	// that the client opens
	stream, err := sess.Open()
	if err != nil {
		panic(err)
	}
	buf, err := ioutil.ReadAll(stream) // the server sends the address and nothing else
	if err != nil {
		panic(err)
	}
	fmt.Println("Assigned an address:", string(buf))
	stream.Close()

	fmt.Println("Handshake complete\nBeginning normal operation...")

	// use this goroutine to wait for and process clients
	for {
		stream, err := sess.Accept()
		if err != nil {
			fmt.Println("Couldn't accept client:", err)
			continue
		}
		fmt.Println("Accepted client")

		go handleStream(stream)
	}
}

// annoyingly echoes to a client
func handleStream(stream muxado.Stream) {
	defer (func() {
		stream.Close()
		fmt.Println("Closed connection to client")
	})()

	for {
		buf := make([]byte, 256)
		_, err := stream.Read(buf)
		if err != nil {
			fmt.Println("Error reading:", err.Error())
			return
		}
		stream.Write(buf)
	}
}
