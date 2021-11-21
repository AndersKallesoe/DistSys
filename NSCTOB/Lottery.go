package main

import (
	"crypto/rand"
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

/*
1. genesis block
	seed
	Initial ledger
	pk's

proof of stake protocol:
each second(slot)
1. compute draw = sig_vk_i ("Lottery", Seed ,Slot )
2. compute val = stake * H("Lottery", Seed, vk_i, Draw)
3. if val>= hardness
	broadcast block

*/
var SlotLength int
var Hardness int
var KeyGen KeyGenerator

func getVal(slot int, draw int) int {
	return 0
}

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
	PublicKey           string
	PrivateKey          string
	lock                sync.Mutex
	blocks              map[string]Block
	seed                *big.Int
	LastBlock           string
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

type Conns struct {
	m map[string]GobConn

	mutex sync.Mutex
}

type GobConn struct {
	conn      net.Conn
	enc       *gob.Encoder
	dec       *gob.Decoder
	PublicKey string
}

type Block struct {
	Predecessor string
	BlockNumber int
	Seed        *big.Int
	Ledger      map[string]int
	IDList      []string
	Signature   string
}

type Message struct {
	Msgtype     string
	Transaction SignedTransaction
	IPandPort   string
	Peers       []string
	PublicKey   string
	Block       Block
	slot        int
	Draw        *big.Int
}

/*Main function*/

func main() {
	Hardness = 99

	KeyGen = MakeKeyGenerator()
	i, _ := rand.Int(rand.Reader, big.NewInt(191919191916843213))
	seed := Hash(i)

	Client1 := makeClient()
	Client2 := makeClient()
	Client3 := makeClient()
	Client4 := makeClient()
	Client5 := makeClient()
	Client6 := makeClient()
	Client7 := makeClient()
	Client8 := makeClient()
	Client9 := makeClient()
	Client10 := makeClient()

	ledger := make(map[string]int)
	ledger[Client1.PublicKey] = 1000000
	ledger[Client2.PublicKey] = 1000000
	ledger[Client3.PublicKey] = 1000000
	ledger[Client4.PublicKey] = 1000000
	ledger[Client5.PublicKey] = 1000000
	ledger[Client6.PublicKey] = 1000000
	ledger[Client7.PublicKey] = 1000000
	ledger[Client8.PublicKey] = 1000000
	ledger[Client9.PublicKey] = 1000000
	ledger[Client10.PublicKey] = 1000000

	GBlock := Block{BlockNumber: 1, Seed: seed, Ledger: ledger, Signature: ""}
	d, n := SplitKey(Client1.PrivateKey)
	GBlock.Signature = signBlock(GBlock, d, n)

	go Client1.StartNetwork(GBlock)
	time.Sleep(time.Second)
	go Client2.ConnectToNetwork(Client1.IPandPort)
	time.Sleep(time.Second)
	go Client3.ConnectToNetwork(Client1.IPandPort)
	time.Sleep(time.Second)
	go Client4.ConnectToNetwork(Client1.IPandPort)
	time.Sleep(time.Second)
	go Client5.ConnectToNetwork(Client1.IPandPort)
	time.Sleep(time.Second)
	go Client6.ConnectToNetwork(Client1.IPandPort)
	time.Sleep(time.Second)
	go Client7.ConnectToNetwork(Client1.IPandPort)
	time.Sleep(time.Second)
	go Client8.ConnectToNetwork(Client1.IPandPort)
	time.Sleep(time.Second)
	go Client9.ConnectToNetwork(Client1.IPandPort)
	time.Sleep(time.Second)
	go Client10.ConnectToNetwork(Client1.IPandPort)
	time.Sleep(time.Second * 2)
	startingTime := time.Now().Add(time.Second * 3)

	go Client1.ParticipateInLottery(startingTime)
	go Client2.ParticipateInLottery(startingTime)
	go Client3.ParticipateInLottery(startingTime)
	go Client4.ParticipateInLottery(startingTime)
	go Client5.ParticipateInLottery(startingTime)
	go Client6.ParticipateInLottery(startingTime)
	go Client7.ParticipateInLottery(startingTime)
	go Client8.ParticipateInLottery(startingTime)
	go Client9.ParticipateInLottery(startingTime)
	go Client10.ParticipateInLottery(startingTime)

	fmt.Println(Client10.ledger.Accounts)
}

/*test */

/* test that the lottery works as intended*/
func testLottery() {
	Hardness = 99
	KeyGen = MakeKeyGenerator()
	i, _ := rand.Int(rand.Reader, big.NewInt(191919191916843213))
	seed := Hash(i)
	Client1 := makeClient()
	ledger := make(map[string]int)
	ledger[Client1.PublicKey] = 1000000
	Client1.ledger.Accounts = ledger
	for i := 0; i < 1000; i++ {
		draw, won := Client1.PlayLottery(seed, i)
		if won {
			fmt.Println("won", won, "draw:", draw)
		}

	}
	Client1.PlayLottery(seed, 1)

}

/*make methods*/
func makeClient() *Client {
	client := new(Client)
	client.ledger = MakeLedger()
	client.peers = []string{}
	client.conns = Conns{m: make(map[string]GobConn)}
	client.index = 0
	client.pendingTransactions = MakePendingTransactions()
	client.postedTransactions = MakePostedTransactions()
	client.LocalAccounts = make(map[string]Account)
	d, e, n := GenerateKeys(257)
	client.PublicKey = KeyToString(e, n)
	client.PrivateKey = KeyToString(d, n)
	client.blocks = make(map[string]Block)
	return client
}
func MakeLedger() *Ledger {
	ledger := new(Ledger)
	ledger.Accounts = make(map[string]int)
	return ledger
}

func (C *Client) MakeSignedTransaction(To string, Amount int) (bool, SignedTransaction) {
	if Amount < 1 {
		return false, SignedTransaction{}
	}
	C.ledger.lock.Lock()
	defer C.ledger.lock.Unlock()
	balance := C.ledger.Accounts[C.PublicKey]
	if balance <= Amount {
		return false, SignedTransaction{}
	}
	ID := C.getID()
	d, n := SplitKey(C.PrivateKey)
	Signature := sign(ID+C.PublicKey+To+strconv.Itoa(Amount), d, n)
	return true, SignedTransaction{ID: ID, From: C.PublicKey, To: To, Amount: Amount, Signature: Signature}
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

func (C *Client) getID() string {
	C.index = C.index + 1
	return C.IPandPort + ":" + strconv.Itoa(C.index)
}

func (conns *Conns) Set(key string, val net.Conn, pk string) {
	conns.m[key] = GobConn{val, gob.NewEncoder(val), gob.NewDecoder(val), pk}
}

func buildBlockString(B Block) string {
	numberstring := strconv.Itoa(B.BlockNumber)
	seedstring := intToString(B.Seed)
	blockAsString := []string{numberstring, seedstring}
	blockAsString = append(blockAsString, B.IDList...)
	return strings.Join(blockAsString, "")
}

func HashBlock(B Block) string {
	return intToString(Hash(stringToInt(buildBlockString(B))))
}

func signBlock(B Block, d *big.Int, n *big.Int) string {
	blockstring := buildBlockString(B)
	return sign(blockstring, d, n)
}

func verifyblock(B Block, e *big.Int, n *big.Int) bool {
	blockstring := buildBlockString(B)

	return verify(blockstring, B.Signature, e, n)
}

func (C *Client) ComputeDraw(seed *big.Int, slot int) *big.Int {
	signString := "lottery" + intToString(seed) + strconv.Itoa(slot)
	d, n := SplitKey(C.PrivateKey)
	draw := sign(signString, d, n)
	return stringToInt(draw)
}

func (C *Client) ComputeVal(seed *big.Int, slot int, draw *big.Int, publicKey string) *big.Int {
	C.ledger.lock.Lock()
	defer C.ledger.lock.Unlock()
	tickets := big.NewInt(int64(1000000)) // change to C.ledger.Accounts[C.PublicKey]
	stringToHash := "lottery" + intToString(seed) + strconv.Itoa(slot) + publicKey + intToString(draw)
	hash := Hash(stringToInt(stringToHash))
	val := big.NewInt(0)
	val.Mul(tickets, hash)
	return val
}

func (C *Client) PlayLottery(seed *big.Int, slot int) (*big.Int, bool) {
	draw := C.ComputeDraw(seed, slot)
	val := C.ComputeVal(seed, slot, draw, C.PublicKey)
	hardness := big.NewInt(2)
	hardness.Exp(hardness, big.NewInt(256), nil)
	hardness.Mul(hardness, big.NewInt(1000000))
	hardness.Mul(hardness, big.NewInt(int64(Hardness)))
	hardness.Div(hardness, big.NewInt(100))
	won := val.Cmp(hardness) >= 0
	return draw, won
}

func (C *Client) ParticipateInLottery(startTime time.Time) {
	/*
		new block every 10 seconds: 100 draws for all clients 1 must win
		set hardness such that there is a 1 % chance to win for a draw
		max hashsize 2^256
		max draw (initial situation): 1000000 * max hashsize
		hardness = 0,99 * max draw
	*/
	currentSlot := 0
	for {
		if time.Now().After(startTime.Add(time.Second * time.Duration(currentSlot))) {
			draw, won := C.PlayLottery(C.seed, currentSlot)
			if won {
				block := C.CreateBlock(C.LastBlock)
				C.Broadcast(Message{Msgtype: "Broadcast Block", Transaction: SignedTransaction{}, Block: block, PublicKey: C.PublicKey, slot: currentSlot, Draw: draw})
			}
			currentSlot++
			time.Sleep(time.Millisecond * 900)
		}
	}
}

func (C *Client) verifyDraw(seed *big.Int, slot int, draw *big.Int, publicKey string) bool {
	signString := "lottery" + intToString(seed) + strconv.Itoa(slot)
	e, n := SplitKey(publicKey)
	return verify(signString, intToString(draw), e, n)
}

func (C *Client) verifyLotteryWinner(seed *big.Int, slot int, draw *big.Int, publicKey string) bool {
	val := C.ComputeVal(seed, slot, draw, publicKey)
	hardness := big.NewInt(2)
	hardness.Exp(hardness, big.NewInt(256), nil)
	hardness.Mul(hardness, big.NewInt(1000000))
	hardness.Mul(hardness, big.NewInt(int64(Hardness)))
	hardness.Div(hardness, big.NewInt(100))
	won := val.Cmp(hardness) >= 0
	return won
}

func (C *Client) StartNetwork(GBlock Block) {

	ln := C.StartListen()
	C.PrintFromClient("Starting new network")
	C.peers = append(C.peers, C.IPandPort)
	C.ledger.Accounts = GBlock.Ledger
	key := intToString(Hash(stringToInt(buildBlockString(GBlock))))
	C.blocks[key] = GBlock
	C.seed = GBlock.Seed
	go C.Listen(ln)
	time.Sleep(time.Second * 11)
	C.PrintFromClient("i sent the genesis block")
	C.Broadcast(Message{Msgtype: "Genesis Block", Transaction: SignedTransaction{}, Block: GBlock})
	for {
	}
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
		C.PublicKey = msg.PublicKey
		conn.Close()
		C.ConnectToPeers()
		for _, conn := range C.conns.m {
			go C.HandleConnection(conn)
		}
		C.Listen(ln)
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
			dec := gob.NewDecoder(conn)
			msg := Message{}
			err = dec.Decode(&msg)
			if err != nil {
				panic(err)
			}
			C.conns.mutex.Lock()
			C.conns.Set(conn.RemoteAddr().String(), conn, msg.PublicKey)
			C.conns.mutex.Unlock()
		}
	}
	C.Broadcast(Message{Msgtype: "Broadcast Presence", IPandPort: C.IPandPort, Transaction: SignedTransaction{}, Block: Block{}})
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
			peers := Message{Peers: C.peers, Transaction: SignedTransaction{}, Block: Block{}}
			enc := gob.NewEncoder(conn)
			err = enc.Encode(&peers)
			if err != nil {
				panic(err)
			}
		case "Connection":
			C.conns.mutex.Lock()
			C.conns.Set(conn.RemoteAddr().String(), conn, msg.PublicKey)
			C.conns.mutex.Unlock()
			pk := Message{PublicKey: C.PublicKey}
			enc := gob.NewEncoder(conn)
			err = enc.Encode(&pk)
			if err != nil {
				panic(err)
			}
			go C.HandleConnection(C.conns.m[conn.RemoteAddr().String()])
		default:
			fmt.Println("No match case found for: " + msg.Msgtype)
		}
	}

}

