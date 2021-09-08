package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

// Declare variable of type net.Conn, called conn.
var conn net.Conn

func send(conn net.Conn, reader *bufio.Reader) {
	for {
		fmt.Print("> ")
		text, _ := reader.ReadString('\n')
		if text == "quit\n" { return }
		fmt.Fprintf(conn, text)
	}
}

func receive(conn net.Conn) {
	for {
		msg, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil { return }
		fmt.Println("From server: " + msg)
		fmt.Print("> ")
	}
}


func main() {

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Please provide IP address and port number in the format <ip>:<port>")
	fmt.Print("> ")
	ipAndPort, err := reader.ReadString('\n')

	fmt.Println("Debug: " + ipAndPort)
	if err != nil { return }
	
	// Attempt to connect to provided ipAndPort.
	conn, _ = net.Dial("tcp", strings.TrimSpace(ipAndPort))
	defer conn.Close()

	go send(conn, reader)
	go receive(conn)

	for {}

}
