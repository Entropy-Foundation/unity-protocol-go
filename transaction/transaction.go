package transaction

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	// "sync"
	"time"

	. "github.com/dfinity/go-dfinity-crypto/groupsig"
	crand "github.com/dfinity/go-dfinity-crypto/rand"
	"github.com/go-chi/chi"
	"github.com/golang/protobuf/proto"
	guuid "github.com/google/uuid"
	"golang.org/x/net/websocket"

	"github.com/klauspost/reedsolomon"

	"github.com/unity-go/api"
	. "github.com/unity-go/electionProcess"
	. "github.com/unity-go/findValues"
	. "github.com/unity-go/unityCoreRPC"
	. "github.com/unity-go/unityNode"
	. "github.com/unity-go/util"
)

// Return list of clients stored on node
func HandleGetClients(w http.ResponseWriter, r *http.Request, node *UnityNode) {
	// w.Write([]byte("welcome"))
	clients := &api.Clients{}
	response, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(node.NodeID[:])+"_Clients"), nil)

	proto.Unmarshal(response, clients)

	data, _ := json.Marshal(clients)
	w.Write(data)
}

// Sync transaction to central Database
func SyncTransactionToCentralDB(w http.ResponseWriter, r *http.Request, node *UnityNode, selfContact api.Contact) {
	// w.Write([]byte("welcome"))

	numberOfTribes, _ := strconv.Atoi(os.Getenv("NUMBER_OF_TRIBES"))
	numberOfRSIPGroups, _ := strconv.Atoi(os.Getenv("NUMBER_OF_RSIP_GROUP"))

	fmt.Println(numberOfRSIPGroups, numberOfTribes)

	// var wg sync.WaitGroup

	for index := 0; index <= numberOfTribes; index++ {

		for index2 := 0; index2 <= numberOfRSIPGroups; index2++ {
			listOfRSIPNodes := GetValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"-Tribe_"+strconv.Itoa(index)+"_RSIP_GROUP_"+strconv.Itoa(index2)))
			// fmt.Println(len(chunkOfBatch.Data), "single chunk UI", len(listOfRSIPNodes.Contact))

			// 	// errorFromPut := PutValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_Node_Group_"+strconv.Itoa(i)), chunkOfBatchInByte)
			// 	// if errorFromPut != nil {
			// 	// 	fmt.Println(errorFromPut)
			// 	// }

			request := api.Request{
				Action: "STORE-TRANSACTION-DETAILS-TO-DHT",
			}

			for k := 0; k <= len(listOfRSIPNodes.Contact)-1; k++ {

				// wg.Add(1)

				BootstrapSync(selfContact, listOfRSIPNodes.Contact[k], request, nil, nil, node)
			}

			fmt.Println("OUTSUDE")

		}
	}
	// wg.Wait()

	w.Write([]byte("Finished"))
}

