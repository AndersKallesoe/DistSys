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
	"math/big"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

/*************/
var KeyGen KeyGenerator

type Ledger struct {
	Accounts map[string]int
	lock     sync.Mutex
}

func MakeLedger() *Ledger {
	ledger := new(Ledger)
	ledger.Accounts = make(map[string]int)
	return ledger
}

func (C *Client) printLedger() {
	for k, v := range C.ledger.Accounts {
		fmt.Println("Account: " + k + " Balance: " + strconv.Itoa(v))
	}
}

type SignedTransaction struct {
	ID        string // Any string
	From      string // A verification key coded as a string
	To        string // A verification key coded as a string
	Amount    int    // Amount to transfer
	Signature string // Potential signature coded as string
}

type Message struct {
	Msgtype     string
	Transaction SignedTransaction
	IPandPort   string
	Peers       []string
	AccountName string
	KeySet      KeySet
}

type Client struct {
	ledger        *Ledger
	peers         []string
	conns         Conns
	reader        *bufio.Reader
	IPandPort     string
	index         int
	transactions  []string
	LocalAccounts map[string]Account
}

type KeySet struct {
	VerificationKey *big.Int
	SigningKey      *big.Int
	Modular         *big.Int
}

// Keeps a list of all Peers in the network
type Conns struct {
	m     map[string]net.Conn
	mutex sync.Mutex
}

func (C *Client) SignedTransaction(t SignedTransaction) {
	C.ledger.lock.Lock()
	defer C.ledger.lock.Unlock()
	s := t.ID + t.From + t.To + strconv.Itoa(t.Amount)
	v, m := SplitPublicKey(t.From)
	/*fmt.Print("string: ")
	fmt.Println(s)
	fmt.Print("signature: ")
	fmt.Println(t.Signature)
	fmt.Print("verification key: ")
	fmt.Println(v)
	fmt.Print("modular: ")
	fmt.Println(m)*/
	if verify(s, t.Signature, v, m) {
		fmt.Println("verification succesful")
		C.ledger.Accounts[t.From] -= t.Amount
		C.ledger.Accounts[t.To] += t.Amount
	} else {
		fmt.Println("verification unsuccesful")
	}

}

func (C *Client) getID() string {
	return C.IPandPort + ":" + strconv.Itoa(C.index) + ":" + strconv.Itoa(len(C.transactions)+1)
}

func makeClient() *Client {
	client := new(Client)
	client.ledger = MakeLedger()
	client.peers = []string{}
	client.conns = Conns{m: make(map[string]net.Conn)}
	client.index = 0
	client.transactions = []string{}
	client.LocalAccounts = make(map[string]Account)
	client.reader = bufio.NewReader(os.Stdin)
	return client
}

// Add connection to the network
func (conns *Conns) Set(key string, val net.Conn) {
	conns.m[key] = val
}

func (C *Client) PeerExists(peer string) bool {
	for p := range C.peers {
		if C.peers[p] == peer {
			return true
		}
	}
	return false
}

// Ask for <IP:Port>, read from terminal, and return it
func (C *Client) GetIPandPort() string {
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

func (C *Client) ConnectToNetwork() {
	IPAndPort := C.GetIPandPort()
	conn, err := net.Dial("tcp", IPAndPort)
	if conn == nil {
		fmt.Println("Starting new network")
		C.peers = append(C.peers, C.IPandPort)
	} else if err != nil {
		return
	} else {
		fmt.Println("connecting to network, requesting list of peers")

		enc := gob.NewEncoder(conn)
		request := Message{Msgtype: "Requesting Peers", Transaction: SignedTransaction{}}
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
		C.peers = append(C.peers, msg.Peers...)
		C.peers = append(C.peers, C.IPandPort)
		fmt.Println("peers are :", C.peers)
		conn.Close()
		C.ConnectToPeers()
	}

}

func (C *Client) ConnectToPeers() {
	//determine which peers to connect to
	peers := C.peers
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
			C.conns.mutex.Lock()
			C.conns.Set(conn.RemoteAddr().String(), conn)
			C.conns.mutex.Unlock()
		}
	}
	C.Broadcastpresence(C.IPandPort)
}

func (C *Client) Broadcastpresence(IPAndPort string) {
	for k := range C.conns.m {
		enc := gob.NewEncoder(C.conns.m[k])
		request := Message{Msgtype: "Broadcast Presence", IPandPort: IPAndPort}
		err := enc.Encode(request)
		if err != nil {
			fmt.Println("Encode error request:", err)
		}
	}
}

func (C *Client) BroadcastTransaction(t SignedTransaction) {
	if C.TransactionExists(t.ID) {
		return
	}
	C.SignedTransaction(t)
	C.transactions = append(C.transactions, t.ID)
	for k := range C.conns.m {
		enc := gob.NewEncoder(C.conns.m[k])
		request := Message{Msgtype: "Broadcast Transaction", Transaction: t}
		err := enc.Encode(request)
		if err != nil {
			fmt.Println("Encode error request:", err)
		}
	}
}

func (C *Client) TransactionExists(transaction string) bool {
	for p := range C.transactions {
		if C.transactions[p] == transaction {
			return true
		}
	}
	return false

}

