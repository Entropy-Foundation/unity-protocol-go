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
	"time"

	. "github.com/dfinity/go-dfinity-crypto/groupsig"
	crand "github.com/dfinity/go-dfinity-crypto/rand"
	"github.com/go-chi/chi"
	"github.com/golang/protobuf/proto"

	guuid "github.com/google/uuid"
	"github.com/unity-go/api"
	. "github.com/unity-go/electionProcess"
	. "github.com/unity-go/findValues"
	. "github.com/unity-go/unityCoreRPC"
	. "github.com/unity-go/unityNode"
	. "github.com/unity-go/util"
)

func HandleGetClients(w http.ResponseWriter, r *http.Request, node *UnityNode) {
	// w.Write([]byte("welcome"))
	clients := &api.Clients{}
	response, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(node.NodeID[:])+"_Clients"), nil)

	proto.Unmarshal(response, clients)

	data, _ := json.Marshal(clients)
	w.Write(data)
}

func FillLeaderQueue(node *UnityNode, transactionRequest api.TransactionWithSignatures, selfContact api.Contact, transactionType string, numberOfTransactions TransactionRequest, uuid string) {

	transactionsRequest := []TransactionWithSignature{}

	dataInJson, _ := json.Marshal(transactionRequest.TransactionWithSignature)

	numberOfBatch := numberOfTransactions.NumberOfBatches

	json.Unmarshal(dataInJson, &transactionsRequest)
	fmt.Println("TransactionType", transactionType, len(transactionsRequest))

	listOfLeaders := api.Contacts{}
	listOfLeaders1 := GetValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_Leaders_Epoch_1_Group_1"))
	listOfLeaders2 := GetValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_Leaders_Epoch_1_Group_2"))

	listOfLeaders.Contact = append(listOfLeaders1.Contact, listOfLeaders2.Contact...)

	if transactionType == "auto" {
		divided := []TransactionWithSignature{}
		// defer r.Body.Close()
		// json.NewDecoder(r.Body).Decode(&transactionsRequest)

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
					From:      divided[index].From,
					To:        divided[index].To,
					Amount:    int32(divided[index].Amount),
					Message:   divided[index].Message,
					Signature: divided[index].Signature,
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
			}

			go BootstrapSync(selfContact, listOfLeaders.Contact[j], request, channel, nil)
		}

		for {
			select {
			case response := <-channel:
				fmt.Println("response", response)
				noOfLeaders--
			}
			if noOfLeaders == 0 {
				break
			}
		}

	} else {
		transactionsQueueFromDB := &api.TransactionWithSignatures{}

		for index := 0; index < len(transactionsRequest); index++ {
			singleTransaction := &api.TransactionWithSignature{
				From:      transactionsRequest[index].From,
				To:        transactionsRequest[index].To,
				Amount:    int32(transactionsRequest[index].Amount),
				Message:   transactionsRequest[index].Message,
				Signature: transactionsRequest[index].Signature,
				Leaders:   &api.VoteResponses{},
				Oracles:   &api.VoteResponses{},
				RSIP:      &api.VoteResponses{},
			}

			transactionsQueueFromDB.TransactionWithSignature = append(transactionsQueueFromDB.TransactionWithSignature, singleTransaction)
		}

		transactionQueueByte, _ := proto.Marshal(transactionsQueueFromDB)

		// node.ValuesDB.Put([]byte(hex.EncodeToString(node.NodeID[:])+"_tx_queue"), transactionQueueByte, nil)

		channel := make(chan []byte)
		noOfLeaders := 1
		request := api.Request{
			Action:          "store-data-in-leaders",
			Data:            transactionQueueByte,
			Param:           uuid,
			TransactionType: transactionType,
		}

		go BootstrapSync(selfContact, listOfLeaders.Contact[0], request, channel, nil)

		for {
			select {
			case response := <-channel:
				fmt.Println("response", response)
				noOfLeaders--
			}
			if noOfLeaders == 0 {
				break
			}
		}

	}

}

