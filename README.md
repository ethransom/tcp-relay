tcp-relay
=========

A simple tcp relay written in Go. The [muxado](https://github.com/inconshreveable/muxado)
library is used to multiplex TCP connections. Services wishing to utilize the relay must
use a muxado session to connect to the relay.

 * `relay.go` implements the relay
 * `echoserver.go` implements a simple echo server that utilizes the relay

Usage
-----

```sh
$ ./relay 8080 & # the relay will listen for services on 8080
$ ./echoserver localhost 8080 & # connect to the relay at localhost:8080
localhost:8081 
# the relay opened 8081 for programs wishing to reach the echoserver
$ telnet localhost 8081
Hello, world
Hello, world
```

`echoserver.go` is a clean, well commented example of how Go programs can connect
and handshake with the relay.

Architectural Overview
-----------

```
                  ---------------                                               
                  | HTTP Client |                                               
                  ---------------                                               
                           \                                                    
  Generic TCP conn. ----->  \                                                   
                   \         \                                                  
                    \         \                                                 
                     v         ----------------                                 
 ---------------               |              |                 --------------- 
 | Echo Client |---------------|  tcp-relay   |=================| HTTP Server | 
 ---------------               |              |      ^          --------------- 
                               ----------------       \                         
                              /                \\      Multiplexed Connections  
                             /                  \\    /                         
                            /                    \\  v                          
                           /                      \\                            
                   ---------------                 \\                           
                   | HTTP Client |                  \\                          
                   ---------------              ---------------                 
                                                | Echo Server |                 
                                                ---------------                 
```

n00bish design decisions
------------------------

 * Services wishing to register with the relay are assigned a forward-facing port.
   If the relay is handling too many services, the service is put on hold until a 
   port opens up. This could take a very long time. 
 * The closing of connections isn't handled very gracefully. It was either that or spaghetti code.
   This means the relay probably isn't very robust, because resources aren't being freed properly.
 * Many programs can't use muxado. It would be nice to have a proxy adaptor/server.



