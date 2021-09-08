package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

/*
1. It runs as a command line program.
2. When it starts up it asks for the IP address and port number of an existing
peer on the network. If the IP address or port is invalid or no peer is found at
the address, the client starts its own new network with only itself as member.
3. Then the client prints its own IP address and the port on which it waits for
connections.
4. Then it will iteratively prompt the user for text strings.
5. When the user types a text string at any connected client, then it will eventually
be printed at all other clients.
6. Only the text string should be printed, no information about who sent it.
The system should be implemented as follows:
	1. When a client connects to an existing peer, it will keep a TCP connection to
	that peer.
	2. Then the client opens its own port where it waits for incoming TCP connec-
	tions.
	3. All the connections will be treated the same, they will be used for both sending
	and receiving strings.
	4. It keeps a set of messages that it already sent. In Go you can make a set as
	a map var MessagesSent map[string]bool. You just map the strings that
	were sent to true. Initially all of them are set to false, so the set is initially
	empty, as it should be.
	5. When a string is typed by the user or a string arrives on any of its connections,
	the client checks if it is already sent. If so, it does nothing. Otherwise it adds
	it to MessagesSent and then sends it on all its connections. (Remember con-
	currency control. Probably several go-routines will access the set at the same
	time. Make sure that does not give problems.)
	6. Whenever a message is added to MessagesSent, also print it for the user to
	see.
	7. Optional: Try to ensure that if clients arrive on the network after it already
	started running, then they also receive the messages sent before they joined
	the network. This is not needed for full grades.
*/

var MessagesSent map[string]bool
var outbound chan string
var conns *Connections

type Connections struct {
	m map[string]net.Conn
}

func (conns *Connections) Set(key string, val net.Conn) {
	conns.m[key] = val
}

func MakeConns() *Connections {
	conns := new(Connections)
	conns.m = make(map[string]net.Conn)
	return conns
}

func HandleConnection(conn net.Conn, outputs chan string, conns *Connections) {
	defer conn.Close()

	otherEnd := conn.RemoteAddr().String()

	conns.Set(otherEnd, conn)

	for {
		msg, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			fmt.Println("Ending session with " + otherEnd)
			delete(conns.m, otherEnd)
			return
		} else {
			//handle strings
			_, ok := MessagesSent[string(msg)]
			if !ok {
				MessagesSent[string(msg)] = false
			}
			if MessagesSent[string(msg)] {
				MessagesSent[string(msg)] = true
				fmt.Println(string(msg))

			}

			fmt.Print("From "+otherEnd+": ", string(msg))
			msgString := fmt.Sprintf("%s : %s", otherEnd, string(msg))
			outputs <- msgString
		}
	}
}

func Broadcast(c chan string, conns *Connections) {
	for {
		msg := <-c
		titlemsg := strings.Title(msg)
		for k := range conns.m {
			conns.m[k].Write([]byte(titlemsg))
			fmt.Printf("Sending %s to %s\n", strings.TrimSpace(msg), k)
		}
	}

}

func send(conn net.Conn, outputs chan string, reader *bufio.Reader) {
	for {
		fmt.Print("> ")
		text, _ := reader.ReadString('\n')
		if text == "quit\n" {
			return
		}
		MessagesSent[text] = true
		outputs <- text
	}
}

func Listen() {
	ln, _ := net.Listen("tcp", ":18081")
	defer ln.Close()
	for {
		fmt.Println("Listening for connections on port 18081...")

		conn, _ := ln.Accept()
		fmt.Println("Got a connection...")
		go HandleConnection(conn, outbound, conns)
	}
}

func PrintHostNames() {
	// _ is convention for throwing the return value away
	name, _ := os.Hostname()
	addrs, _ := net.LookupHost(name)
	fmt.Println("Name: " + name)

	for indx, addr := range addrs {
		fmt.Println("Address number " + strconv.Itoa(indx) + ": " + addr)
	}
}

func main() {
	//ask for ip and read from terminal
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Please provide IP address and port number in the format <ip>:<port>")
	fmt.Print("> ")
	ipAndPort, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	// create channel and list of connections
	outbound = make(chan string)
	conns = MakeConns()
	MessagesSent = make(map[string]bool)

	//attempt to connect to ip
	conn, _ := net.Dial("tcp", strings.TrimSpace(ipAndPort))
	go HandleConnection(conn, outbound, conns)
	go Broadcast(outbound, conns)

	//print own ip and port
	PrintHostNames()
	//Listen for connections
	go Listen()
	go send(conn, outbound, reader)

}