func SelfCallingTxProcessor(w http.ResponseWriter, r *http.Request, node *UnityNode, selfContact api.Contact) {

	transactionType := r.URL.Query().Get("type")
	transactionToken := r.URL.Query().Get("transactionToken")

	start1 := time.Now()

	// if transactionType == "auto" {
	listOfLeaders := GetValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_Leaders_Epoch_1_Group_1"))

	IDData := listOfLeaders.Contact[0].ID
	Add := listOfLeaders.Contact[0].Address
	Qid := int32(listOfLeaders.Contact[0].QuorumID)

	selfContact = api.Contact{
		ID:       IDData,
		Address:  Add,
		QuorumID: Qid,
	}
	// }

	Log("Fetch Leaders databases", time.Since(start1))

	err := os.Truncate("testnet.log", 0)
	if err != nil {
		log.Fatal(err)
	}

	start := time.Now()
	batchChannel := make(chan []byte)

	transactionsRequest := TransactionRequest{}

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
		go Job(node, selfContact, batchChannel, i, transactionsRequest, transactionType, modulo, noOfLeadersToSend, transactionToken)

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
			break
			time.Sleep(500 * time.Millisecond)
		}
	}

	noOfLeaders := 0

	if transactionType == "manual" || transactionsRequest.NumberOfBatches <= 5 {
		noOfLeaders = 1
	} else {
		noOfLeaders = 2
	}

	jobResponses := api.BatchTransactions{}

	for {
		select {
		case response := <-batchChannel:
			fmt.Println("SINGLE JOB RESPONSE")
			noOfLeaders--
			voteResponse := api.BatchTransactions{}
			proto.Unmarshal(response, &voteResponse)

			// jobResponses = voteResponse
			jobResponses.BatchTransaction = append(jobResponses.BatchTransaction, voteResponse.BatchTransaction...)
		}
		if noOfLeaders == 0 {
			break
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

	for index := 0; index < len(batchIds); index++ {
		fmt.Println(batchIds[index])
	}
	fmt.Printf("%s took %v\n", "Finish Whole Scheduler transaction----------------------------------------------------", time.Since(start))
	responseInBytes, _ := json.Marshal(&response)

	for index := 0; index < len(batchIds); index++ {
		Log(batchIds[index], "Finish Whole Scheduler transaction took :", time.Since(start).Seconds(), "::size", len(responseInBytes))
	}

	// proto.Un

	w.Write(responseInBytes)
}

func Job(node *UnityNode, selfContact api.Contact, batchChannel chan []byte, k int, transactionsRequest TransactionRequest, transactionType string, modulo int, noOfLeaderFromPre int, transactionToken string) []BatchTransaction {
	listOfLeaders := GetValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_Leaders_Epoch_1_Group_"+strconv.Itoa(k)))
	jobChannel := make(chan []byte)
	var noOfBatches int32
	var noOfLeader int
	var newNumberOfBatches int32
	var numberOfTransactionInSingleBatch int32
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
			TransactionToken:                 transactionToken,
		}

		fmt.Println("NUMBER OF BATCHES", i, "->", newNumberOfBatches)

		go BootstrapSync(selfContact, listOfLeaders.Contact[i], request, jobChannel, nil)

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
	for {
		select {
		case response := <-jobChannel:

			noOfLeaders--
			voteResponse := api.BatchTransactions{}
			proto.Unmarshal(response, &voteResponse)
			jobResponses.BatchTransaction = append(jobResponses.BatchTransaction, voteResponse.BatchTransaction...)

			fmt.Println("length of single job", len(jobResponses.BatchTransaction))
		}
		if noOfLeaders == 0 {
			break
		}
	}

	data, _ := proto.Marshal(&jobResponses)
	batchChannel <- data
	return nil
}