func (C *Client) HandleConnection(gc GobConn) {
	for {
		dec := gc.dec
		msg := Message{}
		if err := dec.Decode(&msg); err != nil {
			fmt.Println(C.IPandPort)
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
			exists, _ := C.TransactionExists(transaction.ID)
			if !exists {
				C.pendingTransactions.lock.Lock()
				C.pendingTransactions.Transactions = append(C.pendingTransactions.Transactions, transaction)
				C.pendingTransactions.lock.Unlock()
				C.Broadcast(Message{Msgtype: "Broadcast Transaction", Transaction: transaction, Block: Block{}})
			}

		case "Genesis Block":
			C.PrintFromClient("i recieved the genesis block")
			block := msg.Block
			e, n := SplitKey(gc.PublicKey)
			blockverified := verifyblock(block, e, n)
			if blockverified {
				C.ledger.Accounts = block.Ledger
				key := intToString(Hash(stringToInt(buildBlockString(block))))
				C.blocks[key] = block
				C.seed = block.Seed
			}
		case "Broadcast Block": // potentielt noget bøvl ift hvornår blocks bliver broadcastet videre: (overvej block exists metode )
			block := msg.Block
			e, n := SplitKey(gc.PublicKey)
			key := HashBlock(block)
			C.blocks[key] = block
			legalblock := block.BlockNumber == C.blocks[C.LastBlock].BlockNumber+1 && verifyblock(block, e, n) &&
				C.verifyDraw(C.seed, msg.slot, msg.Draw, msg.PublicKey) &&
				C.verifyLotteryWinner(C.seed, msg.slot, msg.Draw, msg.PublicKey)
			if legalblock {
				C.PostBlock(block)
				C.Broadcast(Message{Msgtype: "Broadcast Block", Transaction: SignedTransaction{}, Block: block})
				C.LastBlock = key
			}
		default:
			C.PrintFromClient("No match case found for: " + msg.Msgtype)
		}

	}
}

