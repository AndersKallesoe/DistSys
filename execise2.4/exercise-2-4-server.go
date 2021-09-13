package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
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

type MessagesSentStruct struct {
	messageMap map[string]bool
	mutex      sync.Mutex
}
type Connections struct {
	m map[string]net.Conn
}

var MessagesSent MessagesSentStruct

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
			MessagesSent.mutex.Lock()
			if !MessagesSent.messageMap[string(msg)] {
				MessagesSent.messageMap[string(msg)] = true
				fmt.Print(string(msg))
				fmt.Print("> ")
				msgString := fmt.Sprintf(string(msg))
				outputs <- msgString
			}
			MessagesSent.mutex.Unlock()
		}
	}
}

func Broadcast(c chan string, conns *Connections) {
	for {
		msg := <-c
		for k := range conns.m {
			conns.m[k].Write([]byte(msg))
		}
	}
}

func Send(conn net.Conn, outputs chan string, reader *bufio.Reader) {
	for {
		fmt.Print("> ")
		text, _ := reader.ReadString('\n')
		txt := strings.TrimSpace(text)
		if txt == "quit" {
			return
		} else if txt == "printMap" {
			PrintMap()
		} else {
			MessagesSent.mutex.Lock()
			MessagesSent.messageMap[text] = true
			outputs <- text
			MessagesSent.mutex.Unlock()
		}
	}
}

func Listen(conn net.Conn, outputs chan string, conns *Connections) {
	ln, _ := net.Listen("tcp", ":0")

	defer ln.Close()
	PrintHostNames()
	fmt.Println("Listening for connections on port " + strings.TrimPrefix(ln.Addr().String(), "[::]"))
	fmt.Print("> ")
	for {
		conn, _ := ln.Accept()
		fmt.Println("Got a connection...")
		fmt.Print("> ")
		go HandleConnection(conn, outputs, conns)
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

func PrintMap() {
	MessagesSent.mutex.Lock()
	defer MessagesSent.mutex.Unlock()
	for key, _ := range MessagesSent.messageMap {
		fmt.Println(key)
	}

}

//ask for ip and read from terminal
func GetIPandPort() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Please provide IP address and port number in the format <ip>:<port>")
	fmt.Print("> ")
	ipAndPort, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("no server with <" + ipAndPort + ">")
		return ""
	}
	return ipAndPort
}

func main() {
	//print own ip and port
	ipAndPort := GetIPandPort()

	// create channel and list of connections
	reader := bufio.NewReader(os.Stdin)
	conns := MakeConns()
	outbound := make(chan string)
	MessagesSent = MessagesSentStruct{messageMap: make(map[string]bool)}

	//attempt to connect to ip

	conn, err := net.Dial("tcp", strings.TrimSpace(ipAndPort))
	if conn == nil {
		fmt.Println("Starting new network")
	} else if err != nil {
		return
	} else {
		fmt.Println("connecting to network")
		go HandleConnection(conn, outbound, conns)
	}

	go Broadcast(outbound, conns)

	//Listen for connections
	go Listen(conn, outbound, conns)
	go Send(conn, outbound, reader)
	for {
	}

}
