package main

import (
	"encoding/gob"
	"fmt"
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
	IPandPort           string
	index               int
	pendingTransactions *PendingTransactions
	postedTransactions  *PostedTransactions
	LocalAccounts       map[string]Account
	SPublicKey          string
	SPrivateKey         string
	lock                sync.Mutex
}

type Ledger struct {
	Accounts map[string]int
	lock     sync.Mutex
}

type PendingTransactions struct {
	Transactions map[string]SignedTransaction
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

type Conns struct {
	m map[string]GobConn

	mutex sync.Mutex
}

type GobConn struct {
	conn net.Conn
	enc  *gob.Encoder
	dec  *gob.Decoder
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
	SPublicKey  string
	Block       Block
}

/*Main function*/

func main() {
	KeyGen = MakeKeyGenerator()
	// Initialize the client
	sequencer := makeClient()
	fmt.Println("Sequencer made!")
	go sequencer.StartNetwork()
	time.Sleep(time.Second) //give the system a second to catch up
	client2 := makeClient()
	go client2.ConnectToNetwork(sequencer.IPandPort)
	time.Sleep(time.Second) //give the system a second to catch up
	client3 := makeClient()
	go client3.ConnectToNetwork(sequencer.IPandPort)

	accounta := sequencer.makeAccount()
	accountb := client2.makeAccount()
	accountc := client3.makeAccount()

	ledger := make(map[string]int)

	ledger[PublicKey(accounta)] = 1000
	ledger[PublicKey(accountb)] = 1000
	ledger[PublicKey(accountc)] = 1000

	sequencer.ledger.Accounts = ledger

	client2.ledger.Accounts = ledger

	client3.ledger.Accounts = ledger
	time.Sleep(time.Second) //give the system a second to catch up
	fmt.Println("running tests")
	go Test(client2, accountb, accounta)
	Test(client2, accountb, accountc)
	fmt.Println("Sequencer ledger:")
	sequencer.printLedger()
	fmt.Println("Client2 ledger:")
	client2.printLedger()
	fmt.Println("Client3 ledger:")
	client3.printLedger()
}

/*Creators*/

func makeClient() *Client {
	client := new(Client)
	client.ledger = MakeLedger()
	client.peers = []string{}
	client.conns = Conns{m: make(map[string]GobConn)}
	client.index = 0
	client.pendingTransactions = MakePendingTransactions()
	client.postedTransactions = MakePostedTransactions()
	client.LocalAccounts = make(map[string]Account)
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
	p.Transactions = make(map[string]SignedTransaction)
	return p
}

func (C *Client) makeAccount() Account {
	account := KeyGen.MakeAccount()
	C.LocalAccounts[PublicKey(account)] = account
	fmt.Println("new account create with publickey--> " + PublicKey(account))
	return account
}

func (C *Client) makeTransaction(from Account, to Account, amt int) {
	if !C.validLocalAccount(PublicKey(from)) {
		panic("illegal transaction")
	}
	if !C.validToAccount(PublicKey(to)) {
		panic("illegal transaction")
	}
	t := SignedTransaction{C.getID(), PublicKey(from), PublicKey(to), amt, ""}
	sign := sign(t.ID+t.From+t.To+strconv.Itoa(t.Amount), from.SigningKey, from.Modular)
	t.Signature = sign
	C.pendingTransactions.lock.Lock()
	C.pendingTransactions.Transactions[t.ID] = t
	C.pendingTransactions.lock.Unlock()
	//C.PrintFromClient("broadcasting transaction")
	C.Broadcast(Message{Msgtype: "Broadcast Transaction", Transaction: t, Block: Block{}})
}

/*Threads*/

func (C *Client) StartNetwork() {
	C.PrintFromClient("Starting new network")
	ln := C.StartListen()
	C.peers = append(C.peers, C.IPandPort)
	d, e, n := GenerateKeys(257)
	C.SPublicKey = KeyToString(e, n)
	C.SPrivateKey = KeyToString(d, n)
	go C.CreateBlocks()
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
		request := Message{Msgtype: "Requesting Peers", Transaction: SignedTransaction{}, Block: Block{}}
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
		for _, conn := range C.conns.m {
			go C.HandleConnection(conn)
		}
		C.Listen(ln)
	}

}

func (C *Client) Listen(ln net.Listener) {
	defer ln.Close()
	//fmt.Println("i listen")
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
			peers := Message{Peers: C.peers, SPublicKey: C.SPublicKey, Transaction: SignedTransaction{}, Block: Block{}}
			enc := gob.NewEncoder(conn)
			err = enc.Encode(&peers)
			if err != nil {
				panic(err)
			}
		case "Connection":
			C.conns.mutex.Lock()
			C.conns.Set(conn.RemoteAddr().String(), conn)
			C.conns.mutex.Unlock()
			go C.HandleConnection(C.conns.m[conn.RemoteAddr().String()])
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
		} else if err != nil {
			return
		} else {
			enc := gob.NewEncoder(conn)
			request := Message{Msgtype: "Connection", Transaction: SignedTransaction{}, Block: Block{}}
			err := enc.Encode(request)
			if err != nil {
				panic(err)
			}
			C.conns.mutex.Lock()
			C.conns.Set(conn.RemoteAddr().String(), conn)
			C.conns.mutex.Unlock()
		}
	}
	C.Broadcast(Message{Msgtype: "Broadcast Presence", IPandPort: C.IPandPort, Transaction: SignedTransaction{}, Block: Block{}})
}