// Generate Dummy transactions
func FillLeaderQueue(node *UnityNode, transactionRequest api.TransactionWithSignatures, selfContact api.Contact, transactionType string, numberOfTransactions TransactionRequest, nodeBlock *NodeBlock, uuid string) {

	transactionsRequest := []TransactionWithSignature{}

	dataInJson, _ := json.Marshal(transactionRequest.TransactionWithSignature)

	numberOfBatch := numberOfTransactions.NumberOfBatches

	json.Unmarshal(dataInJson, &transactionsRequest)
	fmt.Println("TransactionType", transactionType, len(transactionsRequest))

	if transactionType == "auto" {
		divided := []TransactionWithSignature{}
		// defer r.Body.Close()
		// json.NewDecoder(r.Body).Decode(&transactionsRequest)

		listOfLeaders := api.Contacts{}
		listOfLeaders1 := append(nodeBlock.LeaderList["1"].Contact[:0:0], nodeBlock.LeaderList["1"].Contact...)
		listOfLeaders2 := append(nodeBlock.LeaderList["2"].Contact[:0:0], nodeBlock.LeaderList["2"].Contact...)

		// listOfLeaders2 := GetValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_Leaders_Epoch_1_Group_2"))

		listOfLeaders.Contact = append(listOfLeaders1, listOfLeaders2...)

		modulo := 0
		if numberOfBatch >= 10 {
			modulo = int(numberOfBatch % 10)
		}

		noOfLeaders := 0
		if numberOfBatch >= 10 {
			noOfLeaders = len(listOfLeaders.Contact)
		} else if numberOfBatch < 10 {
			noOfLeaders = int(numberOfBatch)
		}

		// transectionChunks := len(transactionsRequest) / noOfLeaders

		// for index := 0; index < len(transactionsRequest); index += transectionChunks {
		// 	end := index + transectionChunks

		// 	if end > len(transactionsRequest) {
		// 		end = len(transactionsRequest)
		// 	}

		// 	divided = append(divided, transactionsRequest[index:end])

		// }

		// transactionPerLeader := 0
		transactionChunkPerLeader := 0
		channel := make(chan []byte)
		transectionChunks := (len(transactionsRequest) - (modulo * int(numberOfTransactions.NumberOfTransactionInSingleBatch))) / noOfLeaders

		fmt.Println("Length of transaction request", len(transactionsRequest), "BEFORE")

		for j := 0; j < noOfLeaders; j++ {
			if modulo > 0 {
				modulo--
				transactionChunkPerLeader = transectionChunks + int(numberOfTransactions.NumberOfTransactionInSingleBatch)
			} else {
				transactionChunkPerLeader = transectionChunks
			}

			fmt.Println("TransactionChunk", transactionChunkPerLeader)
			for index := 0; index < len(transactionsRequest); index += transactionChunkPerLeader {
				end := index + transactionChunkPerLeader

				if end > len(transactionsRequest) {
					end = len(transactionsRequest)
				}

				divided = transactionsRequest[index:end]
				fmt.Println(index, end, len(divided), len(transactionsRequest))

				if j != noOfLeaders-1 {
					transactionsRequest = transactionsRequest[end:]
				}

				break

			}

			fmt.Println("Length of transaction request", len(transactionsRequest))

			// listOfOracles.Contact = append(listOfOracles.Contact[:j], listOfOracles.Contact[j+1:]...)

			// transactionQueueByte, _ := proto.Marshal(divided[j])
			transactionsRequestAPI := &api.TransactionWithSignatures{}

			for index := 0; index < len(divided); index++ {
				transactionsRequestForSingle := api.TransactionWithSignature{
					From:    divided[index].From,
					To:      divided[index].To,
					Amount:  int32(divided[index].Amount),
					Message: divided[index].Message,
					// Signature: divided[index].Signature,
					Signature: []byte{},
					Leaders:   &api.VoteResponses{},
					Oracles:   &api.VoteResponses{},
					RSIP:      &api.VoteResponses{},
				}
				transactionsRequestAPI.TransactionWithSignature = append(transactionsRequestAPI.TransactionWithSignature, &transactionsRequestForSingle)
			}
			transactionQueueByte, _ := proto.Marshal(transactionsRequestAPI)

			request := api.Request{
				Action: "store-data-in-leaders",
				Data:   transactionQueueByte,
				Param:  uuid,
			}

			go BootstrapSync(selfContact, listOfLeaders.Contact[j], request, channel, nil, node)
			transactionQueueByte = nil
		}

	Outer:
		for {
			response := <-channel
			fmt.Println("response", response)
			noOfLeaders--
			if noOfLeaders == 0 {
				listOfLeaders = api.Contacts{}
				// listOfLeaders1 = api.Contacts{}
				// listOfLeaders2 = api.Contacts{}
				divided = nil

				break Outer
			}

		}

	} else {
		transactionsQueueFromDB := &api.TransactionWithSignatures{}

		for index := 0; index < len(transactionsRequest); index++ {
			singleTransaction := &api.TransactionWithSignature{
				From:    transactionsRequest[index].From,
				To:      transactionsRequest[index].To,
				Amount:  int32(transactionsRequest[index].Amount),
				Message: transactionsRequest[index].Message,
				// Signature: transactionsRequest[index].Signature,
				Signature: []byte{},

				Leaders: &api.VoteResponses{},
				Oracles: &api.VoteResponses{},
				RSIP:    &api.VoteResponses{},
			}

			transactionsQueueFromDB.TransactionWithSignature = append(transactionsQueueFromDB.TransactionWithSignature, singleTransaction)
		}

		transactionQueueByte, _ := proto.Marshal(transactionsQueueFromDB)

		node.ValuesDB.Put([]byte(hex.EncodeToString(node.NodeID[:])+"_tx_queue"), transactionQueueByte, nil)
		transactionsQueueFromDB = nil
		transactionQueueByte = nil

	}
	dataInJson = nil

	transactionsRequest = nil

}

