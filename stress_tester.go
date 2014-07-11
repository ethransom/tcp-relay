package main

import (
	"flag"
	"log"
	"net"
	"time"
)

var clients, packets int
var host, port string

func client(id int, ch chan bool) {
	log.Printf("Spawning client #%d", id)

	conn, err := net.Dial("tcp", host+":"+port)
	if err != nil {
		log.Fatal("error connecting to relay:", err)
	}
	defer conn.Close()

	for i := 0; i < packets; i++ {
		_, err = conn.Write([]byte("HEAD"))

		time.Sleep(time.Microsecond)
	}

	log.Println("Client", id, "shutting down")
	ch <- true
}

func main() {
	flag.IntVar(&clients, "clients", 5, "Number of clients to spawn")
	flag.IntVar(&packets, "packets", 100, "Number of packets to send per client")

	flag.Parse()
	host = flag.Arg(0)
	port = flag.Arg(1)
	if host == "" || port == "" {
		panic("usage: ./stress_tester [host] [port]")
	}

	log.Println("connecting to", host+":"+port)
	log.Println("using", clients, "clients and", packets, "per client")

	ch := make(chan bool)

	for i := 0; i < clients; i++ {
		go client(i, ch)

		time.Sleep(time.Second)
	}

	// wait for all clients to close
	for i := 0; i < clients; i++ {
		<-ch
	}
}
