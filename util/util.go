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

	. "github.com/unity-go/mongodb"
)

// var (
// 	Queue goconcurrentqueue.Queue
// )

var (
	outfile, _ = os.OpenFile("testnet.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	Logger     = log.New(outfile, " ", log.LstdFlags)
)

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

var MaxMsgSize = 1024 * 1024 * 1024

const IDLength = 20

var Debug bool
var ShowTime bool

type NodeID []byte

type TransactionChunk []byte
type TransactionChunks []TransactionChunk

type SecretShares struct {
	secret1 string
	secret2 string
}

type TransactionWithSignature struct {
	ID                     string
	From, To, Message      string
	Amount                 int
	Signature              string
	Status                 string
	Reason                 []string
	Leaders, Oracles, RSIP VoteResponses
}

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

type RSIPDecimal struct {
	Group   int
	Decimal string
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

func Contains(a []string, x string) bool {
	for _, n := range a {
		if strings.Contains(n, x) {
			return true
		}
	}
	return false
}