// Send data to all leader
func SelfCallingTxProcessor(w http.ResponseWriter, r *http.Request, node *UnityNode, selfContact api.Contact, nodeBlock *NodeBlock) {

	transactionType := r.URL.Query().Get("type")
	transactionToken := r.URL.Query().Get("transactionToken")

	start1 := time.Now()

	if transactionType == "auto" {
		listOfLeaders := api.Contacts{}
		listOfLeaders.Contact = append(nodeBlock.LeaderList["1"].Contact[:0:0], nodeBlock.LeaderList["1"].Contact...)

		IDData := listOfLeaders.Contact[0].ID
		Add := listOfLeaders.Contact[0].Address
		Qid := int32(listOfLeaders.Contact[0].QuorumID)

		selfContact = api.Contact{
			ID:       IDData,
			Address:  Add,
			QuorumID: Qid,
		}
		listOfLeaders = api.Contacts{}
	}

	Log("Fetch Leaders From Database", time.Since(start1))

	err := os.Truncate("testnet.log", 0)
	if err != nil {
		log.Fatal(err)
	}

	start := time.Now()
	batchChannel := make(chan []byte)

	transactionsRequest := TransactionRequest{}
	transactionWithSignature := TransactionWithSignature{}

	var modulo int
	var noOfLeader int
	defer r.Body.Close()

	if transactionType == "auto" {
		json.NewDecoder(r.Body).Decode(&transactionsRequest)

		if transactionsRequest.NumberOfBatches >= 10 {
			modulo = int(transactionsRequest.NumberOfBatches) % 10
			noOfLeader = 5
		} else if transactionsRequest.NumberOfBatches < 10 && transactionsRequest.NumberOfBatches > 5 {
			noOfLeader = int(transactionsRequest.NumberOfBatches)
		} else if transactionsRequest.NumberOfBatches <= 5 {
			noOfLeader = int(transactionsRequest.NumberOfBatches)
		} else {
			noOfLeader = 5
		}
	} else {
		json.NewDecoder(r.Body).Decode(&transactionWithSignature)
		if transactionWithSignature.Amount == 0 || transactionWithSignature.From == "" || transactionWithSignature.To == "" {
			fmt.Println("Here")
			w.Write([]byte("Enter valid transaction details"))
			return
		}
	}
	for i := 1; i < 3; i++ {
		noOfLeadersToSend := 0
		if transactionType == "auto" {
			if noOfLeader >= 5 {
				noOfLeadersToSend = 5
			} else {
				noOfLeadersToSend = noOfLeader
			}
		}
		go Job(node, selfContact, batchChannel, i, transactionsRequest, transactionType, modulo, noOfLeadersToSend, nodeBlock, transactionWithSignature, transactionToken)

		if transactionsRequest.NumberOfBatches < 10 && transactionsRequest.NumberOfBatches > 5 && transactionType == "auto" {
			noOfLeader = noOfLeader - 5
		}

		if transactionsRequest.NumberOfBatches >= 10 && transactionType == "auto" {
			if modulo > 5 {
				modulo = modulo - 5
			} else {
				modulo = 0
			}
		}

		if transactionType == "manual" || transactionsRequest.NumberOfBatches <= 5 {
			time.Sleep(500 * time.Millisecond)
			break
		}
	}

	noOfLeaders := 0

	if transactionType == "manual" || transactionsRequest.NumberOfBatches <= 5 {
		noOfLeaders = 1
	} else {
		noOfLeaders = 2
	}

	jobResponses := api.BatchTransactions{}

OuterSecond:
	for {
		response := <-batchChannel
		fmt.Println("SINGLE JOB RESPONSE")
		noOfLeaders--
		voteResponse := api.BatchTransactions{}
		proto.Unmarshal(response, &voteResponse)

		// jobResponses = voteResponse
		jobResponses.BatchTransaction = append(jobResponses.BatchTransaction, voteResponse.BatchTransaction...)
		if noOfLeaders == 0 {
			break OuterSecond
		}
	}

	batchIds := []string{}

	for index := 0; index < len(jobResponses.BatchTransaction); index++ {
		batchIds = append(batchIds, jobResponses.BatchTransaction[index].ID)
	}

	response := ResponseType{
		Status:  "Success",
		Message: "Transaction executed successfully",
		Data:    batchIds,
	}

	if transactionType == "auto" {
		response.BatchSize = int(transactionsRequest.NumberOfTransactionInSingleBatch)
		response.NumberOfBatches = int(transactionsRequest.NumberOfBatches)
		response.TotalTime = (float64(time.Since(start) / time.Millisecond)) / 1000
		response.TotalTransaction = int(transactionsRequest.NumberOfTransactionInSingleBatch * transactionsRequest.NumberOfBatches)
		response.TPS = (float64(transactionsRequest.NumberOfTransactionInSingleBatch*transactionsRequest.NumberOfBatches) / (float64(time.Since(start)/time.Millisecond) / 1000))
	}

	transactionsRequest = TransactionRequest{}
	jobResponses = api.BatchTransactions{}
	batchIds = nil
	fmt.Printf("%s took %v\n", "Finish Whole Scheduler transaction----------------------------------------------------", time.Since(start))
	Log("Finish Whole Scheduler transaction took----------------------------------------------------", time.Since(start))
	responseInBytes, _ := json.Marshal(&response)

	// go func() {
	// 	for index := 0; index < 1000; index++ {
	// 		runtime.GC()
	// 	}
	// }()

	// proto.Un

	w.Write(responseInBytes)
}

