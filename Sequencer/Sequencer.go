package main

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
	"time"
)

var KeyGen KeyGenerator

/*Structs*/

type Client struct {
	ledger              *Ledger
	peers               []string
	conns               Conns
	reader              *bufio.Reader
	IPandPort           string
	index               int
	pendingTransactions *PendingTransactions
	postedTransactions  *PostedTransactions
	localAccounts       map[string]Account
	SPublicKey          string
	SPrivateKey         string
}

type Ledger struct {
	Accounts map[string]int
	lock     sync.Mutex
}

type PendingTransactions struct {
	Transactions []SignedTransaction
	lock         sync.Mutex
}

type PostedTransactions struct {
	Transactions []string
	lock         sync.Mutex
}

type SignedTransaction struct {
	ID        string // Any string
	From      string // A verification key coded as a string
	To        string // A verification key coded as a string
	Amount    int    // Amount to transfer
	Signature string // Signature coded as string
}

type KeySet struct {
	VerificationKey *big.Int
	SigningKey      *big.Int
	Modular         *big.Int
}

type Conns struct {
	m     map[string]net.Conn
	mutex sync.Mutex
}

type Block struct {
	BlockNumber int
	IDList      []string
}

type Message struct {
	Msgtype     string
	Transaction SignedTransaction
	IPandPort   string
	Peers       []string
	AccountName string
	KeySet      KeySet
	SPublicKey  string
	Block       Block
}

/*Main function*/

func main() {
	KeyGen = MakeKeyGenerator()
	// Initialize the client
	client1 := makeClient("")
	client2 := makeClient(client1.IPandPort)
	client3 := makeClient(client1.IPandPort)
	client1.Broadcast(Message{Msgtype: "Phase 2"})
	time.Sleep(time.Second * 5)
	Test(client1, client2)
	Test(client1, client3)
}

/*Creators*/

func makeClient(IPandPort string) *Client {
	client := new(Client)
	client.ledger = MakeLedger()
	client.peers = []string{}
	client.conns = Conns{m: make(map[string]net.Conn)}
	client.index = 0
	client.pendingTransactions = MakePendingTransactions()
	client.postedTransactions = MakePostedTransactions()
	client.localAccounts = make(map[string]Account)
	client.reader = bufio.NewReader(os.Stdin)
	if IPandPort == "" {
		client.StartNetwork()
	} else {
		client.ConnectToNetwork(IPandPort)
	}

	return client
}

func MakeLedger() *Ledger {
	ledger := new(Ledger)
	ledger.Accounts = make(map[string]int)
	return ledger
}

func MakePostedTransactions() *PostedTransactions {
	p := new(PostedTransactions)
	p.Transactions = []string{}
	return p
}

func MakePendingTransactions() *PendingTransactions {
	p := new(PendingTransactions)
	p.Transactions = []SignedTransaction{}
	return p
}

/*Threads*/

func (C *Client) StartNetwork() {
	C.PrintFromClient("Starting new network")
	ln := C.StartListen()
	C.peers = append(C.peers, C.IPandPort)
	d, e, n := GenerateKeys(257)
	C.SPublicKey = KeyToString(e, n)
	C.SPrivateKey = KeyToString(d, n)
	go C.ManageBlocks()
	C.Listen(ln)
}

func (C *Client) ConnectToNetwork(IPAndPort string) {
	conn, err := net.Dial("tcp", IPAndPort)
	if conn == nil {
		panic("no connection")
	} else if err != nil {
		panic(err)
	} else {
		enc := gob.NewEncoder(conn)
		request := Message{Msgtype: "Requesting Peers"}
		err := enc.Encode(&request)
		if err != nil {
			panic(err)
		}
		dec := gob.NewDecoder(conn)
		msg := Message{}
		err = dec.Decode(&msg)
		if err != nil {
			panic(err)
		}
		ln := C.StartListen()
		C.peers = append(C.peers, msg.Peers...)
		C.peers = append(C.peers, C.IPandPort)
		C.SPublicKey = msg.SPublicKey
		conn.Close()
		C.Listen(ln)
		C.ConnectToPeers()
	}

}

