package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

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

func PrintHostNames() {
	// _ is convention for throwing the return value away
	name, _ := os.Hostname()
	addrs, _ := net.LookupHost(name)
	fmt.Println("Name: " + name)

	for indx, addr := range addrs {
		fmt.Println("Address number " + strconv.Itoa(indx) + ": " + addr)
	}
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
		}
	}

}

func main() {
	PrintHostNames()
	conns := MakeConns()
	outbound := make(chan string)
	go Broadcast(outbound, conns)
	ln, _ := net.Listen("tcp", ":18081")
	defer ln.Close()
	for {
		fmt.Println("Listening for connections on port 18081...")

		conn, _ := ln.Accept()
		fmt.Println("Got a connection...")
		go HandleConnection(conn, outbound, conns)
	}
}