// Start point of batch transaction
func Job(node *UnityNode, selfContact api.Contact, batchChannel chan []byte, k int, transactionsRequest TransactionRequest, transactionType string, modulo int, noOfLeaderFromPre int, nodeBlock *NodeBlock, transactionWithSignature TransactionWithSignature, transactionToken string) []BatchTransaction {
	// get laders list
	listOfLeaders := api.Contacts{}
	listOfLeaders.Contact = append(nodeBlock.LeaderList[strconv.Itoa(k)].Contact[:0:0], nodeBlock.LeaderList[strconv.Itoa(k)].Contact...)

	// make channel
	jobChannel := make(chan []byte)
	var noOfBatches int32
	var noOfLeader int
	var newNumberOfBatches int32
	var numberOfTransactionInSingleBatch int32
	singleTransaction := api.TransactionWithSignature{}

	// number of batches for single leader to proceed
	if transactionType == "auto" {
		if transactionsRequest.NumberOfBatches > 10 {
			noOfBatches = int32(int(transactionsRequest.NumberOfBatches)-int(modulo)) / 10
			noOfLeader = 5
		} else if transactionsRequest.NumberOfBatches < 10 && transactionsRequest.NumberOfBatches > 5 {
			newNumberOfBatches = 1
			noOfLeader = noOfLeaderFromPre
		} else if transactionsRequest.NumberOfBatches == 10 {
			newNumberOfBatches = 1
			noOfLeader = 5
		} else {
			newNumberOfBatches = 1
			noOfLeader = noOfLeaderFromPre
		}
		numberOfTransactionInSingleBatch = transactionsRequest.NumberOfTransactionInSingleBatch
		fmt.Println("EARLIER", numberOfTransactionInSingleBatch)
	} else {
		singleTransaction.From = transactionWithSignature.From
		singleTransaction.To = transactionWithSignature.To
		singleTransaction.Amount = int32(transactionWithSignature.Amount)
		singleTransaction.Message = transactionWithSignature.Message
		singleTransaction.Leaders = &api.VoteResponses{}
		singleTransaction.Oracles = &api.VoteResponses{}
		singleTransaction.RSIP = &api.VoteResponses{}
	}

	if transactionType == "manual" {
		noOfLeader = 1
	}
	for i := 0; i < noOfLeader; i++ {

		if transactionType == "auto" {
			if modulo > 0 {
				modulo--
				newNumberOfBatches = noOfBatches + 1
			} else {
				newNumberOfBatches = noOfBatches
			}
		}

		request := api.Request{
			Action:                           "Start-Queued-Batch",
			Param:                            strconv.Itoa(k),
			Value:                            []byte(strconv.Itoa(i * 100)),
			NumberOfBatches:                  newNumberOfBatches,
			NumberOfTransactionInSingleBatch: numberOfTransactionInSingleBatch,
			TransactionType:                  transactionType,
			SingleTransaction:                &singleTransaction,
			TransactionToken:                 transactionToken,
		}

		fmt.Println("NUMBER OF BATCHES", i, "->", newNumberOfBatches)

		go BootstrapSync(selfContact, listOfLeaders.Contact[i], request, jobChannel, nil, node)

		if transactionType == "auto" {
			timeInMilisecond := i * 100

			if i != 4 {
				fmt.Println("timeInMilisecond", timeInMilisecond)
				time.Sleep(time.Duration(timeInMilisecond) * time.Millisecond)
				fmt.Println("Time DUration", time.Duration(timeInMilisecond)*time.Millisecond)
			}
		} else {
			break
		}

	}

	noOfLeaders := 0
	if transactionType == "manual" {
		noOfLeaders = 1
	} else {
		if transactionsRequest.NumberOfBatches > 10 {
			noOfLeaders = 5
		} else if transactionsRequest.NumberOfBatches < 10 && transactionsRequest.NumberOfBatches > 5 {
			noOfLeaders = int(noOfLeaderFromPre)
		} else {
			noOfLeaders = int(noOfLeaderFromPre)

		}
	}
	jobResponses := api.BatchTransactions{}
OuterSeven:
	for {
		response := <-jobChannel

		noOfLeaders--
		voteResponse := api.BatchTransactions{}
		proto.Unmarshal(response, &voteResponse)
		jobResponses.BatchTransaction = append(jobResponses.BatchTransaction, voteResponse.BatchTransaction...)

		if Debug == true {
			fmt.Println("length of single job", len(jobResponses.BatchTransaction), "noOfLeaders", noOfLeaders)
		}
		if noOfLeaders == 0 {
			break OuterSeven
		}
	}

	listOfLeaders = api.Contacts{}

	data, _ := proto.Marshal(&jobResponses)
	jobResponses = api.BatchTransactions{}
	batchChannel <- data
	return nil
}

