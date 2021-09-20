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
	"reflect"
	"strconv"
	"strings"
	"sync"
)

/*************/
type Ledger struct {
	Accounts map[string]int
	lock     sync.Mutex
}

type Transaction struct {
	ID     string
	From   string
	To     string
	Amount int
}


type Message struct {
	Msgtype     string
	Transaction Transaction
	IPandPort   string
	Peers       []string
}

type Client struct {
	ledger       *Ledger
	peers        []string
	conns        Conns
	IPandPort    string
	index        int
	transactions []string
	
}

// Keeps a list of all Peers in the network
type Conns struct {
	m     map[string]net.Conn
	mutex sync.Mutex
}

func MakeLedger() *Ledger {
	ledger := new(Ledger)
	ledger.Accounts = make(map[string]int)
	return ledger
}

func (c *Client) printLedger() {
	for k, v := range c.ledger.Accounts {
		fmt.Println("Account: " + k + " Balance: " + strconv.Itoa(v))
	}
}

func (l *Ledger) Transaction(t *Transaction) {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.Accounts[t.From] -= t.Amount
	l.Accounts[t.To] += t.Amount
}

func (c *Client) getID() string {
	return c.IPandPort + ":" + strconv.Itoa(c.index) + ":" + strconv.Itoa(len(c.transactions) + 1)
}

func makeClient() *Client {
	client := new(Client)
	client.ledger = MakeLedger()
	client.peers = []string{}
	client.conns = Conns{m: make(map[string]net.Conn)}
	client.index = 0
	client.transactions = []string{}
	return client
}

// Add connection to the network
func (conns *Conns) Set(key string, val net.Conn) {
	conns.m[key] = val
}

func (c *Client) PeerExists(peer string) bool {
	for p := range c.peers {
		if c.peers[p] == peer {
			return true
		}
	}
	return false
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
		request := Message{Msgtype: "Requesting Peers"}
		fmt.Println(request)
		err := enc.Encode(&request)
		if err != nil {
			fmt.Println("Encode error request:", err)
		}

		dec := gob.NewDecoder(conn)
		msg := Message{}
		err = dec.Decode(&msg)
		if err != nil {
			fmt.Println("Decode error in list of peers:", err)
		}
		c.peers = append(c.peers, msg.Peers...)
		c.peers = append(c.peers, c.IPandPort)
		fmt.Println("peers are :", c.peers)
		conn.Close()
		c.ConnectToPeers()
	}

}


func (c *Client) ConnectToPeers() {
	//determine which peers to connect to
	peers := c.peers
	if len(peers) <= 11 {
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
			request := Message{Msgtype: "Connection"}
			err := enc.Encode(request)
			if err != nil {
				fmt.Println("Encode error request:", err)
			}
			c.conns.mutex.Lock()
			c.conns.Set(conn.RemoteAddr().String(), conn)
			c.conns.mutex.Unlock()
		}
	}
	c.Broadcastpresence(c.IPandPort)
}

func (c *Client) Broadcastpresence(IPAndPort string) {

	for k := range c.conns.m {
		enc := gob.NewEncoder(c.conns.m[k])
		request := Message{Msgtype: "Broadcast Presence", IPandPort: IPAndPort}
		err := enc.Encode(request)
		if err != nil {
			fmt.Println("Encode error request:", err)
		}
	}
}

func (c *Client) BroadcastTransaction(t Transaction) {
	if c.TransactionExists(t.ID) {
		return
	}
	c.ledger.Transaction(&t)
	c.transactions = append(c.transactions, t.ID)
	for k := range c.conns.m {
		enc := gob.NewEncoder(c.conns.m[k])
		request := Message{Msgtype: "Broadcast Transaction", Transaction: t}
		err := enc.Encode(request)
		if err != nil {
			fmt.Println("Encode error request:", err)
		}
	}
}

func (c *Client) TransactionExists(transaction string) bool {
	for p := range c.transactions {
		if c.transactions[p] == transaction {
			return true
		}
	}
	return false

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
		msg := Message{}
		dec := gob.NewDecoder(conn)
		err := dec.Decode(&msg)
		if err != nil {
			fmt.Println("Decode error in msg:", err)
		}
		switch msg.Msgtype {
		case "Requesting Peers":
			peers := Message{Peers: c.peers}
			enc := gob.NewEncoder(conn)
			err = enc.Encode(&peers)
			if err != nil {
				fmt.Println("Encode error in list of peers:", err)
			}
		case "Connection":
			c.conns.mutex.Lock()
			c.conns.Set(conn.RemoteAddr().String(), conn)
			c.conns.mutex.Unlock()
			go c.HandleConnection(conn)
		default:
			fmt.Println("No match case found for: " + msg.Msgtype)
		}

	}
}

func getIP() string {
	// _ is convention for throwing the return value away
	name, _ := os.Hostname()
	addrs, _ := net.LookupHost(name)
	IP := addrs[len(addrs)-1]
	fmt.Println("IP : " + IP)
	return IP
}

func (c *Client) HandleConnection(conn net.Conn) {
	for {
		dec := gob.NewDecoder(conn)
		msg := Message{}
		err := dec.Decode(&msg)
		if err != nil {
			fmt.Println("Encode error in broadcasting presence to network:", err)
			conn.Close()
			return
		}
		switch msg.Msgtype {
			case "Broadcast Presence":
				if !c.PeerExists(msg.IPandPort) {
					c.peers = append(c.peers, msg.IPandPort)
					c.Broadcastpresence(msg.IPandPort)
				}
			case "Broadcast Transaction":
				transaction := msg.Transaction
				c.BroadcastTransaction(transaction)
			default:
				fmt.Println("No match case found for: " + msg.Msgtype)
		}
	}
}

func (c *Client) takeInput() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		text, _ := reader.ReadString('\n')
		txt := strings.TrimSpace(text)
		if txt == "quit" {
			return
		} else if txt == "printLedger" {
			c.printLedger()
		} else if txt == "printPeers" {
			fmt.Println(c.peers)
		} else if txt == "Transaction" {
			t := c.RequestTransactionInfo()
			c.BroadcastTransaction(t)
		}
	}
}

func (c *Client) RequestTransactionInfo() Transaction {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("From: ")
	fmt.Print("> ")
	from, err := reader.ReadString('\n')

	if err != nil {
		fmt.Println("formatting error: " + from)
		return Transaction{}
	}

	fmt.Println("To: ")
	fmt.Print("> ")
	to, err := reader.ReadString('\n')

	if err != nil {
		fmt.Println("formatting error: " + to)
		return Transaction{}
	}

	fmt.Println("Amount ")
	fmt.Print("> ")
	amount, err := reader.ReadString('\n')

	if err != nil {
		fmt.Println("formatting error: " + amount)
		return Transaction{}
	}
	amount = strings.TrimSpace(amount)
	fmt.Println(amount)
	amt, err := strconv.Atoi(amount)
	fmt.Println(amt, err, reflect.TypeOf(amt))
	if err != nil {
		fmt.Println("formatting error: amount not an integer")
		return Transaction{}
	}
	return Transaction{From: from, To: to, Amount: amt, ID: c.getID()}

}

func (c *Client) PrintHostNames() {
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
	client.PrintHostNames()
	// Request IP and Port to connect to
	ln := client.StartListen()
	client.ConnectToNetwork()
	for _, conn := range client.conns.m {
		go client.HandleConnection(conn)
	}
	go client.Listen(ln)
	go client.takeInput()
	for {}
}
