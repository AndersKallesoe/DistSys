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
	ledger    *Ledger
	peers     []string
	conns     Conns
	IPandPort string
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
		c.peers = append(c.peers, c.IPandPort)
	} else if err != nil {
		return
	} else {
		fmt.Println("connecting to network, requesting list of peers")

		enc := gob.NewEncoder(conn)
		request := "Requesting Peers"
		err := enc.Encode(&request)
		if err != nil {
			fmt.Println("Encode error request:", err)
		}

		dec := gob.NewDecoder(conn)
		peers := []string{}
		err = dec.Decode(&peers)
		if err != nil {
			fmt.Println("Decode error in list of peers:", err)
		}
		c.peers = append(c.peers, peers...)
		c.peers = append(c.peers, c.IPandPort)
		fmt.Println("peers are :", c.peers)
		conn.Close()
		c.ConnectToPeers()
	}

}

func (c *Client) ConnectToPeers() {
	//determine which peers to connect to
	peers := c.peers
	if len(peers) < 12 {
		peers = peers[:len(peers)-1]
	} else {
		peers = peers[len(peers)-11 : len(peers)-1]
	}
	for p := range peers {
		conn, err := net.Dial("tcp", peers[p])
		if conn == nil {
			fmt.Println("There was an error in connecting to: ", peers[p])
		} else if err != nil {
			return
		} else {
			enc := gob.NewEncoder(conn)
			request := "Connection"
			err := enc.Encode(&request)
			if err != nil {
				fmt.Println("Encode error request:", err)
			}
			c.conns.Set(conn.RemoteAddr().String(), conn)

		}
	}
}

func (c *Client) StartListen() net.Listener {
	ln, _ := net.Listen("tcp", ":0")
	IP := getIP()
	Port := strings.TrimPrefix(ln.Addr().String(), "[::]:")
	c.IPandPort = IP + ":" + Port
	fmt.Println("Listening for connections on: <" + c.IPandPort + ">")
	return ln
}

func (c *Client) Listen(ln net.Listener) {
	defer ln.Close()
	for {
		conn, _ := ln.Accept()
		fmt.Println("Got a connection, awaiting request...")
		request := ""
		dec := gob.NewDecoder(conn)
		err := dec.Decode(&request)
		if err != nil {
			fmt.Println("Decode error in msg:", err)
		}
		fmt.Println(request)
		if request == "Requesting Peers" {
			peers := c.peers
			enc := gob.NewEncoder(conn)
			err = enc.Encode(&peers)
			if err != nil {
				fmt.Println("Encode error in list of peers:", err)
			}
			conn.Close()
		} else if request == "Connection" {
			c.conns.Set(conn.RemoteAddr().String(), conn)
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
	ln := client.StartListen()
	client.ConnectToNetwork()
	go client.Listen(ln)

	for {
	}
}
