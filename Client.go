package main

/*
1. Keep a list of peers in the order in which their joined the network, with the latest peer to arrive being at the end.
2. When connecting to a peer, ask for its list of peers.
3. Then add yourself to the end of your own list.
4. Then connect to the ten peers before you on the list. If the list has length less than 11 then just connect to all peers but yourself.
5. Then broadcast your own presence.
6. When a new presence is broadcast, add it to the end of your list of peers.
7. When a transaction is made, broadcast the Transaction object.
8. When a transaction is received, update the local Ledger object
*/
import (
	"bufio"
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

/*************/
type Ledger struct {
	Accounts map[string]int
	lock     sync.Mutex
}

func MakeLedger() *Ledger {
	ledger := new(Ledger)
	ledger.Accounts = make(map[string]int)
	return ledger
}

type Transaction struct {
	ID     string
	From   string
	To     string
	Amount int
}

func (l *Ledger) Transaction(t *Transaction) {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.Accounts[t.From] -= t.Amount
	l.Accounts[t.To] += t.Amount
}

/*********************/

type Client struct {
	ledger *Ledger
	peers  []string
	conns  Conns
}

func makeClient() *Client {
	client := new(Client)
	client.ledger = MakeLedger()
	client.peers = []string{}
	client.conns = Conns{m: make(map[string]net.Conn)}
	return client
}

// Keeps a list of all Peers in the network
type Conns struct {
	m     map[string]net.Conn
	mutex sync.Mutex
}

// Add connection to the network
func (conns *Conns) Set(key string, val net.Conn) {
	conns.m[key] = val
}

// Ask for <IP:Port>, read from terminal, and return it
func (c *Client) GetIPandPort() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Please provide IP address and port number in the format <ip>:<port>")
	fmt.Print("> ")
	IPAndPort, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("formatting error: <" + IPAndPort + "> not an IP and port")
		return ""
	}
	IPAndPort = strings.TrimSpace(IPAndPort)
	return IPAndPort
}

func (c *Client) ConnectToNetwork() {
	IPAndPort := c.GetIPandPort()
	conn, err := net.Dial("tcp", IPAndPort)
	if conn == nil {
		fmt.Println("Starting new network")
	} else if err != nil {
		return
	} else {
		fmt.Println("connecting to network")
		dec := gob.NewDecoder(conn)
		peers := []string{}
		err := dec.Decode(&peers)
		if err != nil {
			fmt.Println("Decode error in list of peers:", err)
		}
		c.peers = append(c.peers, peers...)

	}

}

func (c *Client) ConnectToPeers() {

}

func (c *Client) Listen() {
	ln, _ := net.Listen("tcp", ":0")
	defer ln.Close()
	IP := getIP()
	Port := strings.TrimPrefix(ln.Addr().String(), "[::]:")
	IPandPort := IP + ":" + Port
	fmt.Println("Listening for connections on: <" + IPandPort + ">")
	c.peers = append(c.peers, IPandPort)
	fmt.Println("peers are :", c.peers)

	for {
		conn, _ := ln.Accept()
		fmt.Println("Got a connection, sending peers...")
		peers := c.peers
		enc := gob.NewEncoder(conn)
		err := enc.Encode(peers)
		if err != nil {
			fmt.Println("Encode error in list of peers:", err)
		}

	}
}

func getIP() string {
	// _ is convention for throwing the return value away
	name, _ := os.Hostname()
	addrs, _ := net.LookupHost(name)
	IP := addrs[1]
	fmt.Println("IP : " + IP)
	return IP
}

func main() {
	// Initialize the client
	client := makeClient()
	// Request IP and Port to connect to
	client.ConnectToNetwork()

	go client.Listen()

	for {
	}
}