// To check socket hello
func Echo(ws *websocket.Conn) {
	var err error

	for {
		var reply string

		if err = websocket.Message.Receive(ws, &reply); err != nil {
			fmt.Println("Can't receive")
			break
		}

		fmt.Println("Received back from client: " + reply)

		msg := "Received:  " + reply
		fmt.Println("Sending to client: " + msg)

		if err = websocket.Message.Send(ws, msg); err != nil {
			fmt.Println("Can't send")
			break
		}
	}
}

// Generate dummy transaction
func CreateData(w http.ResponseWriter, r *http.Request, node *UnityNode, selfContact api.Contact, nodeBlock *NodeBlock) {

	// params := chi.URLParam(r, "id")
	// no, _ := strconv.Atoi(params)

	transactionType := r.URL.Query().Get("type")

	transactionsRequest := TransactionRequest{}
	transections := api.TransactionWithSignatures{}

	defer r.Body.Close()

	if transactionType == "auto" {

		json.NewDecoder(r.Body).Decode(&transactionsRequest)

		no := transactionsRequest.NumberOfBatches * transactionsRequest.NumberOfTransactionInSingleBatch

		clients := api.Clients{}

		response, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(node.NodeID[:])+"_Clients"), nil)

		proto.Unmarshal(response, &clients)

		noOfData := len(clients.Client)

		for index := 0; index < int(no); index++ {

			minNo := 0
			maxNo := noOfData
			selectRandNumber1 := rand.Intn(maxNo-minNo) + minNo
			selectRandNumber2 := rand.Intn(maxNo-minNo) + minNo

			if selectRandNumber1 == selectRandNumber2 {
				selectRandNumber1 = rand.Intn(maxNo-minNo) + minNo
				selectRandNumber2 = rand.Intn(maxNo-minNo) + minNo
			}

			selectRandNumberAmount := rand.Intn(100-1) + 1
			// secret := &Seckey{}

			// secret.SetHexString(clients.Client[selectRandNumber1].PrivateKey)

			fromData := clients.Client[selectRandNumber1].Address
			toData := clients.Client[selectRandNumber2].Address
			amount := selectRandNumberAmount

			transactionRequest := api.TransactionWithSignature{Amount: int32(amount), From: fromData, To: toData}

			// transactionRequestInByte, _ := proto.Marshal(&transactionRequest)
			// sign := Sign(*secret, transactionRequestInByte)
			transactionRequest.Signature = []byte{}

			transections.TransactionWithSignature = append(transections.TransactionWithSignature, &transactionRequest)

		}
		fmt.Println("transections", len(transections.TransactionWithSignature))

	} else {
		transactionRequest := []TransactionWithSignature{}
		e := json.NewDecoder(r.Body).Decode(&transactionRequest)
		fmt.Println(e)

		fmt.Println("transections in manual", transactionRequest)

		for index := 0; index < len(transactionRequest); index++ {
			transactionRequestForAPI := api.TransactionWithSignature{Amount: int32(transactionRequest[index].Amount), From: transactionRequest[index].From, To: transactionRequest[index].To, Signature: []byte{}, Message: transactionRequest[index].Message}
			transections.TransactionWithSignature = append(transections.TransactionWithSignature, &transactionRequestForAPI)
		}

	}

	uuid := guuid.New()

	uuidData := uuid.String()

	FillLeaderQueue(node, transections, selfContact, transactionType, transactionsRequest, nodeBlock, uuidData)

	stringData := []string{}
	stringData = append(stringData, uuidData)

	transactionsRequest = TransactionRequest{}
	transections = api.TransactionWithSignatures{}
	msg := ResponseType{
		Status:  "Success",
		Message: "Transactions are generated and stored successfully",
		Data:    stringData,
	}
	res, _ := json.Marshal(&msg)

	w.Write(res)

}

