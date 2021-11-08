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
	ledger        *Ledger
	peers         []string
	conns         Conns
	reader        *bufio.Reader
	IPandPort     string
	index         int
	transactions  []string
	localAccounts map[string]Account
	SPublicKey    string
	SPrivateKey   string
}

type Ledger struct {
	Accounts map[string]int
	lock     sync.Mutex
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
	block Block
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
	client.transactions = []string{}
	client.localAccounts = make(map[string]Account)
	client.reader = bufio.NewReader(os.Stdin)
	if IPandPort == "" {
		client.StartNetwork()
	} else {
		client.ConnectToNetwork(IPandPort)
	}

	/*
		for _, conn := range client.conns.m {
			go client.HandleConnection(conn)
		}
		go client.Listen(client.StartListen())*/
	return client
}

func MakeLedger() *Ledger {
	ledger := new(Ledger)
	ledger.Accounts = make(map[string]int)
	return ledger
}

/*Threads*/

func (C *Client) StartNetwork() {
	C.PrintFromClient("Starting new network")
	ln := C.StartListen()
	C.peers = append(C.peers, C.IPandPort)
	d, e, n := GenerateKeys(257)
	C.SPublicKey = KeyToString(e, n)
	C.SPrivateKey = KeyToString(d, n)
	C.Phase1(ln)
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
		C.ConnectToPeers()
		C.Listen(ln)
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
			go C.Phase1(conn)
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

func (C *Client) Phase1(conn net.Conn){
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
		case "Phase 2":
			C.Broadcast(msg)
			go C.Phase2(conn)
			break
		default:
			C.PrintFromClient("No match case found for: " + msg.Msgtype)
		}
	}
}

func (C *Client) Phase2(conn net.Conn) {
	for {
		dec := gob.NewDecoder(conn)
		msg := Message{}
		err := dec.Decode(&msg)
		if err != nil {
			conn.Close()
			panic(err)
		}
		switch msg.Msgtype {
		case "Broadcast Transaction":
			transaction := msg.Transaction
			C.Broadcast(Message{Msgtype: "BroadCast Transaction", Transaction: transaction})
		case "Broadcast Block":
			block := msg.
		default:
			C.PrintFromClient("No match case found for: " + msg.Msgtype)
		}

	}
}

func (C *Client) Broadcast(m Message) {
	for k := range C.conns.m {
		enc := gob.NewEncoder(C.conns.m[k])
		err := enc.Encode(m)
		if err != nil {
			panic(err)
		}
	}
}

/* Test */
func Test(C1 *Client, C2 *Client) {

}

/*Helper functions*/

func (C *Client) StartListen() net.Listener {
	ln, _ := net.Listen("tcp", ":0")
	IP := getIP()
	Port := strings.TrimPrefix(ln.Addr().String(), "[::]:")
	C.IPandPort = IP + ":" + Port
	fmt.Println("Listening for connections on: <" + C.IPandPort + ">")
	return ln
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
	return C.IPandPort + ":" + strconv.Itoa(C.index) + ":" + strconv.Itoa(len(C.transactions)+1)
}

func (C *Client) PeerExists(peer string) bool {
	for p := range C.peers {
		if C.peers[p] == peer {
			return true
		}
	}
	return false
}