func (C *Client) HandleConnection(gc GobConn) {
	blocknr := 0
	fmt.Print("handling connection between: ")
	fmt.Print(gc.conn.LocalAddr())
	fmt.Print(" and ")
	fmt.Println(gc.conn.RemoteAddr())

	for {
		dec := gc.dec
		/*var receivedData string
		if err := dec.Decode(&receivedData); err != nil {
			// handle error
		}
		finalData := strings.Split(receivedData, " ")
		msg := Message{}

		if finalData[0] == "newblock" {
			if err := dec.Decode(&msg); err != nil {
				panic(err)
			}
		} else {
			fmt.Print("finalData: ")
			fmt.Println(finalData)
			fmt.Print("receivedData: ")
			fmt.Println(receivedData)
			panic("error in decoding" + finalData[0])
		}*/
		msg := Message{}
		if err := dec.Decode(&msg); err != nil {
			panic(err)
		}
		switch msg.Msgtype {
		case "Broadcast Presence":
			if !C.PeerExists(msg.IPandPort) {
				C.peers = append(C.peers, msg.IPandPort)
				C.Broadcast(msg)
			}
		case "Broadcast Transaction":
			transaction := msg.Transaction
			if !C.TransactionExists(transaction.ID) {
				C.pendingTransactions.lock.Lock()
				C.pendingTransactions.Transactions[transaction.ID] = transaction
				C.pendingTransactions.lock.Unlock()
				C.Broadcast(Message{Msgtype: "Broadcast Transaction", Transaction: transaction, Block: Block{}})
			}
		case "Broadcast Block":
			block := msg.Block
			if block.BlockNumber == blocknr {
				C.PostBlock(block)
				blocknr++
				C.Broadcast(Message{Msgtype: "Broadcast Block", Transaction: SignedTransaction{}, Block: block})
			} else if block.BlockNumber > blocknr {
				panic("A block was recieved out of order")
			}
		default:
			C.PrintFromClient("No match case found for: " + msg.Msgtype)
		}

	}
}

/* Test */
func Test(C *Client, from Account, to Account) {
	for i := 0; i < 1000; i++ {
		time.Sleep(time.Millisecond)
		C.lock.Lock()
		C.makeTransaction(from, to, 1)
		C.lock.Unlock()
	}
}

/*Helper functions*/

func (C *Client) printLedger() {
	for k, v := range C.ledger.Accounts {
		fmt.Println("Account: " + k + " Balance: " + strconv.Itoa(v))
	}
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

func (C *Client) Broadcast(m Message) {
	for k := range C.conns.m {
		/*if err := C.conns.m[k].enc.Encode("newblock add "); err != nil {
			panic(err)
		}*/
		if err := C.conns.m[k].enc.Encode(&m); err != nil {
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
	C.ledger.lock.Lock()
	defer C.ledger.lock.Unlock()
	s := t.ID + t.From + t.To + strconv.Itoa(t.Amount)
	v, m := SplitPublicKey(t.From)
	if !verify(s, t.Signature, v, m) {
		C.PrintFromClient("signature invalid on transaction: " + t.ID)
		return
	}
	if !(C.ledger.Accounts[t.From]-t.Amount >= 0) {
		//C.PrintFromClient("amount to large on transaction: " + t.ID) too many prints
		return
	}
	C.ledger.Accounts[t.From] -= t.Amount
	C.ledger.Accounts[t.To] += t.Amount
}

func (C *Client) CreateBlocks() {
	blocknr := -1
	for {
		time.Sleep(time.Second * 10)
		blocknr++
		transactions := []string{}
		C.pendingTransactions.lock.Lock()
		C.PrintFromClient("Sequencer now in CreateBlocks method!")
		for t := range C.pendingTransactions.Transactions {
			st := C.pendingTransactions.Transactions[t]
			C.PostTransaction(st)
			transactions = append(transactions, st.ID)
		}
		C.pendingTransactions.Transactions = make(map[string]SignedTransaction)
		C.pendingTransactions.lock.Unlock()
		C.Broadcast(Message{Msgtype: "Broadcast Block", Transaction: SignedTransaction{}, Block: Block{blocknr, transactions}})
		C.PrintFromClient("Block made and broadcast!")
	}
}

func (C *Client) PostBlock(block Block) {
	//verify block
	for b := range block.IDList {
		id := block.IDList[b]
		for !C.TransactionExists(id) {
			time.Sleep(time.Microsecond)
		}
		C.pendingTransactions.lock.Lock()
		transaction := C.pendingTransactions.Transactions[id]
		C.pendingTransactions.lock.Unlock()
		C.PostTransaction(transaction)

	}
	C.PrintFromClient("Block posted")
}

func (C *Client) TransactionExists(transaction string) bool {
	C.postedTransactions.lock.Lock()
	defer C.postedTransactions.lock.Unlock()
	for p := range C.postedTransactions.Transactions {
		if C.postedTransactions.Transactions[p] == transaction {
			return true
		}
	}

	C.pendingTransactions.lock.Lock()
	defer C.pendingTransactions.lock.Unlock()
	for p := range C.pendingTransactions.Transactions {
		if C.pendingTransactions.Transactions[p].ID == transaction {
			return true
		}
	}
	return false
}

func (conns *Conns) Set(key string, val net.Conn) {
	conns.m[key] = GobConn{val, gob.NewEncoder(val), gob.NewDecoder(val)}
}

func getIP() string {
	// _ is convention for throwing the return value away
	name, _ := os.Hostname()
	addrs, _ := net.LookupHost(name)
	IP := addrs[len(addrs)-1]
	return IP
}

func (C *Client) PrintFromClient(s string) {
	fmt.Println(C.IPandPort + " --> " + s)
}

func (C *Client) getID() string {
	C.index = C.index + 1
	return C.IPandPort + ":" + strconv.Itoa(C.index)
}

func (C *Client) PeerExists(peer string) bool {
	for p := range C.peers {
		if C.peers[p] == peer {
			return true
		}
	}
	return false
}
