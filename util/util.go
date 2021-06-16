package util

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	// "github.com/enriquebris/goconcurrentqueue"
	. "github.com/syndtr/goleveldb/leveldb/util"

	. "github.com/unity-go/go-bls"
	. "github.com/unity-go/mongodb"
)

// var (
// 	Queue goconcurrentqueue.Queue
// )

var (
	outfile, _ = os.OpenFile("testnet.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	Logger     = log.New(outfile, " ", log.LstdFlags)
	A          = os.Getenv("TIME")
)

// type ClientConnection struct {
// 	Address    string
// 	Connection *grpc.ClientConn
// }

type ChunkWithId struct {
	ID   int32
	Data []byte
}

type LogResponse struct {
	ID    string
	Value []string
}

type LogResponses struct {
	Time string
	ID   string
	Log  []LogResponse
}

var Mdb = Mongo{}

var MaxMsgSize = 1024 * 1024 * 32

const IDLength = 20

var Debug bool
var ShowTime bool

type NodeID []byte

type TransactionChunk []byte
type TransactionChunks []TransactionChunk

// BLS

var SystemData = System{}

type SecretShares struct {
	secret1 string
	secret2 string
}

// BytesPrefix returns key range that satisfy the given prefix.
// This only applicable for the standard 'bytes comparer'.
func BytesPrefix(prefix []byte) *Range {
	var limit []byte
	for i := len(prefix) - 1; i >= 0; i-- {
		c := prefix[i]
		if c < 0xff {
			limit = make([]byte, i+1)
			copy(limit, prefix)
			limit[i] = c + 1
			break
		}
	}
	return &Range{prefix, limit}
}

type TransactionWithSignature struct {
	ID                         string
	From, To, Message, BatchID string
	Amount                     int
	Signature                  []byte
	Status                     string
	Reason                     []string
	Leaders, Oracles, RSIP     VoteResponses
}

type TransactionWithSignatures []TransactionWithSignature

type TransactionWithSignature2 struct {
	From      string `json:"from,omitempty"`
	To        string `json:"to,omitempty"`
	Amount    int    `json:"amount,omitempty"`
	Signature string `json:"signature,omitempty"`
}

type TransactionWithoutSignature struct {
	From, To string `json: "from"`
	Amount   int
}

type MessageResponse struct {
	ID                  string
	Time                float64
	NumberOfTransaction int
}

type Contact struct {
	ID             NodeID
	Address        string
	QuorumID       int
	IsOracle       bool
	IsLeader       bool
	Oracle         NodeID
	PubKey         string
	Category       string
	SelectedNumber int
	SelectedTribe  int
}
type Contacts []Contact

type VoteResponse struct {
	ID              []byte `json:"id"`
	TransactionID   string `json:"txId"`
	Address         string `json:"address"`
	Vote            string `json:"vote, omitempty"`
	Reason          string `json:"reason, omitempty"`
	Threshold       string `json:"threshold, omitempty"`
	ThresholdOfRSIP string `json:"thresholdOfRSIP, omitempty"`
}

type VoteResponses []VoteResponse

type Client struct {
	Name       string `json:"name"`
	Address    string `json:"address"`
	PublicKey  string `json:"public"`
	PrivateKey string `json:"private"`
	Amount     int    `json:"amount"`
}

func Contains(a []string, x string) bool {
	for _, n := range a {
		if strings.Contains(n, x) {
			return true
		}
	}
	return false
}

func IsChunkContains(a []ChunkWithId, x ChunkWithId) bool {
	for _, n := range a {
		if x.ID == n.ID {
			return true
		}
	}
	return false
}

// func IsConnectionContains(a []ClientConnection, address string) *grpc.ClientConn {
// 	for _, n := range a {
// 		// fmt.Println(n)
// 		if n.Address == address {
// 			return n.Connection
// 		}
// 	}
// 	return nil
// }

type RSIPDecimal struct {
	Group   int
	Decimal int32
}
type RSIPDecimals []RSIPDecimal

type TransactionRequest struct {
	NumberOfBatches                  int32
	NumberOfTransactionInSingleBatch int32
}

type BatchTransaction struct {
	Data              []TransactionWithSignature
	Signature         string
	SignatureOfLeader string
	SignatureOfOracle string
	SignatureOfRSIP   []string
}

type ResponseType struct {
	Status           string
	Message          string
	Data             []string
	TotalTime        float64
	BatchSize        int
	NumberOfBatches  int
	TotalTransaction int
	TPS              float64
}

func NewNodeID(data string) []byte {
	decoded, _ := hex.DecodeString(data)
	ret := [20]byte{}

	for i := 0; i < IDLength; i++ {
		ret[i] = decoded[i]
	}

	// fmt.Println(ret[:20], "2")

	return ret[:20]
	// return
}

func NewRandomNodeID() []byte {
	ret := [20]byte{}
	buffer := make([]byte, IDLength)
	_, err := rand.Read(buffer)
	check(err)

	for i, b := range buffer {
		ret[i] = b
	}

	return ret[:20]
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func CheckErr(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

func Elapsed(what string) func() {
	start := time.Now()
	return func() {
		fmt.Printf("%s took %v\n", what, time.Since(start))
	}
}

func Timed(f func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		f(w, r)
		end := time.Now()
		fmt.Println("The request took", end.Sub(start))
	}
}

func Log(id string, a ...interface{}) {

	str := ""

	for index := 0; index < len(a); index++ {
		str1 := fmt.Sprintf("%v", a[index])
		str += " " + str1
	}

	Mdb.CreateLogs(id, str)

	// Logger.Println(a...)
}