func CreateData(w http.ResponseWriter, r *http.Request, node *UnityNode, selfContact api.Contact) {

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
			secret := &Seckey{}

			secret.SetHexString(clients.Client[selectRandNumber1].PrivateKey)

			fromData := clients.Client[selectRandNumber1].Address
			toData := clients.Client[selectRandNumber2].Address
			amount := selectRandNumberAmount

			transactionRequest := api.TransactionWithSignature{Amount: int32(amount), From: fromData, To: toData}

			transactionRequestInByte, _ := proto.Marshal(&transactionRequest)
			sign := Sign(*secret, transactionRequestInByte)
			transactionRequest.Signature = sign.GetHexString()

			transections.TransactionWithSignature = append(transections.TransactionWithSignature, &transactionRequest)

		}
		fmt.Println("transections", len(transections.TransactionWithSignature))

	} else {
		transactionRequest := []TransactionWithSignature{}
		e := json.NewDecoder(r.Body).Decode(&transactionRequest)
		fmt.Println(e)

		fmt.Println("transections in manual", transactionRequest)

		for index := 0; index < len(transactionRequest); index++ {
			transactionRequestForAPI := api.TransactionWithSignature{Amount: int32(transactionRequest[index].Amount), From: transactionRequest[index].From, To: transactionRequest[index].To, Signature: transactionRequest[index].Signature, Message: transactionRequest[index].Message}
			transections.TransactionWithSignature = append(transections.TransactionWithSignature, &transactionRequestForAPI)
		}

	}

	uuid := guuid.New()

	uuidData := uuid.String()

	FillLeaderQueue(node, transections, selfContact, transactionType, transactionsRequest, uuidData)

	stringData := []string{}
	stringData = append(stringData, uuidData)

	msg := ResponseType{
		Status:  "Success",
		Message: "Transactions are generated and stored successfully",
		Data:    stringData,
	}
	res, _ := json.Marshal(&msg)

	w.Write(res)

}
func HandleSignTransaction(w http.ResponseWriter, r *http.Request, node *UnityNode) {
	clients := api.Clients{}
	client := api.Client{}

	transactionRequest := api.TransactionWithSignature{}

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&transactionRequest); err != nil {
		return
	}

	response, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(node.NodeID[:])+"_Clients"), nil)

	proto.Unmarshal(response, &clients)
	fmt.Println(len(clients.Client))

	for j := 0; j <= len(clients.Client)-1; j++ {
		if clients.Client[j].Address == transactionRequest.From {
			client = *clients.Client[j]
		}
	}

	// if (Client{}) == client {
	// 	w.Write([]byte("Client details not found"))
	// 	return
	// }

	secret := &Seckey{}

	secret.SetHexString(client.PrivateKey)

	transactionRequestInByte, _ := proto.Marshal(&transactionRequest)
	sign := Sign(*secret, transactionRequestInByte)

	transactionRequest.Signature = sign.GetHexString()

	transactionResponse, _ := json.Marshal(&transactionRequest)
	w.Write(transactionResponse)

}

func HandleGetValue(w http.ResponseWriter, r *http.Request, node *UnityNode) {
	// w.Write([]byte("welcome"))
	params := chi.URLParam(r, "id")
	if Debug == true {
		fmt.Println(params)
	}

	response, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(node.NodeID[:])+params), nil)

	if strings.Contains(params, "Decimal_DKG") {

		apiRSIPDecimalData := &api.RSIPDecimals{}
		utilRSIPDecimal := RSIPDecimals{}

		proto.Unmarshal(response, apiRSIPDecimalData)

		for index := 0; index < len(apiRSIPDecimalData.RSIPDecimal); index++ {

			newData := RSIPDecimal{
				Group:   int(apiRSIPDecimalData.RSIPDecimal[index].Group),
				Decimal: apiRSIPDecimalData.RSIPDecimal[index].Decimal,
			}

			utilRSIPDecimal = append(utilRSIPDecimal, newData)

		}

		out, _ := json.Marshal(utilRSIPDecimal)
		w.Write(out)

	} else if strings.Contains(params, "_tx_queue") {
		transactionWithoutSignature := &api.TransactionWithSignatures{}
		batchTransaction := BatchTransaction{}

		proto.Unmarshal(response, transactionWithoutSignature)

		for index := 0; index < len(transactionWithoutSignature.TransactionWithSignature); index++ {

			newData := TransactionWithSignature{
				From:      transactionWithoutSignature.TransactionWithSignature[index].From,
				To:        transactionWithoutSignature.TransactionWithSignature[index].To,
				Amount:    int(transactionWithoutSignature.TransactionWithSignature[index].Amount),
				Signature: transactionWithoutSignature.TransactionWithSignature[index].Signature,
			}

			batchTransaction.Data = append(batchTransaction.Data, newData)

		}
		out, _ := json.Marshal(batchTransaction.Data)
		w.Write(out)

	} else if strings.Contains(params, "dkg") {
		w.Write(response)
	} else {
		transactionsRequestContactAPI := &api.Contacts{}
		transactionsRequestContact := Contacts{}

		proto.Unmarshal(response, transactionsRequestContactAPI)

		for index := 0; index < len(transactionsRequestContactAPI.Contact); index++ {
			contact := Contact{
				Address:  transactionsRequestContactAPI.Contact[index].Address,
				ID:       transactionsRequestContactAPI.Contact[index].ID,
				IsLeader: transactionsRequestContactAPI.Contact[index].IsLeader,
				IsOracle: transactionsRequestContactAPI.Contact[index].IsOracle,
				Category: transactionsRequestContactAPI.Contact[index].Category,
			}

			transactionsRequestContact = append(transactionsRequestContact, contact)
		}

		out, _ := json.Marshal(transactionsRequestContact)

		w.Write(out)
	}

}

