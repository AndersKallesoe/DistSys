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
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

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

type Client struct {
	ledger *Ledger
	peers  Peers
}

func makeClient() *Client {
	client := new(Client)
	client.ledger = MakeLedger()
	client.peers = Peers{m: make(map[string]net.Conn)}
	return client
}

// Keeps a list of all Peers in the network
type Peers struct {
	m     map[string]net.Conn
	mutex sync.Mutex
}

// Add peer to the network
func (peers *Peers) Set(key string, val net.Conn) {
	peers.m[key] = val
}

//ask for ip and read from terminal
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
	c.ConnectToPeer(IPAndPort)
}

func (c *Client) ConnectToPeer(IPAndPort string) {
	peer, err := net.Dial("tcp", IPAndPort)
	if peer == nil {
		fmt.Println("Starting new network")
	} else if err != nil {
		return
	} else {
		fmt.Println("connecting to network")
	}
	PrintHostNames()
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
	// Initialize the client
	client := makeClient()
	// Request IP and Port to connect to
	client.ConnectToNetwork()

}
