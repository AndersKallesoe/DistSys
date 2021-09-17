package account

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
	"net"
	"sync"
)

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

func main() {
	//initialize the client
	client := makeClient()

}