func (C *Client) CreateBlock(Predecessor string) Block {
	/*blocknr := -1*/
	transactions := []string{}
	C.pendingTransactions.lock.Lock()
	for t := range C.pendingTransactions.Transactions {
		st := C.pendingTransactions.Transactions[t]
		C.PostTransaction(st)
		transactions = append(transactions, st.ID)
	}
	C.pendingTransactions.Transactions = []SignedTransaction{}
	C.pendingTransactions.lock.Unlock()

	block := Block{Predecessor: Predecessor, BlockNumber: 1, IDList: transactions}
	e, n := SplitKey(C.PrivateKey)
	block.Signature = signBlock(block, e, n)
	C.LastBlock = HashBlock(block)
	return block
}

func (C *Client) PostBlock(block Block) {
	//verify block
	for b := range block.IDList {
		id := block.IDList[b]
		exists, transaction := C.TransactionExists(id)
		for !exists {
			time.Sleep(time.Microsecond)
			exists, transaction = C.TransactionExists(id)
		}
		C.PostTransaction(transaction)
	}
	C.PrintFromClient("Block posted")
}

func (C *Client) PostTransaction(t SignedTransaction) {
	C.ledger.lock.Lock()
	defer C.ledger.lock.Unlock()
	s := t.ID + t.From + t.To + strconv.Itoa(t.Amount)
	v, m := SplitKey(t.From)
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

func (C *Client) Broadcast(m Message) {
	for k := range C.conns.m {
		if err := C.conns.m[k].enc.Encode(&m); err != nil {
			panic(err)
		}
	}
}

func (C *Client) PrintFromClient(s string) {
	fmt.Println(C.IPandPort + " --> " + s)
}

func getIP() string {
	// _ is convention for throwing the return value away
	name, _ := os.Hostname()
	addrs, _ := net.LookupHost(name)
	IP := addrs[len(addrs)-1]
	return IP
}

func (C *Client) TransactionExists(transaction string) (bool, SignedTransaction) {
	C.postedTransactions.lock.Lock()
	defer C.postedTransactions.lock.Unlock()
	for p := range C.postedTransactions.Transactions {
		if C.postedTransactions.Transactions[p] == transaction {
			return true, SignedTransaction{}
		}
	}

	C.pendingTransactions.lock.Lock()
	defer C.pendingTransactions.lock.Unlock()
	for p := range C.pendingTransactions.Transactions {
		if C.pendingTransactions.Transactions[p].ID == transaction {
			return true, C.pendingTransactions.Transactions[p]
		}
	}
	return false, SignedTransaction{}
}

func (C *Client) PeerExists(peer string) bool {
	for p := range C.peers {
		if C.peers[p] == peer {
			return true
		}
	}
	return false
}