// get value by key
func HandleGetValue(w http.ResponseWriter, r *http.Request, node *UnityNode) {
	// w.Write([]byte("welcome"))
	params := chi.URLParam(r, "id")
	if Debug == true {
		fmt.Println(params)
	}

	response, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(node.NodeID[:])+params), nil)

	str := hex.EncodeToString(response)
	fmt.Println((str))
	fmt.Println((response))

	w.Write(response)

	// if strings.Contains(params, "Decimal_DKG") {

	// 	apiRSIPDecimalData := &api.RSIPDecimals{}
	// 	utilRSIPDecimal := RSIPDecimals{}

	// 	proto.Unmarshal(response, apiRSIPDecimalData)

	// 	for index := 0; index < len(apiRSIPDecimalData.RSIPDecimal); index++ {

	// 		newData := RSIPDecimal{
	// 			Group:   int(apiRSIPDecimalData.RSIPDecimal[index].Group),
	// 			Decimal: apiRSIPDecimalData.RSIPDecimal[index].Decimal,
	// 		}

	// 		utilRSIPDecimal = append(utilRSIPDecimal, newData)

	// 	}

	// 	out, _ := json.Marshal(utilRSIPDecimal)
	// 	w.Write(out)

	// } else if strings.Contains(params, "_tx_queue") {
	// 	transactionWithoutSignature := &api.TransactionWithSignatures{}
	// 	batchTransaction := BatchTransaction{}

	// 	proto.Unmarshal(response, transactionWithoutSignature)

	// 	for index := 0; index < len(transactionWithoutSignature.TransactionWithSignature); index++ {

	// 		newData := TransactionWithSignature{
	// 			From:   transactionWithoutSignature.TransactionWithSignature[index].From,
	// 			To:     transactionWithoutSignature.TransactionWithSignature[index].To,
	// 			Amount: int(transactionWithoutSignature.TransactionWithSignature[index].Amount),
	// 			// Signature: transactionWithoutSignature.TransactionWithSignature[index].Signature,
	// 		}

	// 		batchTransaction.Data = append(batchTransaction.Data, newData)

	// 	}
	// 	out, _ := json.Marshal(batchTransaction.Data)
	// 	w.Write(out)

	// } else if strings.Contains(params, "dkg") {
	// 	w.Write(response)
	// } else {
	// 	transactionsRequestContactAPI := &api.Contacts{}
	// 	transactionsRequestContact := Contacts{}

	// 	proto.Unmarshal(response, transactionsRequestContactAPI)

	// 	for index := 0; index < len(transactionsRequestContactAPI.Contact); index++ {
	// 		contact := Contact{
	// 			Address:  transactionsRequestContactAPI.Contact[index].Address,
	// 			ID:       transactionsRequestContactAPI.Contact[index].ID,
	// 			IsLeader: transactionsRequestContactAPI.Contact[index].IsLeader,
	// 			IsOracle: transactionsRequestContactAPI.Contact[index].IsOracle,
	// 			Category: transactionsRequestContactAPI.Contact[index].Category,
	// 		}

	// 		transactionsRequestContact = append(transactionsRequestContact, contact)
	// 	}

	// 	out, _ := json.Marshal(transactionsRequestContact)

	// 	w.Write(out)
	// }

}

// Handle election proccess
func HandleElection(w http.ResponseWriter, r *http.Request, node *UnityNode, selfContact api.Contact, nodeBlock *NodeBlock) {
	Scheduler(node, selfContact, nodeBlock)
	w.Write([]byte("Election Proccess Initiated..."))
}

// create client on node
func CreateClient(w http.ResponseWriter, req *http.Request, node *UnityNode, selfContact api.Contact, nodeBlock *NodeBlock) {
	client := api.Client{}
	clients := api.Clients{}
	json.NewDecoder(req.Body).Decode(&client)

	// add Address, Public & Private key to incoming client
	newclient := AddRSAKeys()

	client.Address = newclient.Address
	client.PublicKey = newclient.PublicKey
	client.PrivateKey = newclient.PrivateKey

	clients = readContact()

	clients.Client = append(clients.Client, &client)

	defer writeContact(clients, node, selfContact, nodeBlock)

	json.NewEncoder(w).Encode(client)
}

func CreatePKI(w http.ResponseWriter, req *http.Request, node *UnityNode, selfContact api.Contact) {
	client := api.Client{}

	// add Address, Public & Private key to incoming client

	client = AddRSAKeys()
	data, _ := json.Marshal(client)
	w.Write(data)
}

// read contact from node
func readContact() (clients api.Clients) {

	byteValue, _ := ioutil.ReadFile("seed.json")
	json.Unmarshal([]byte(byteValue), &clients)

	return clients
}

// create public/private keys
func AddRSAKeys() api.Client {
	newClient := api.Client{}

	randomNumber := crand.NewRand()
	fmt.Println("Here", randomNumber)
	sec := NewSeckeyFromRand(randomNumber.Deri(1))

	pub := NewPubkeyFromSeckey(*sec)
	addr := GetAddressFromPublicKey(pub)

	newClient.Address = hex.EncodeToString(addr[:])
	newClient.PublicKey = sec.GetHexString()
	newClient.PrivateKey = pub.GetHexString()

	return newClient
}

