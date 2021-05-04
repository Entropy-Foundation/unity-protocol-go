package unityNode

import (
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"sync"

	db "github.com/syndtr/goleveldb/leveldb"
	"github.com/unity-go/api"
	. "github.com/unity-go/util"
)

const (
	VALUES_DB_PATH = "db/values-"
)

// unityNode structure
type UnityNode struct {
	NodeID   NodeID
	ValuesDB *db.DB
	QuorumID int
	Mutex    sync.Mutex
}

type Address struct {
	NodeID   NodeID
	Address  string
	QuorumID int
	IsOracle bool
}

func NewUnityNode(self *Address) *UnityNode {
	newNode := &UnityNode{
		NodeID:   self.NodeID,
		QuorumID: self.QuorumID,
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
	return newNode
}

func DialContact(contact api.Contact) (*rpc.Client, error) {
	connection, err := net.Dial("tcp", contact.Address)
	if err != nil {
		// connection, err = net.DialTimeout("tcp", contact.Aiyaj, 5*time.Second)
		fmt.Println("error", err)
		// return nil, err
	}

	return rpc.NewClient(connection), nil
}
