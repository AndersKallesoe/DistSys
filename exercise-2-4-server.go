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

type Node struct {
	MessagesSent MessagesSentStruct
}

func mkNode() *Node {
	n := new(Node)
	n.MessagesSent = MessagesSentStruct{messageMap: make(map[string]bool)}
	return n
}

type MessagesSentStruct struct {
	messageMap map[string]bool
	mutex      sync.Mutex
}
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

func (n *Node) HandleConnection(conn net.Conn, outputs chan string, conns *Connections) {
	defer conn.Close()
	otherEnd := conn.RemoteAddr().String()
	conns.Set(otherEnd, conn)
	n.SendMessages(outputs)
	for {
		msg, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			fmt.Println("Ending session with " + otherEnd)
			delete(conns.m, otherEnd)
			return
		} else {
			//handle strings
			n.MessagesSent.mutex.Lock()
			if !n.MessagesSent.messageMap[string(msg)] {
				n.MessagesSent.messageMap[string(msg)] = true
				fmt.Print(string(msg))
				fmt.Print("> ")
				msgString := fmt.Sprintf(string(msg))
				outputs <- msgString
			}
			n.MessagesSent.mutex.Unlock()
		}
	}
}

func (n *Node) Broadcast(c chan string, conns *Connections) {
	for {
		msg := <-c
		for k := range conns.m {
			conns.m[k].Write([]byte(msg))
		}
	}
}

func (n *Node) Send(conn net.Conn, outputs chan string, reader *bufio.Reader) {
	for {
		fmt.Print("> ")
		text, _ := reader.ReadString('\n')
		txt := strings.TrimSpace(text)
		if txt == "quit" {
			return
		} else if txt == "printMap" {
			n.PrintMap()
		} else {
			n.MessagesSent.mutex.Lock()
			n.MessagesSent.messageMap[text] = true
			outputs <- text
			n.MessagesSent.mutex.Unlock()
		}
	}
}

func (n *Node) Listen(conn net.Conn, outputs chan string, conns *Connections) {
	ln, _ := net.Listen("tcp", ":0")

	defer ln.Close()
	n.PrintHostNames()
	fmt.Println("Listening for connections on port " + strings.TrimPrefix(ln.Addr().String(), "[::]"))
	fmt.Print("> ")
	for {
		conn, _ := ln.Accept()
		fmt.Println("Got a connection...")
		fmt.Print("> ")
		go n.HandleConnection(conn, outputs, conns)
	}
}

func (n *Node) SendMessages(outputs chan string) {
	n.MessagesSent.mutex.Lock()
	for key, _ := range n.MessagesSent.messageMap {
		outputs <- key
	}
	n.MessagesSent.mutex.Unlock()
}

func (n *Node) PrintHostNames() {
	// _ is convention for throwing the return value away
	name, _ := os.Hostname()
	addrs, _ := net.LookupHost(name)
	fmt.Println("Name: " + name)

	for indx, addr := range addrs {
		fmt.Println("Address number " + strconv.Itoa(indx) + ": " + addr)
	}
}

func (n *Node) PrintMap() {
	n.MessagesSent.mutex.Lock()
	defer n.MessagesSent.mutex.Unlock()
	for key, _ := range n.MessagesSent.messageMap {
		fmt.Println(key)
	}

}

//ask for ip and read from terminal
func (n *Node) GetIPandPort() string {
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
	n := mkNode()
	//print own ip and port
	ipAndPort := n.GetIPandPort()

	// create channel and list of connections

	reader := bufio.NewReader(os.Stdin)
	conns := MakeConns()
	outbound := make(chan string)

	//attempt to connect to ip

	conn, err := net.Dial("tcp", strings.TrimSpace(ipAndPort))
	if conn == nil {
		fmt.Println("Starting new network")
	} else if err != nil {
		return
	} else {
		fmt.Println("connecting to network")
		go n.HandleConnection(conn, outbound, conns)
	}

	go n.Broadcast(outbound, conns)

	//Listen for connections
	go n.Listen(conn, outbound, conns)
	go n.Send(conn, outbound, reader)
	for {
	}

}