func GetAddressFromPublicKey(pubKey *Pubkey) [20]byte {
	bytes := pubKey.Serialize()
	address := [20]byte{}
	hash := sha256.Sum256(bytes)
	copy(address[:], hash[12:])
	return address
}

// store list of client on node
func writeContact(clients api.Clients, node *UnityNode, self api.Contact, nodeBlock *NodeBlock) {

	clientsToStore, _ := proto.Marshal(&clients)

	err := PutValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_Clients"), clientsToStore)
	if err != nil {
		return
	}

	request := api.Request{
		Action: "Store-Updated-Client-Across-The-Network",
		Data:   clientsToStore,
	}
	// contactsFromOwnDb := GetValuesById(node, []byte(string(self.ID[:])))
	contactsFromOwnDb := &api.Contacts{}
	contactsFromOwnDb.Contact = append(nodeBlock.AllNodesInNetwork[:0:0], nodeBlock.AllNodesInNetwork...)

	for i := 0; i < len(contactsFromOwnDb.Contact); i++ {
		BootstrapSync(self, contactsFromOwnDb.Contact[i], request, nil, nil, node)
	}

	file, _ := json.MarshalIndent(clients, "", "    ")
	ioutil.WriteFile("seed.json", file, 0644)
}

// Return test net log
func TestnetLog(w http.ResponseWriter, r *http.Request, node *UnityNode, selfContact api.Contact) {
	var batchIds []string
	// multipleBatch := []string{}
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&batchIds); err != nil {
		return
	}

	// data := []LogStruct{}
	data := []LogResponse{}

	// for index := 0; index < len(batchIds); index++ {
	d := Mdb.GetLogs(batchIds)

	// Value: d,
	for index := 0; index < len(batchIds); index++ {
		single := LogResponse{
			ID: batchIds[index],
		}
		for index2 := 0; index2 < len(d); index2++ {
			if single.ID == d[index2].BatchID {
				if len(single.Value) > 0 {
					// fmt.Println(d[index2].Value)
					str1 := strings.Split(d[index2].Value, " ")

					// fmt.Println(len(single.Value))
					// for index3 := 0; index3 < len(single.Value); index3++ {
					// str2 := strings.Split(single.Value[index3], " ")

					// fmt.Println(strings.Join(str1[0 : len(str1)-1][:], " ") == strings.Join(str2[0 : len(str2)-1][:], " "))
					if !Contains(single.Value, strings.Join(str1[0 : len(str1)-3][:], " ")) {
						single.Value = append(single.Value, d[index2].Value)
					}
					// }
				} else {
					single.Value = append(single.Value, d[index2].Value)
				}
			}
		}
		data = append(data, single)
	}

	// single.Value = append(single.Value)

	// data = append(data, d...)
	// }

	// data := LogResponses{}

	// for i := 0; i < len(batchIds); i++ {
	// 	single := LogResponse{}
	// 	single.ID = batchIds[i]
	// 	data.Log = append(data.Log, single)
	// }

	// file, err := os.Open("testnet.log")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// scanner := bufio.NewScanner(file)

	// for scanner.Scan() {
	// 	for j := 0; j < len(data.Log); j++ {
	// 		if strings.Contains(scanner.Text(), data.Log[j].ID) {

	// 			text := strings.Replace(scanner.Text(), data.Log[j].ID, "-->", 1)

	// 			data.Log[j].Value = append(data.Log[j].Value, text)
	// 		} else {

	// 			data.Time = scanner.Text()
	// 		}
	// 	}
	// }

	response, _ := json.MarshalIndent(data, "", "    ")

	w.Write(response)
}

// Return transaction history
func TransactionHistory(w http.ResponseWriter, r *http.Request, node *UnityNode, selfContact api.Contact) {

	fmt.Println("GET params were:", r.URL.Query())

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	batchID := r.URL.Query().Get("batchID")

	// from := chi.URLParam(r, "from")
	// to := chi.URLParam(r, "to")

	result := Mdb.RetrieveTransactionHistory(from, to, batchID)

	response, _ := json.MarshalIndent(result, "", "    ")

	w.Write(response)
}

