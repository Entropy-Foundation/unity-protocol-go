package unityNode

import (
	"encoding/hex"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"net"
	"net/rpc"
	"sync"
	"time"

	db "github.com/syndtr/goleveldb/leveldb"

	. "github.com/gorilla/websocket"
	"github.com/unity-go/api"
	. "github.com/unity-go/util"
)

const (
	VALUES_DB_PATH = "db/values-"
)

// unityNode structure
type UnityNode struct {
	NodeID                    NodeID                      // node's unique identity
	ValuesDB                  *db.DB                      // levelDb instance of node
	QuorumID                  int                         // quorm id of node
	Mutex                     sync.Mutex                  // Mutex for node
	Connections               map[string]*grpc.ClientConn // TCP connection
	WsConnection              *Conn                       // socket instance
	DHTContact                api.Contact                 // DHT node contact
	IsBatchReconstructedAtDHT map[string]bool             // map for batch id to check wether it is reconstructed or not
	MapMutex                  sync.RWMutex                // mutex for read and write
	CurrentElectionBlock      int                         // current election block number
}

type NodeBlock struct {
	BlockNumber         int
	NodeID              NodeID                        // node's unique identity
	Address             string                        // IP address
	OracleList          map[string]*api.Contacts      // Array of addresses of Oracles
	LeaderList          map[string]*api.Contacts      // Array Of addresses of Leaders
	RSIPList            map[string][]GroupList        // Array Of addresses of RSIPs
	TribeList           map[string]*api.Contacts      // Array Of addresses of Tirbes
	AllNodesInNetwork   []*api.Contact                // Array Of addresses of All nodes
	NodePositionInTribe int                           // Number of Tribe where node is
	NodePositionInRSIP  int                           // Number of RSIP group where node is
	LeadersPublicKey    map[string][]byte             // All leader groups public key
	OraclesPublicKey    map[string][]byte             // All oracle groups public key
	RSIPPublicKey       map[string][]GroupsPublicKeys // All RSIP groups public key
	IsOracle            bool                          // true if node is oracle
	IsLeader            bool                          // true if node is leader
	IsRSIP              bool                          // true if node is RSIP
	Timestamp           string                        // time when block is created
}

type GroupList struct {
	Group    string       // group number
	Contacts api.Contacts // list of contact
}

type GroupsPublicKeys struct {
	Group     string // group number
	PublicKey []byte // group public key
}

type Address struct {
	NodeID   NodeID
	Address  string
	QuorumID int
	IsOracle bool
}

// Return an instance of unity node
func NewUnityNode(self *Address) (*UnityNode, *NodeBlock) {
	newNode := &UnityNode{
		NodeID:                    self.NodeID,
		QuorumID:                  self.QuorumID,
		IsBatchReconstructedAtDHT: make(map[string]bool),
		Connections:               make(map[string]*grpc.ClientConn),
	}

	hexID := hex.EncodeToString(self.NodeID[:])
	if Debug == true {
		fmt.Println(hexID)
	}

	conn, err := db.OpenFile(VALUES_DB_PATH+hexID, nil)
	if err != nil {
		log.Println(err)
		panic("Unable to open values database")
	}

	// defer conn.Close()
	newNode.ValuesDB = conn

	nodeBlock := &NodeBlock{
		NodeID:            self.NodeID,
		Address:           self.Address,
		OracleList:        make(map[string]*api.Contacts),
		LeaderList:        make(map[string]*api.Contacts),
		TribeList:         make(map[string]*api.Contacts),
		RSIPList:          make(map[string][]GroupList),
		LeadersPublicKey:  make(map[string][]byte),
		OraclesPublicKey:  make(map[string][]byte),
		RSIPPublicKey:     make(map[string][]GroupsPublicKeys),
		AllNodesInNetwork: []*api.Contact{},
		Timestamp:         time.Now().String(),
	}
	return newNode, nodeBlock
}

// Dial function
func DialContact(contact api.Contact) (*rpc.Client, error) {
	connection, err := net.Dial("tcp", contact.Address)
	if err != nil {
		// connection, err = net.DialTimeout("tcp", contact.Aiyaj, 5*time.Second)
		fmt.Println("error", err)
		// return nil, err
	}

	return rpc.NewClient(connection), nil
}