func (C *Client) StartListen() net.Listener {
	ln, _ := net.Listen("tcp", ":0")
	IP := getIP()
	Port := strings.TrimPrefix(ln.Addr().String(), "[::]:")
	C.IPandPort = IP + ":" + Port
	fmt.Println("Listening for connections on: <" + C.IPandPort + ">")
	return ln
}

func (C *Client) Listen(ln net.Listener) {
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
			peers := Message{Peers: C.peers}
			enc := gob.NewEncoder(conn)
			err = enc.Encode(&peers)
			if err != nil {
				fmt.Println("Encode error in list of peers:", err)
			}
		case "Connection":
			C.conns.mutex.Lock()
			C.conns.Set(conn.RemoteAddr().String(), conn)
			C.conns.mutex.Unlock()
			go C.HandleConnection(conn)
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

func (C *Client) HandleConnection(conn net.Conn) {
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
			if !C.PeerExists(msg.IPandPort) {
				C.peers = append(C.peers, msg.IPandPort)
				C.Broadcastpresence(msg.IPandPort)
			}
		case "Broadcast Transaction":
			transaction := msg.Transaction
			C.BroadcastTransaction(transaction)
		default:
			fmt.Println("No match case found for: " + msg.Msgtype)
		}

	}
}

func (C *Client) takeInput() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		text, _ := reader.ReadString('\n')
		txt := strings.TrimSpace(text)
		if txt == "quit" {
			return
		} else if txt == "printLedger" {
			C.printLedger()
		} else if txt == "printPeers" {
			fmt.Println(C.peers)
		} else if txt == "makeAccount" {
			C.addAccount()
		} else if txt == "Transaction" {
			t := C.RequestTransactionInfo()
			C.BroadcastTransaction(t)
		}
	}
}

func (C *Client) RequestTransactionInfo() SignedTransaction {
	id := C.getID()
	from := C.requestFrom()
	to := C.requestTo()
	amt := C.requestAmount()

	m := (id + from + to + strconv.Itoa(amt))

	sign := sign(m, C.LocalAccounts[from].SigningKey, C.LocalAccounts[from].Modular)
	fmt.Print("signature: ")
	fmt.Println(sign)
	t := SignedTransaction{ID: id, From: from, To: to, Amount: amt, Signature: sign}
	return t

}

func (C *Client) requestFrom() string {
	fmt.Println("From: ")
	fmt.Print("> ")
	from, err := C.reader.ReadString('\n')
	from = strings.TrimSpace(from)
	if err != nil {
		fmt.Println("formatting error: " + from)
		return C.requestFrom()
	}
	if from == "new" {
		acc := C.addAccount()
		return PublicKey(acc)
	} else if C.validLocalAccount(from) {
		return from
	}
	return C.requestFrom()
}

func (C *Client) addAccount() Account {
	account := KeyGen.MakeAccount()
	C.LocalAccounts[PublicKey(account)] = account
	fmt.Println("new account create with publickey--> " + PublicKey(account))
	return account
}

func (C *Client) validLocalAccount(from string) bool {
	_, contains := C.LocalAccounts[from]
	return contains
}

func (C *Client) validToAccount(to string) bool {
	_, exists := C.ledger.Accounts[to]
	local := C.validLocalAccount(to)
	return exists || local
}

func (C *Client) requestTo() string {
	fmt.Println("To: ")
	fmt.Print("> ")
	to, err := C.reader.ReadString('\n')
	to = strings.TrimSpace(to)
	if err != nil {
		fmt.Println("formatting error: " + to)
		return C.requestTo()
	}
	if to == "new" {
		acc := C.addAccount()
		return PublicKey(acc)
	}
	if C.validToAccount(to) {
		return to
	}
	return C.requestTo()
}
func (C *Client) requestAmount() int {
	fmt.Println("Amount:")
	fmt.Print("> ")
	amount, err := C.reader.ReadString('\n')

	if err != nil {
		fmt.Println("formatting error: " + amount)
		return C.requestAmount()
	}
	amount = strings.TrimSpace(amount)
	amt, err := strconv.Atoi(amount)
	if err != nil {
		fmt.Println("formatting error: amount not an integer")
		return C.requestAmount()
	}
	return amt
}

func (C *Client) PrintHostNames() {
	// _ is convention for throwing the return value away
	name, _ := os.Hostname()
	addrs, _ := net.LookupHost(name)
	fmt.Println("Name: " + name)

	for indx, addr := range addrs {
		fmt.Println("Address number " + strconv.Itoa(indx) + ": " + addr)
	}
}

func main() {
	KeyGen = MakeKeyGenerator()
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
	fmt.Println("When the client is running the following commands exist:")
	fmt.Println("Transaction   -   start requesting info to complete a transaction")
	fmt.Println("makeAccount   -   creates a new account and prints the id which is the public key")
	fmt.Println("printLedger   -   prints the clients version of the ledger")
	fmt.Println("printPeers    -   print the peers the client is connected to")
	fmt.Println("quit          -   shut down the client")
	fmt.Println("*******************************************************************")
	client.takeInput()

}