// Return transactions by id
func TransactionsByBatchIds(w http.ResponseWriter, r *http.Request, node *UnityNode, selfContact api.Contact) {

	batchIds := []string{}
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&batchIds); err != nil {
		return
	}
	// response, _ := proto.Marshal(&transactions)

	batchWiseResponse := api.BatchTransactions{}

	result := Mdb.RetrieveTransactionByBatchIds(batchIds)

	for index := 0; index < len(batchIds); index++ {
		batch := api.BatchTransaction{
			ID: batchIds[index],
		}
		for i := 0; i < len(result); i++ {
			if batchIds[index] == result[i].BatchID {
				batch.Data = append(batch.Data, &result[i])
			}
		}

		batchWiseResponse.BatchTransaction = append(batchWiseResponse.BatchTransaction, &batch)

	}

	response, _ := json.MarshalIndent(batchWiseResponse, "", "    ")

	w.Write(response)
}

// For Erasure
func MultipleTxRequests(w http.ResponseWriter, r *http.Request, node *UnityNode, selfContact api.Contact) {

	transactions := api.TransactionWithSignatures{}
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&transactions); err != nil {
		return
	}
	response, _ := proto.Marshal(&transactions)

	// get all 5 leaders
	listOfLeaders := GetValuesById(node, []byte(hex.EncodeToString(selfContact.ID[:])+"_Leaders_Epoch_1_Group_1"))

	// Erasure Coding //

	// 1 :- create an encoder with 4 data shards (where your data goes) and 1 parity shards (calculated)
	enc, err := reedsolomon.New(3, 1)
	checkErr(err)
	if Debug == true {
		fmt.Printf("Step-1: %t", enc)
	}

	// 2 Split original data into shards
	shards, _ := enc.Split(response)
	// fmt.Println("Step-2:", shards)

	// 3 Encode the shards
	err = enc.Encode(shards)
	checkErr(err)
	// fmt.Println("Step-3:", shards)

	// 4 Verify
	ok, err := enc.Verify(shards)
	checkErr(err)
	if Debug == true {
		fmt.Println("Step-4:", ok)
	}

	// 5 delete some client data
	// shards[0] = nil
	shards[2] = nil
	if Debug == true {
		fmt.Println("Step-5:", shards)
	}

	// 6 Reconstruct just the missing data shards
	err = enc.ReconstructData(shards)
	checkErr(err)

	if Debug == true {
		for i := 0; i < len(shards); i++ {
			fmt.Println("\n")
			fmt.Println("##############################", i, "->", shards[i], "#######################################")
		}
	}

	file, err := os.Create("data.json")
	// src := io.MultiReader(out)
	// err = enc.Join(file, shards, len(shards))

	if Debug == true {
		fmt.Println("-----------", file, selfContact, "-------------")
		fmt.Println("\n")
	}

	DistributeChunks(selfContact, shards, listOfLeaders)

	ReconstructTxFromChunks(selfContact, node, listOfLeaders)

	original, _ := json.Marshal(file)

	w.Write(original)
}

// For Erasure
func DistributeChunks(selfContact api.Contact, shards [][]byte, listOfLeaders api.Contacts) {

	// shardsInBytes, _ := proto.Marshal(shards)
	if Debug == true {
		fmt.Println(listOfLeaders)
	}

	// Distribute Tx Chunks
	index := 0
	for i := 0; i < len(listOfLeaders.Contact); i++ {
		shardsInBytes := shards[index]

		if listOfLeaders.Contact[i].Address != selfContact.Address {
			if Debug == true {
				fmt.Println("\n-------------------------------", "ROUND", i, "------------------------------------------")
			}

			res, err := BatchingOfTx(selfContact, listOfLeaders.Contact[i], "Distribute-shards-among-leaders", shardsInBytes)
			if Debug == true {
				fmt.Println("\n", "Response->", i, res, "<-------->", err)
			}

			index++
		}
	}

	// Circulation of Tx Chunks
	for i := 0; i < len(listOfLeaders.Contact); i++ {
		if listOfLeaders.Contact[i].Address != selfContact.Address {
			if Debug == true {
				fmt.Println("\n-------------------------------", "CIRCULATE-SHARDS", i, "------------------------------------------")
			}

			res, _ := BatchingOfTx(selfContact, listOfLeaders.Contact[i], "Circulate-shards-among-leaders", nil)
			if Debug == true {
				fmt.Println(res)
			}

		}
	}
}

// For Erasure
func ReconstructTxFromChunks(selfContact api.Contact, node *UnityNode, listOfLeaders api.Contacts) {
	// retrieve all leaders shards
	for i := 0; i < len(listOfLeaders.Contact); i++ {
		res, _ := BatchingOfTx(selfContact, listOfLeaders.Contact[i], "Recover-transaction-among-leaders", nil)

		transactions := api.TransactionWithSignatures{}
		proto.Unmarshal(res, &transactions)
		if Debug == true {
			fmt.Println(transactions, "-----------------recovered\n")
		}

	}
}

// check error and print
func checkErr(err error) {
	if err != nil {
		fmt.Println(err)
	}
}
