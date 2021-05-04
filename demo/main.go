package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	. "github.com/unity-go/util"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

func main() {
	MakeRequest()
}

func MakeRequest() {

	byteValue, _ := ioutil.ReadFile("demo/200k-dataset.json")
	transactions := []TransactionWithSignature{}
	// transactionsResult := []TransactionWithSignature{}
	transactionRequest := TransactionRequest{
		NumberOfBatches:                  10,
		NumberOfTransactionInSingleBatch: 2000,
	}

	if err := json.Unmarshal(byteValue, &transactions); err != nil {
		fmt.Println("proto.Unmarshal", err)
	}

	// bytesRepresentation, err := json.Marshal(message)
	// if err != nil {
	// 	log.Fatalln(err)
	// }

	fmt.Println("Here")

	start := time.Now()
	_, err := http.Post("http://13.58.225.215:6044/fill-leader-queue", "application/json", bytes.NewBuffer(byteValue))
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("RESPONSE took", time.Since(start))
	// json.NewDecoder(resp.Body).Decode(&transactionsResult)

	requestInBytes, _ := json.Marshal(transactionRequest)

	respFromTx, err := http.Post("http://13.58.225.215:6044/self-transaction-processor", "application/json", bytes.NewBuffer(requestInBytes))
	if err != nil {
		log.Fatalln(err)
	}

	// buf := new(bytes.Buffer)
	// buf.ReadFrom(stream)

	// buf.Bytes()

	// var result map[string]interface{}

	// response := ""

	fmt.Println("Got Response", respFromTx)
}