func HandleElection(w http.ResponseWriter, r *http.Request, node *UnityNode, selfContact api.Contact) {
	Scheduler(node, selfContact)
	w.Write([]byte("Election Proccess Initiated..."))
}

func CreateClient(w http.ResponseWriter, req *http.Request, node *UnityNode, selfContact api.Contact) {
	client := api.Client{}
	clients := api.Clients{}
	json.NewDecoder(req.Body).Decode(&client)

	// add Address, Public & Private key to incoming client
	client = AddRSAKeys(client)

	clients = readContact()

	clients.Client = append(clients.Client, &client)

	defer writeContact(clients, node, selfContact)

	json.NewEncoder(w).Encode(client)
}

func AddRSAKeys(request api.Client) api.Client {
	newClient := api.Client{}
	newClient = request

	randomNumber := crand.NewRand()
	sec := NewSeckeyFromRand(randomNumber.Deri(1))
	pub := NewPubkeyFromSeckey(*sec)
	addr := GetAddressFromPublicKey(pub)

	newClient.Address = hex.EncodeToString(addr[:])
	newClient.PublicKey = sec.GetHexString()
	newClient.PrivateKey = pub.GetHexString()

	return newClient
}

func readContact() (clients api.Clients) {

	byteValue, _ := ioutil.ReadFile("seed.json")
	json.Unmarshal([]byte(byteValue), &clients)

	return clients
}

func writeContact(clients api.Clients, node *UnityNode, self api.Contact) {

	clientsToStore, _ := proto.Marshal(&clients)

	err := PutValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_Clients"), clientsToStore)
	if err != nil {
		return
	}

	request := api.Request{
		Action: "Store-Updated-Client-Across-The-Network",
		Data:   clientsToStore,
	}
	contactsFromOwnDb := GetValuesById(node, []byte(string(self.ID[:])))
	for i := 0; i < len(contactsFromOwnDb.Contact); i++ {
		BootstrapSync(self, contactsFromOwnDb.Contact[i], request, nil, nil)
	}

	file, _ := json.MarshalIndent(clients, "", "    ")
	ioutil.WriteFile("seed.json", file, 0644)
}

func GetAddressFromPublicKey(pubKey *Pubkey) [20]byte {
	bytes := pubKey.Serialize()
	address := [20]byte{}
	hash := sha256.Sum256(bytes)
	copy(address[:], hash[12:])
	return address
}

// var clients []Client

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

func TransactionHistory(w http.ResponseWriter, r *http.Request, node *UnityNode, selfContact api.Contact) {

	fmt.Println("GET params were:", r.URL.Query())

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	batchID := r.URL.Query().Get("batchID")
	TxID := r.URL.Query().Get("txID")

	// from := chi.URLParam(r, "from")
	// to := chi.URLParam(r, "to")

	result := Mdb.RetrieveTransactionHistory(from, to, batchID, TxID)

	var response []byte

	if result == nil {
		noMatch := "No match found."

		response, _ = json.MarshalIndent(noMatch, "", "    ")
	} else {
		response, _ = json.MarshalIndent(result, "", "    ")
	}

	w.Write(response)
}

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

func checkErr(err error) {
	if err != nil {
		fmt.Println(err)
	}
}