func (C *Client) Listen(ln net.Listener) {
	defer ln.Close()
	for {
		conn, _ := ln.Accept()
		msg := Message{}
		dec := gob.NewDecoder(conn)
		err := dec.Decode(&msg)
		if err != nil {
			panic(err)
		}
		switch msg.Msgtype {
		case "Requesting Peers":
			peers := Message{Peers: C.peers, SPublicKey: C.SPublicKey}
			enc := gob.NewEncoder(conn)
			err = enc.Encode(&peers)
			if err != nil {
				panic(err)
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
			C.PrintFromClient("There was an error in connecting to: " + peers[p])
		} else if err != nil {
			return
		} else {
			enc := gob.NewEncoder(conn)
			request := Message{Msgtype: "Connection"}
			err := enc.Encode(request)
			if err != nil {
				panic(err)
			}
			C.conns.mutex.Lock()
			C.conns.Set(conn.RemoteAddr().String(), conn)
			C.conns.mutex.Unlock()
		}
	}
	C.Broadcast(Message{Msgtype: "Broadcast Presence"})
}

func (C *Client) HandleConnection(conn net.Conn) {
	for {
		dec := gob.NewDecoder(conn)
		msg := Message{}
		err := dec.Decode(&msg)
		if err != nil {
			conn.Close()
			panic(err)
		}
		switch msg.Msgtype {
		case "Broadcast Presence":
			if !C.PeerExists(msg.IPandPort) {
				C.peers = append(C.peers, msg.IPandPort)
				C.Broadcast(Message{Msgtype: "Broadcast Presence"})
			}
		case "Broadcast Transaction":
			transaction := msg.Transaction
			if C.TransactionExists(transaction) {
				C.pendingTransactions.lock.Lock()
				C.pendingTransactions.Transactions = append(C.pendingTransactions.Transactions, transaction)
				C.pendingTransactions.lock.Unlock()
				C.Broadcast(Message{Msgtype: "BroadCast Transaction", Transaction: transaction})
			}
		case "Broadcast Block":
			//
			block := msg.Block
			C.Broadcast(Message{Msgtype: "BroadCast Block", Block: block})
		default:
			C.PrintFromClient("No match case found for: " + msg.Msgtype)
		}

	}
}

func (C *Client) ManageBlocks() {
	blocknr := -1
	for {
		time.Sleep(time.Second * 10)
		blocknr++
		transactions := []string{}
		C.pendingTransactions.lock.Lock()

		for t := range C.pendingTransactions.Transactions {
			st := C.pendingTransactions.Transactions[t]
			C.PostTransaction(st)
			transactions = append(transactions, st.ID)
		}

	}

	// broadcast block
}

/* Test */
func Test(C1 *Client, C2 *Client) {

}

/*Helper functions*/

func (C *Client) Broadcast(m Message) {
	for k := range C.conns.m {
		enc := gob.NewEncoder(C.conns.m[k])
		err := enc.Encode(m)
		if err != nil {
			panic(err)
		}
	}
}

func (C *Client) StartListen() net.Listener {
	ln, _ := net.Listen("tcp", ":0")
	IP := getIP()
	Port := strings.TrimPrefix(ln.Addr().String(), "[::]:")
	C.IPandPort = IP + ":" + Port
	fmt.Println("Listening for connections on: <" + C.IPandPort + ">")
	return ln
}

func (C *Client) PostTransaction(t SignedTransaction) {

}

func (C *Client) TransactionExists(transaction SignedTransaction) bool {
	for p := range C.postedTransactions.Transactions {
		if C.postedTransactions.Transactions[p] == transaction.ID {
			return true
		}
	}
	C.pendingTransactions.lock.Lock()
	for p := range C.pendingTransactions.Transactions {
		if C.pendingTransactions.Transactions[p].ID == transaction.ID {
			return true
		}
	}
	C.pendingTransactions.lock.Unlock()
	return false
}

func (conns *Conns) Set(key string, val net.Conn) {
	conns.m[key] = val
}

func getIP() string {
	// _ is convention for throwing the return value away
	name, _ := os.Hostname()
	addrs, _ := net.LookupHost(name)
	IP := addrs[len(addrs)-1]
	return IP
}

func (C *Client) PrintFromClient(s string) {
	fmt.Println(C.getID() + " --> " + s)
}

func (C *Client) getID() string {
	C.index = C.index + 1
	return C.IPandPort + ":" + strconv.Itoa(C.index) + ":" + strconv.Itoa(C.index)
}

func (C *Client) PeerExists(peer string) bool {
	for p := range C.peers {
		if C.peers[p] == peer {
			return true
		}
	}
	return false
}
