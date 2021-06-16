package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	. "github.com/unity-go/go-bls"

	"github.com/unity-go/api"
	// . "github.com/unity-go/dkg"
	. "github.com/unity-go/findValues"
	"github.com/unity-go/reedsolomon"
	. "github.com/unity-go/unityCoreRPC"
	. "github.com/unity-go/unityNode"
	. "github.com/unity-go/util"

	// groupSign "github.com/dfinity/go-dfinity-crypto/groupsig"
	guuid "github.com/google/uuid"
)

// struct for unitynode
type UnityCore2 struct {
	Node      *UnityNode
	NodeBlock *NodeBlock
}

// simple hello rpc to chech GRPC
func (uc *UnityCore2) SayHello(ctx context.Context, in *api.PingMessage) (*api.PingMessage, error) {

	log.Printf("Receive message %s", in.Greeting)

	return &api.PingMessage{Greeting: "bar"}, nil
}

// This will store contact on node by giving its NodeID as a key
func (uc *UnityCore2) Sync(ctx context.Context, request *api.Request) (*api.BootstarpSyncResponse, error) {

	req := request
	res := api.BootstarpSyncResponse{}
	uc.NodeBlock.AllNodesInNetwork = append(uc.NodeBlock.AllNodesInNetwork, req.Sender)

	value := api.Contacts{
		Contact: uc.NodeBlock.AllNodesInNetwork,
	}

	fmt.Println(len(value.Contact))
	res.Contacts = &value
	return &res, nil

}

// This function will prepare for request of gRPC call
func PrepareRequest(data []byte) api.Request {
	req := api.Request{}
	proto.Unmarshal(data, &req)

	return req
}

// This function will prepare for response of gRPC call
func PrepareResponse(contacts *api.Contacts, data []byte) []byte {
	res := api.BootstarpSyncResponse{
		Contacts: contacts,
		Data:     data,
	}
	resInBytes, _ := proto.Marshal(&res)

	return resInBytes
}

// This function is service of server which handle different types of request
func (uc *UnityCore2) SyncAction(ctx context.Context, request *api.Request) (*api.BootstarpSyncResponse, error) {

	req := request
	res := api.BootstarpSyncResponse{}

	// store BLS system value
	if req.Action == "bls_system" {
		errorFromSave := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"bls_system"), req.Data)
		if errorFromSave != nil {
			fmt.Println(errorFromSave)
		}
	}

	// store BLS param value
	if req.Action == "bls_params" {
		errorFromSave := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"bls_params"), req.Data)
		if errorFromSave != nil {
			fmt.Println(errorFromSave)
		}
	}

	// For ORACLES

	// store oracle's private key
	if req.Action == "bls_oracle_secret" {
		errorFromSave := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"bls_oracle_secret"), req.Data)
		if errorFromSave != nil {
			fmt.Println(errorFromSave)
		}
	}

	// store oracle's group private key
	if req.Action == "bls_oracle_group_secret" {
		errorFromSave := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"bls_oracle_group_secret"), req.Data)
		if errorFromSave != nil {
			fmt.Println(errorFromSave)
		}
	}

	// store oracle's public key
	if req.Action == "bls_oracle_public" {
		errorFromSave := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"bls_oracle_public"), req.Data)
		if errorFromSave != nil {
			fmt.Println(errorFromSave)
		}
	}

	// store oracle's group public key
	if req.Action == "bls_oracle_group_public" {
		uc.Node.MapMutex.Lock()
		uc.NodeBlock.OraclesPublicKey[req.Param] = req.Data
		uc.Node.MapMutex.Unlock()
	}

	// For LEADERS

	// store leader's private key
	if req.Action == "bls_leader_secret" {
		errorFromSave := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"bls_leader_secret"), req.Data)
		if errorFromSave != nil {
			fmt.Println(errorFromSave)
		}
	}

	// store leader's group private key
	if req.Action == "bls_leader_group_secret" {
		errorFromSave := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"bls_leader_group_secret"), req.Data)
		if errorFromSave != nil {
			fmt.Println(errorFromSave)
		}
	}

	// store leader's public key
	if req.Action == "bls_leader_public" {
		errorFromSave := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"bls_leader_public"), req.Data)
		if errorFromSave != nil {
			fmt.Println(errorFromSave)
		}
	}

	// store leader's group public key
	if req.Action == "bls_leader_group_public" {
		uc.Node.MapMutex.Lock()
		uc.NodeBlock.LeadersPublicKey[req.Param] = req.Data
		uc.Node.MapMutex.Unlock()
	}

	// For RSIP

	// store RSIP's private key
	if req.Action == "bls_RSIP_secret" {
		errorFromSave := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"bls_RSIP_secret"), req.Data)
		if errorFromSave != nil {
			fmt.Println(errorFromSave)
		}
	}

	// store RSIP's group private key
	if req.Action == "bls_RSIP_group_secret" {
		errorFromSave := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"bls_RSIP_group_secret"), req.Data)
		if errorFromSave != nil {
			fmt.Println(errorFromSave)
		}
	}

	// store RSIP's public key
	if req.Action == "bls_RSIP_public" {
		errorFromSave := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"bls_rsip_public"), req.Data)
		if errorFromSave != nil {
			fmt.Println(errorFromSave)
		}
	}

	// store RSIP's group public key
	if req.Action == "bls_RSIP_group_public" {
		uc.Node.MapMutex.Lock()
		uc.NodeBlock.RSIPPublicKey[strconv.Itoa(int(req.TribeNumber))] = append(uc.NodeBlock.RSIPPublicKey[strconv.Itoa(int(req.TribeNumber))], GroupsPublicKeys{
			Group:     strconv.Itoa(int(req.RSIPGroup)),
			PublicKey: req.Data,
		})
		uc.Node.MapMutex.Unlock()
	}

	if req.Action == "NodeType" {
		if req.Param == "IsOracle" {
			uc.NodeBlock.IsOracle = true
		}

		if req.Param == "IsLeader" {
			uc.NodeBlock.IsLeader = true
		}

		if req.Param == "IsRSIP" {
			uc.NodeBlock.NodePositionInTribe = int(req.RSIPGroup)
			uc.NodeBlock.NodePositionInRSIP = int(req.TribeNumber)
			uc.NodeBlock.IsRSIP = true
		}
	}

	// retrive bls key
	if req.Action == "retrive_bls_secret" {
		key := ""
		json.Unmarshal(req.Data, &key)

		getSecretKey, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+key), nil)
		fmt.Println("getSecretKey", getSecretKey)
		res.Data = getSecretKey
	}

	// store batch data in leader when user hits generate transactions API
	if req.Action == "store-data-in-leaders" {

		transactionsQueueFromDB := &api.TransactionWithSignatures{}
		upendData := &api.TransactionWithSignatures{}

		proto.Unmarshal(req.Data, upendData)

		transactionsQueueFromDB.TransactionWithSignature = append(transactionsQueueFromDB.TransactionWithSignature, upendData.TransactionWithSignature...)

		transactionQueueByte, _ := proto.Marshal(transactionsQueueFromDB)

		uc.Node.ValuesDB.Put([]byte(hex.EncodeToString(req.Target.ID[:])+"_tx_queue"+req.Param), transactionQueueByte, nil)

		fmt.Println("Stored")

		req.Data = nil
		upendData = nil
		transactionsQueueFromDB = nil
		transactionQueueByte = nil
		res.Data = []byte("stored")

	}

	// store Oracle group list on node
	if req.Action == "Oracle-List-Save" {
		if Debug == true {
			fmt.Println(hex.EncodeToString(req.Target.ID[:]) + "_Oracles_Epoch_1")
		}

		errorFromSave := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_Oracles_Epoch_1"), req.Data)
		if errorFromSave != nil {
			fmt.Println(errorFromSave)
		}

		listOfOracles := GetValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_Oracles_Epoch_1"))
		if Debug == true {
			fmt.Println("")
			fmt.Println("-------------------All Oracles of Epoch 1------------------")
			fmt.Println("")
		}

		if Debug == true {
			// for i := 0; i <= len(listOfOracles.Contact)-1; i++ {
			// 	fmt.Println(listOfOracles.Contact[i])
			// }

			fmt.Println("Number of Oracles are", len(listOfOracles.Contact))
		}

		// return nil, nil
	}

	// store group of oracles
	if strings.Contains(req.Action, "_Epoch_1_Oracle_Group_") {
		contactList := &api.Contacts{}
		proto.Unmarshal(req.Data, contactList)
		uc.Node.MapMutex.Lock()
		uc.NodeBlock.OracleList[req.Param] = contactList
		uc.Node.MapMutex.Unlock()

	}

	// store tribe of group
	if strings.Contains(req.Action, "_Tribe_Group_") {

		contactList := &api.Contacts{}
		proto.Unmarshal(req.Data, contactList)

		fmt.Println("Here to store TRIBE", len(contactList.Contact))

		uc.Node.MapMutex.Lock()
		uc.NodeBlock.TribeList[req.Param] = contactList
		uc.Node.MapMutex.Unlock()
	}

	// store batch chunk on RSIP after batch gone through consensus, and send that chunk on DHT node
	if strings.Contains(req.Action, "_Batch_Chunk_") {

		startTime := time.Now()

		var wg sync.WaitGroup
		err := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_BATCH_DATA"), req.Data)
		if err != nil {
			fmt.Println(err)
		}

		request := api.Request{
			Action:   "STORE-TRANSACTION-DETAILS-TO-DHT",
			Param:    req.Param,
			Data:     req.Data,
			ByteSize: req.ByteSize,
		}

		wg.Add(1)
		go BootstrapSync(*req.Target, req.DHTContact, request, nil, &wg, uc.Node)
		wg.Wait()

		Log(req.Param, "Time taken by single RSIP node to store chunk and get success response from DHT :", time.Since(startTime).Seconds(), "::size", int(req.ByteSize))

	}

	// store Leader's list
	if strings.Contains(req.Action, "Leaders_Epoch_1_Group_") {

		leaders := &api.Contacts{}
		proto.Unmarshal(req.Data, leaders)

		uc.Node.MapMutex.Lock()
		uc.NodeBlock.LeaderList[req.Param] = leaders
		uc.Node.MapMutex.Unlock()
	}

	// store dht contact(address)
	if req.Action == "STORE_DHT_CONTACT" {
		contactInByte, _ := json.Marshal(req.Sender)
		errorFromSave := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_DHT_Contact"), contactInByte)
		if errorFromSave != nil {
			fmt.Println(errorFromSave)
		}
	}

	// Reconstruct batch from chunks and store on mongo of DHT node as soon as it can reconstruct
	if req.Action == "STORE-TRANSACTION-DETAILS-TO-DHT" {

		startTime := time.Now()

		chunkData := ChunkWithId{}
		json.Unmarshal(req.Data, &chunkData)

		err := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_"+req.Param+"_Batch_Chunk_"+strconv.Itoa(int(chunkData.ID))), req.Data)
		if err != nil {
			fmt.Println(err)
		}

		aggregatedChunks := []ChunkWithId{}

		counter := 0
		iter := uc.Node.ValuesDB.NewIterator(BytesPrefix([]byte(hex.EncodeToString(req.Target.ID[:])+"_"+req.Param+"_Batch_Chunk_")), nil)
		for iter.Next() {
			// Use key/value.
			// key := iter.Key()
			value := iter.Value()

			partialChunk := ChunkWithId{}
			json.Unmarshal(value, &partialChunk)

			// fmt.Println("key", string(key))

			if counter == 0 {
				aggregatedChunks = append(aggregatedChunks, partialChunk)
			}
			if counter != 0 {
				if IsChunkContains(aggregatedChunks, partialChunk) == false {

					if len(aggregatedChunks) != 3 {
						aggregatedChunks = append(aggregatedChunks, partialChunk)
					}
				}
			}

			counter++

			// fmt.Println(len(aggregatedChunks), "length")

			// lastBin := strings.LastIndex(string(key), "_Batch_Chunk_")
			// batchId := string(key)[lastBin+13:]

			// batchIdFromDb, err := Mdb.GetBatchId(batchId)
			// fmt.Println("batchIdFromDb", batchIdFromDb, err)

			// if err != nil {
			// 	var latestTransactions []interface{}

			// 	json.Unmarshal(value, &latestTransactions)
			// 	fmt.Println("length at second", len(latestTransactions))

			// 	Mdb.BulkAdd(latestTransactions)

			// 	Mdb.StoreBatchId(batchId)
			// }

			// uc.Node.ValuesDB.Delete(key, nil)

		}

		// reconstructed batch
		var reconstructedBatch []interface{}

		if uc.Node.IsBatchReconstructedAtDHT[req.Param] == false {
			if len(aggregatedChunks) == 3 {
				// fmt.Println("Aggregated chunks")
				startTime1 := time.Now()
				uc.Node.MapMutex.Lock()
				uc.Node.IsBatchReconstructedAtDHT[req.Param] = true
				uc.Node.MapMutex.Unlock()
				reconstructedBatch = ReconstructBatch(4, int(req.ByteSize), aggregatedChunks)
				fmt.Println(len(reconstructedBatch), "reconstructedBatch")

				Mdb.BulkAdd(reconstructedBatch)

				Mdb.StoreBatchId(req.Param)

				Log(req.Param, "Reconstruct batch and store on MongoDb took :", time.Since(startTime1).Seconds(), "::size", int(req.ByteSize))

			}
		}

		fmt.Println(counter)
		iter.Release()
		err = iter.Error()
		if err != nil {
			panic(err)
		}

		Log(req.Param, "Time taken on DHT node to process chunk, if this chunks will be third chunk then DHT will reconstruct batch and store in MongoDB :", time.Since(startTime).Seconds(), "::size", int(req.ByteSize))

		// fmt.Println("STORE-TRANSACTION-DETAILS-TO-DHT ------>  ENDING", req.Target.Address)
	}

	// store list of decimal numbers of groups of RSIP
	if strings.Contains(req.Action, "_Decimal_DKG") {
		errorFromSave := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+req.Action), req.Data)
		if errorFromSave != nil {
			fmt.Println(errorFromSave)
		}
	}

	// store tribe
	if strings.Contains(req.Action, "-Tribe_") {

		RSIPNodes := api.Contacts{}

		proto.Unmarshal(req.Data, &RSIPNodes)
		uc.Node.MapMutex.Lock()
		uc.NodeBlock.RSIPList[strconv.Itoa(int(req.TribeNumber))] = append(uc.NodeBlock.RSIPList[strconv.Itoa(int(req.TribeNumber))], GroupList{
			Group:    strconv.Itoa(int(req.RSIPGroup)),
			Contacts: RSIPNodes,
		})
		uc.Node.MapMutex.Unlock()
	}

	// TO make tcp connections between nodes
	if req.Action == "_TCP_CONNECTIONS" {

		var wg sync.WaitGroup
		contactsFromOwnDb := GetValuesById(uc.Node, []byte(string(uc.Node.NodeID[:])))

		request := api.Request{
			Action: "_TCP_CONNECTIONS_FOR_OTHERS",
		}
		for l := 0; l < len(contactsFromOwnDb.Contact); l++ {

			if contactsFromOwnDb.Contact[l].Address != req.Target.Address {
				wg.Add(1)

				go BootstrapSync(*req.Target, contactsFromOwnDb.Contact[l], request, nil, &wg, uc.Node)
			}
		}

		wg.Wait()
	}

	// TO make tcp connections between nodes
	if req.Action == "_TCP_CONNECTIONS_FOR_OTHERS" {

		// fmt.Println("IN OTHERS")
	}

	// To store block

	if req.Action == "STORE_BLOCK_IN_DB" {
		blockDataInByte, _ := json.Marshal(uc.NodeBlock)
		err := PutValuesById(uc.Node, []byte(string(uc.Node.NodeID[:])+"_Block_"+strconv.Itoa(uc.Node.CurrentElectionBlock)), blockDataInByte)
		CheckErr(err)
	}

	// starting election process

	if req.Action == "STARTING_ELECTION_PROCESS" {
		uc.Node.CurrentElectionBlock = int(req.Index)
		uc.NodeBlock.BlockNumber = int(req.Index)
		uc.NodeBlock.RSIPList = make(map[string][]GroupList)
		uc.NodeBlock.RSIPPublicKey = make(map[string][]GroupsPublicKeys)
	}

	/*
		Initial step of transaction process from where,
		Leader will take batch details from own storage and
		Send it to oracles
		Once batch gone through the consensus, from here we are sending chunks to
		RSIP and DHT node to store batch in database
	*/
	if req.Action == "Start-Queued-Batch" {
		start := time.Now()
		leaderGroupNumber, _ := strconv.Atoi(req.Param)
		transactionType := ""
		var numberOfBatches int
		transactionType = string(req.TransactionType)

		// fmt.Println("\nSTART QUEUE function------------------------------------------>", req.Target.Address)

		transactionsQueueFromDB, singleBatch, remainingBatch := api.TransactionWithSignatures{}, api.TransactionWithSignatures{}, api.TransactionWithSignatures{}

		batchChannel := make(chan api.BatchTransaction)

		batchResponses := api.BatchTransactions{}

		if transactionType == "auto" {
			response, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"_tx_queue"+req.TransactionToken), nil)

			proto.Unmarshal(response, &transactionsQueueFromDB)

		} else {
			transactionsQueueFromDB.TransactionWithSignature = append(transactionsQueueFromDB.TransactionWithSignature, req.SingleTransaction)
		}

		// var mutex sync.Mutex
		// mutex.Lock()

		if req.NumberOfBatches == 0 {
			numberOfBatches = 1
		} else {
			numberOfBatches = int(req.NumberOfBatches)
		}
		lengthOfSingleBatch := int(req.NumberOfTransactionInSingleBatch)

		invalidTransactions := api.BatchTransactions{}

		for index := 0; index < numberOfBatches; index++ {

			start2 := time.Now()

			if transactionType == "auto" {
				// retrieve all batch from queue

				// fmt.Println("BEFORE", lengthOfSingleBatch)

				// retrieve first batch from queue
				if len(transactionsQueueFromDB.TransactionWithSignature) >= lengthOfSingleBatch {
					singleBatch.TransactionWithSignature = append(singleBatch.TransactionWithSignature, transactionsQueueFromDB.TransactionWithSignature[:lengthOfSingleBatch]...)
					remainingBatch.TransactionWithSignature = append(remainingBatch.TransactionWithSignature, transactionsQueueFromDB.TransactionWithSignature[lengthOfSingleBatch:]...)
				}

				transactionsQueueFromDB.TransactionWithSignature = remainingBatch.TransactionWithSignature
			} else {
				singleBatch.TransactionWithSignature = transactionsQueueFromDB.TransactionWithSignature
			}

			individualBatch := api.BatchTransaction{}
			individualBatch.ID = guuid.New().String()
			individualBatch.Data = singleBatch.TransactionWithSignature

			fmt.Println("INDIVIDUAL BATCH", len(individualBatch.Data))

			Log(individualBatch.ID, "Get batch for leader from database took :", time.Since(start2).Seconds(), "::size", len(individualBatch.Data))

			for index := 0; index < len(individualBatch.Data); index++ {
				uuid := guuid.New()
				individualBatch.Data[index].ID = uuid.String()
			}

			singleBatch = api.TransactionWithSignatures{}

			remainingBatch = api.TransactionWithSignatures{}

			// remove invalid transactions from batch

			clients := api.Clients{}
			response, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(uc.Node.NodeID[:])+"_Clients"), nil)

			proto.Unmarshal(response, &clients)

			faultyTx := api.BatchTransaction{}
			faultyTx.ID = individualBatch.ID
			for index := 0; index < len(individualBatch.Data); index++ {
				// individualBatch.Data[index].ID = uuid.String()

				verifiedBalance := VerifyAmount(uc.Node, *individualBatch.Data[index], clients)
				if verifiedBalance != "" {
					// invalidTransactions = append(invalidTransactions, api.BatchTransaction{
					// 	ID: individualBatch.ID,
					// 	Data:
					// })
					individualBatch.Data[index].Status = "FAIL"
					faultyTx.Data = append(faultyTx.Data, individualBatch.Data[index])
					individualBatch.Data = append(individualBatch.Data[:index], individualBatch.Data[index+1:]...)
				}

			}

			invalidTransactions.BatchTransaction = append(invalidTransactions.BatchTransaction, &api.BatchTransaction{
				ID:   faultyTx.ID,
				Data: faultyTx.Data,
			})

			fmt.Println("invalid transactions", len(faultyTx.Data))
			fmt.Println("Valid transactions", len(individualBatch.Data))

			// first batch goes for validation process
			go QueueHandleTransaction(individualBatch, uc.Node, req.Target, leaderGroupNumber, batchChannel, uc.NodeBlock)

			individualBatch = api.BatchTransaction{}

			if transactionType == "auto" && index != (numberOfBatches-1) {

				timeToSleep, _ := strconv.Atoi(string(req.Value))

				time.Sleep(time.Millisecond * time.Duration(timeToSleep+500))

				// fmt.Println("Time Duration in tribe", time.Millisecond*time.Duration(timeToSleep+500))
			}

		}

	OuterLoop:
		for {
			response := <-batchChannel

			// vote
			numberOfBatches--
			tribeNumber := numberOfBatches % 5

			fmt.Println("tribe number", tribeNumber)
			batchResponses.BatchTransaction = append(batchResponses.BatchTransaction, &response)

			fmt.Println("AT LAST transactions", len(response.Data))
			for index := 0; index < len(invalidTransactions.BatchTransaction); index++ {
				if invalidTransactions.BatchTransaction[index].ID == response.ID {
					fmt.Println("AT LAST invalid transactions", len(invalidTransactions.BatchTransaction[index].Data))
					response.Data = append(response.Data, invalidTransactions.BatchTransaction[index].Data...)
				}
			}

			// store data on nodes

			var latestTransactions []interface{}

			for i := 0; i < len(response.Data); i++ {

				var status string

				if response.Data[i].Status == "FAIL" {
					status = response.Data[i].Status
				} else {
					status = "SUCCESS"
				}

				response.Data[i].Status = status
				latestTx := TransactionWithSignature{}
				latestTx.ID = response.Data[i].ID
				latestTx.From = response.Data[i].From
				latestTx.To = response.Data[i].To
				latestTx.Amount = int(response.Data[i].Amount)
				latestTx.Status = status
				latestTx.Signature = response.Signature
				latestTx.Message = response.Data[i].Message
				latestTx.BatchID = response.ID
				latestTransactions = append(latestTransactions, &latestTx)
				// go Mdb.Add(latestTx, mongoChan)
			}

			// store updated client in network

			start3 := time.Now()
			clients := api.Clients{}
			contactsFromOwnDb := api.Contacts{}
			contactsFromOwnDb.Contact = uc.NodeBlock.AllNodesInNetwork
			dataInBytes, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(uc.Node.NodeID[:])+"_Clients"), nil)
			proto.Unmarshal(dataInBytes, &clients)

			for k := 0; k < len(response.Data); k++ {

				if response.Data[k].Status == "SUCCESS" {

					for j := 0; j <= len(clients.Client)-1; j++ {
						if clients.Client[j].Address == response.Data[k].From {
							clients.Client[j].Amount = clients.Client[j].Amount - response.Data[k].Amount
						}
						if clients.Client[j].Address == response.Data[k].To {
							clients.Client[j].Amount = clients.Client[j].Amount + response.Data[k].Amount
						}

					}
				}
				// batch.Data[k].Status = "SUCCESS"
			}
			byteValue, _ := proto.Marshal(&clients)

			if ShowTime == true {
				Log(response.ID, "After Checking amount for batch :", time.Since(start3).Seconds())
			}

			Log(response.ID, "After Checking amount for batch :", time.Since(start3))

			// chanName := make(chan []byte)

			fmt.Println("lengthOfClients", len(byteValue))

			// transactionInByte, _ := json.Marshal(&latestTransactions)

			var wg sync.WaitGroup

			// go func() {
			for i := 0; i <= len(contactsFromOwnDb.Contact)-1; i++ {
				wg.Add(1)
				// if contactsFromOwnDb[i].ID != req.Target.ID {

				request := api.Request{
					Action: "Store-Updated-Client-Across-The-Network",
					Data:   byteValue,
				}

				go BootstrapSync(*req.Target, contactsFromOwnDb.Contact[i], request, nil, &wg, uc.Node)
			}
			wg.Wait()
			start6 := time.Now()
			Mdb.BulkAdd(latestTransactions)

			Log(response.ID, "Store batch to central mongoDb :", time.Since(start6))

			// chunkOfBatchInByte, _ := json.Marshal(latestTransactions)

			// msg := MessageResponse{
			// 	ID:                  response.ID,
			// 	Time:                (float64(time.Since(start) / time.Millisecond)) / 1000,
			// 	NumberOfTransaction: len(chunkOfBatchInByte),
			// }

			// msgInByte, _ := json.Marshal(msg)

			// err := uc.Node.WsConnection.WriteMessage(1, msgInByte)
			// fmt.Println(err, "here2")
			// if err != nil {
			// 	log.Println("write:", err)
			// 	// break
			// }

			// fmt.Println("-------saving batch by selecting tribes------")

			// // To Do

			// timeToChunk := time.Now()

			// // Batch to chunks of 8
			// enc, err := reedsolomon.New(3, 1)
			// CheckErr(err)

			// // Split
			// chunksOfBatch, err := enc.Split(chunkOfBatchInByte)
			// CheckErr(err)

			// // Encode & Verify
			// err = enc.Encode(chunksOfBatch)
			// CheckErr(err)

			// newChunks := []ChunkWithId{}

			// for i := 0; i < len(chunksOfBatch); i++ {
			// 	chunk := ChunkWithId{}
			// 	chunk.ID = int32(i + 1)
			// 	chunk.Data = chunksOfBatch[i]

			// 	newChunks = append(newChunks, chunk)
			// }

			// verify, err := enc.Verify(chunksOfBatch)
			// CheckErr(err)
			// if Debug == true {
			// 	fmt.Println("\nVerified: ", verify, len(chunksOfBatch))
			// }

			// fmt.Println(len(newChunks), "chunksOfBatch")

			// Log(response.ID, "Time to create and verify four chunks of batch using erasure :", time.Since(timeToChunk).Seconds(), "::size", len(chunkOfBatchInByte))

			// // dht contact

			// dhtContactInByte, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"_DHT_Contact"), nil)
			// DHTContact := api.Contact{}
			// json.Unmarshal(dhtContactInByte, &DHTContact)

			// startTime := time.Now()

			// var wg2 sync.WaitGroup

			// for index := 0; index < len(newChunks); index++ {

			// 	chunkInByte, _ := json.Marshal(newChunks[index])

			// 	// for len(latestTransactions) > 0 {
			// 	for i := 0; i <= 2; i++ {
			// 		// var chunkOfBatch []interface{}
			// 		// var chunkOfBatchRemaining []interface{}

			// 		// for k := 0; k < 1000; k++ {
			// 		// randNumber := 0
			// 		// min := 0
			// 		// max := len(latestTransactions)
			// 		// if max > 0 {
			// 		// 	randNumber = rand.Intn(max-min) + min
			// 		// }
			// 		// if len(latestTransactions) != 0 {
			// 		// selectedNode := latestTransactions[1000]
			// 		// chunkOfBatch = append(chunkOfBatch, latestTransactions[:1000]...)
			// 		// chunkOfBatchRemaining = append(chunkOfBatchRemaining, latestTransactions[1000:]...)

			// 		// latestTransactions = chunkOfBatchRemaining

			// 		// chunkOfBatch = append(chunkOfBatch, selectedNode)
			// 		// max--

			// 		// }p
			// 		// }
			// 		// if Debug == true {
			// 		// }

			// 		listOfRSIPNodes := GetValuesById(uc.Node, []byte(hex.EncodeToString(uc.Node.NodeID[:])+"-Tribe_"+strconv.Itoa(tribeNumber)+"_RSIP_GROUP_"+strconv.Itoa(i)))
			// 		fmt.Println(len(chunkInByte), "single chunk UI", len(listOfRSIPNodes.Contact), "_Batch_Chunk_"+response.ID)

			// 		request := api.Request{
			// 			Action:     "_Batch_Chunk_" + response.ID,
			// 			Data:       chunkInByte,
			// 			Param:      response.ID,
			// 			DHTContact: &DHTContact,
			// 			ByteSize:   int64(len(chunkOfBatchInByte)),
			// 		}

			// 		// var wg sync.WaitGroup
			// 		for k := 0; k <= len(listOfRSIPNodes.Contact)-1; k++ {

			// 			if listOfRSIPNodes.Contact[k].Category == "alpha" {
			// 				wg2.Add(1)

			// 				go BootstrapSync(*req.Target, listOfRSIPNodes.Contact[k], request, nil, &wg2, uc.Node)
			// 				// break
			// 			}
			// 		}

			// 	}
			// }
			// wg2.Wait()
			// Log(response.ID, "Time to complete whole storage process from leader to RSIP nodes of group and then DHT :", time.Since(startTime).Seconds(), "::size", len(chunkOfBatchInByte))
			// // till here
			// chunkOfBatchInByte = nil

			// store data to global DHT

			// for index2 := 0; index2 <= 2; index2++ {
			// 	listOfRSIPNodes := GetValuesById(uc.Node, []byte(hex.EncodeToString(uc.Node.NodeID[:])+"-Tribe_"+strconv.Itoa(tribeNumber)+"_RSIP_GROUP_"+strconv.Itoa(index2)))
			// 	request := api.Request{
			// 		Action: "STORE-TRANSACTION-DETAILS-TO-DHT",
			// 		Param:  response.ID,
			// 	}

			// 	var wg sync.WaitGroup
			// 	for k := 0; k <= len(listOfRSIPNodes.Contact)-1; k++ {

			// 		if listOfRSIPNodes.Contact[k].Category == "alpha" {

			// 			wg.Add(1)

			// 			go BootstrapSync(*req.Target, listOfRSIPNodes.Contact[k], request, nil, &wg, uc.Node)
			// 			break
			// 			// time.Sleep(200 * time.Millisecond)
			// 		}
			// 	}

			// 	wg.Wait()

			// 	fmt.Println("OUTSUDE")
			// }

			// }

			// GetValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_Tribe"))

			if ShowTime == true {
				//batchResponsesInBytes, _ := proto.Marshal(&batchResponses)
				Log(response.ID, "Single Batch Response Time :", time.Since(start).Seconds())
				//Log(response.ID, "Single Batch Response Time :", time.Since(start), len(transactionsQueueFromDB.TransactionWithSignature), len(singleBatch.TransactionWithSignature), len(remainingBatch.TransactionWithSignature))
			}

			uc.Node.ValuesDB.Put([]byte(hex.EncodeToString(req.Target.ID[:])+"_tx_queue"+req.TransactionToken), []byte{}, nil)
			if numberOfBatches == 0 {
				go func() {
					for elem := range batchChannel {
						if Debug == true {
							fmt.Println(elem)
						}
					}
					close(batchChannel)
				}()
				break OuterLoop
			}
		}

		batchResponsesInBytes, _ := proto.Marshal(&batchResponses)

		transactionsQueueFromDB, singleBatch, remainingBatch = api.TransactionWithSignatures{}, api.TransactionWithSignatures{}, api.TransactionWithSignatures{}
		batchResponses = api.BatchTransactions{}

		res.Data = batchResponsesInBytes
	}

	/*
	 This function checks validation of perticular transaction from single batch
	 It will call on Leaders and RSIPs as they both have same process to follow
	*/
	if req.Action == "Transaction-To-Proceed" {

		start := time.Now()
		isInvalid := false

		transactionRequest := api.BatchTransaction{}
		proto.Unmarshal(req.Data, &transactionRequest)

		if ShowTime == true {
			if req.Param == "RSIP-NODE" {
				// Log(transactionRequest.ID, "Time to unmarshal REQUEST in RSIP", time.Since(start15))
			}
		}
		clients := api.Clients{}
		response, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(uc.Node.NodeID[:])+"_Clients"), nil)

		proto.Unmarshal(response, &clients)

		//start16 := time.Now()
		hash := sha256.Sum256(req.Data)
		if req.Param == "RSIP-NODE" {
			hashIn32 := [32]byte{}
			for index := 0; index < len(req.Hash); index++ {
				hashIn32[index] = req.Hash[index]
			}
			// keyInByte := "bls_oracle_group_" + strconv.Itoa(int(req.RSIPGroup)) + "_public"
			keyInByte := uc.NodeBlock.OraclesPublicKey[strconv.Itoa(int(req.RSIPGroup))]

			verified := VerifyBLSBatchSignature(uc.Node, hashIn32, req.Signature, keyInByte)

			if verified != "" {
				// fmt.Println("Failed to verify signature__________________________________________________________________________ RSIP BETA", req.Target.Address)
			} else {
				// if Debug == true {
				// }
			}
		} else {
			// Store batch in levelDB
			uc.Node.ValuesDB.Put([]byte(hex.EncodeToString(uc.Node.NodeID[:])+"_batch_"+transactionRequest.ID), req.Data, nil)
		}
		if ShowTime == true {
			// if req.Param == "RSIP-NODE" {
			// 	Log(transactionRequest.ID, "time to verify signature in RSIP", time.Since(start16))
			// }
		}

		voteResponse := &api.VoteResponse{}

		for index := 0; index < len(transactionRequest.Data); index++ {
			if req.Param == "RSIP-NODE" {
				voteResponse = transactionRequest.Data[index].RSIP.VoteResponse[0]
			} else {
				voteResponse = transactionRequest.Data[index].Leaders.VoteResponse[0]
			}

			verifiedBalance := VerifyAmount(uc.Node, *transactionRequest.Data[index], clients)

			if verifiedBalance != "" {

				if voteResponse.Vote == "NO" {
				} else {
					// isInvalid = true
					isInvalid = false

				}
			} else {
				if voteResponse.Vote == "YES" {
				} else {
					// isInvalid = true
					isInvalid = false

				}
			}

			// voteResponses.VoteResponse = append(voteResponses.VoteResponse, &voteResponse)

		}

		shareToBytes := []byte{}
		if isInvalid == false {
			// Generate Share of Aggregates Signature by Other Leaders
			paraBytes, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"bls_params"), nil)
			params, _ := ParamsFromBytes(paraBytes)

			pairing := GenPairing(params)
			sysBytes, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"bls_system"), nil)
			system, _ := SystemFromBytes(pairing, sysBytes)
			share := Signature{}
			key := ""
			if req.Param == "RSIP-NODE" {
				key = "bls_RSIP_group_secret"

				hashIn32 := [32]byte{}
				for index := 0; index < len(req.Hash); index++ {
					hashIn32[index] = req.Hash[index]
				}

				share = GenerateShares(uc.Node, hashIn32, key, system)

				// if ShowTime == true {
				// 	Log(transactionRequest.ID, "Verify and Generate share from single node(RSIP) took :", time.Since(start).Seconds())
				// }

			} else {
				key = "bls_leader_group_secret"
				share = GenerateShares(uc.Node, hash, key, system)

				if ShowTime == true {
					Log(transactionRequest.ID, "Verify(check validity) and Generate share from single node(Leader) took :", time.Since(start).Seconds())
				}

			}
			// share = GenerateShares(uc.Node, hash, key, system)

			shareToBytes = system.SigToBytes(share)
			system.Free()
			pairing.Free()
			params.Free()

		}

		if ShowTime == true {
			// Log(transactionRequest.ID, "Verify and Vote from single node took", time.Since(start134))
		}

		transactionRequest = api.BatchTransaction{}
		req.Data = nil
		voteResponse = nil

		res.Data = shareToBytes

	}

	/*
	 Calls on Oracle, it will verify the shares coming from the leaders,
	 And once it validate the batch, it generate its shares and send batch to further validation
	 on RSIP
	*/
	if req.Action == "Transaction-To-Validate-And-Check-Threshold-From-Oracles-On-First-Oracle" {

		leaderGroupNumber := req.Param
		// start3 := time.Now()

		//start := time.Now()
		// err := godotenv.Load()
		// if err != nil {
		// 	fmt.Println("Error loading .env file")
		// }

		if Debug == true {
			fmt.Println("", "First Oracle------------------------------")
		}

		// numberOfRSIPGroup := os.Getenv("NUMBER_OF_RSIP_GROUP")
		// numberOfTribesGroup := os.Getenv("NUMBER_OF_TRIBES")
		// numberOfGroup, _ := strconv.Atoi(numberOfRSIPGroup)
		// numberOfTribes, _ := strconv.Atoi(numberOfTribesGroup)

		transactionRequest := api.BatchTransaction{}

		proto.Unmarshal(req.Data, &transactionRequest)

		// Store batch in levelDB
		uc.Node.ValuesDB.Put([]byte(hex.EncodeToString(uc.Node.NodeID[:])+"_batch_"+transactionRequest.ID), req.Data, nil)

		start := time.Now()
		// Verify batch usign selected leader group
		// keyInByte := "bls_leader_group_" + leaderGroupNumber + "_public"
		keyInByte := uc.NodeBlock.LeadersPublicKey[leaderGroupNumber]

		signature := req.Signature
		hash := sha256.Sum256(req.Data)
		verified := VerifyBLSBatchSignature(uc.Node, hash, signature, keyInByte)

		if verified != "" {
			// fmt.Println("Failed to verify signature__________________________________________________________________________ LEADER")
		}

		// if ShowTime == true && req.Target.SelectedTribe == 0 {
		if ShowTime == true {
			Log(transactionRequest.ID, "Verify signature by single oracle received from leader took :", time.Since(start).Seconds(), "::size", len(signature))
		}

		// generate shares
		// transactionRequestInByte, _ = proto.Marshal(&transactionRequest)

		thresholdOfOracle, _ := strconv.Atoi(os.Getenv("THRESHOLD_OF_ORACLES"))
		totalOracles, _ := strconv.Atoi(os.Getenv("NUMBER_OF_ORACLES"))
		totalOracleGroups, _ := strconv.Atoi(os.Getenv("NUMBER_OF_ORACLE_GROUPS"))

		numberOfOracles := totalOracles / (totalOracleGroups + 1)
		// Generate Share of Aggregates Signature by Batch Proposer

		start2 := time.Now()

		memberIds := rand.Perm(numberOfOracles)[:thresholdOfOracle]
		paraBytes, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"bls_params"), nil)
		params, _ := ParamsFromBytes(paraBytes)

		pairing := GenPairing(params)
		sysBytes, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"bls_system"), nil)
		system, _ := SystemFromBytes(pairing, sysBytes)

		shares := []Signature{}

		key := "bls_oracle_group_secret"
		shares = append(shares, GenerateShares(uc.Node, hash, key, system))

		// if ShowTime == true && req.Target.SelectedTribe == 0 {
		if ShowTime == true {
			Log(transactionRequest.ID, "Generate share by single alpha oracle took :", time.Since(start2).Seconds())
		}

		// fmt.Println("IN ALPHA", hash)
		// if verified != "" {
		// 	fmt.Println("Failed to verify signature__________________________________________________________________________ LEADER")
		// } else {
		if Debug == true {
			fmt.Println("VERIFIED", verified)
		}
		// }

		if ShowTime == true {
			//fmt.Println("verified +++++++++++++++++++++", verified)
		}

		start3 := time.Now()

		RSIPGroup := DeterministicValue(hash[:], signature)

		// Select RSIP group based on signature and hash determined value
		RSIPDecimalData := api.RSIPDecimals{}
		decimalWithGroup, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"_Tribe_"+strconv.Itoa(int(req.Target.SelectedTribe))+"_Decimal_DKG"), nil)
		proto.Unmarshal(decimalWithGroup, &RSIPDecimalData)

		getSelectedGroup := *RSIPDecimalData.RSIPDecimal[RSIPGroup]
		selectedRSIPGroup := getSelectedGroup.Group

		// fmt.Println(selectedRSIPGroup, "Selected RSIP")

		// if ShowTime == true && req.Target.SelectedTribe == 0 {
		if ShowTime == true {
			Log(transactionRequest.ID, "Select RSIP group based on Hash and Signature by Alpha Oracle took :", time.Since(start3).Seconds())
		}

		if ShowTime == true {
			// Log(transactionRequest.ID, "Single Alpha Oracle votes by itself", time.Since(start6))
		}
		start4 := time.Now()
		oraclesFromDB := &api.Contacts{}

		// oracles, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"_Epoch_1_Oracle_Group_"+strconv.Itoa(int(req.Target.SelectedNumber))), nil)
		oraclesFromDB = uc.NodeBlock.OracleList[strconv.Itoa(int(req.Target.SelectedNumber))]

		// proto.Unmarshal(oracles, oraclesFromDB)

		// if Debug == true {
		// 	Log(transactionRequest.ID, "Verify and check threshold for leader from first oracle took", time.Since(start))
		// }

		betaOraclesCount := len(oraclesFromDB.Contact) - 1
		betaOraclesResponseChan := make(chan []byte)

		request := api.Request{
			Action:    "Transaction-to-beta",
			Param:     leaderGroupNumber,
			RSIPGroup: int32(selectedRSIPGroup),
			Hash:      hash[:],
			Signature: signature,
			BatchID:   transactionRequest.ID,
			// System:    sysBytes,
		}

		for j := 0; j <= len(oraclesFromDB.Contact)-1; j++ {

			if !Equal(oraclesFromDB.Contact[j].ID, req.Target.ID) {
				if Debug == true {
					fmt.Println("BETA", j)
				}
				oraclesFromDB.Contact[j].SelectedTribe = req.Target.SelectedTribe
				go BootstrapSync(*req.Target, oraclesFromDB.Contact[j], request, betaOraclesResponseChan, nil, uc.Node)

			}

		}

	OutsideLoop:
		for {
			response := <-betaOraclesResponseChan
			betaOraclesCount--
			signShareFromOtherLeaders, _ := system.SigFromBytes(response)

			shares = append(shares, signShareFromOtherLeaders)

			// fmt.Println(len(shares) == thresholdOfOracle)
			if len(shares) == thresholdOfOracle {
				betaOraclesCount = 0
			}
			if Debug == true {
				fmt.Println("", response)
			}
			if betaOraclesCount == 0 {
				// if ShowTime == true && req.Target.SelectedTribe == 0 {
				if ShowTime == true {
					Log(transactionRequest.ID, "Each alpha ORACLE getting response from all other ALPHAs and Betas ORACLE took :", time.Since(start4).Seconds())
				}
				go func() {
					for elem := range betaOraclesResponseChan {
						if Debug == true {
							fmt.Println(elem)
						}
					}
					close(betaOraclesResponseChan)
				}()
				break OutsideLoop
			}
		}

		start5 := time.Now()

		aggregatedSignature, err := Threshold(shares, memberIds, system)
		if err != nil {
			fmt.Println(err, "Signature Generation Error")
		}

		aggregatedSignatureInBytes := system.SigToBytes(aggregatedSignature)

		for i := 0; i < thresholdOfOracle; i++ {
			shares[i].Free()
		}
		aggregatedSignature.Free()

		// fmt.Println("aggregatedSignatureInBytes", aggregatedSignatureInBytes)

		// if ShowTime == true && req.Target.SelectedTribe == 0 {
		if ShowTime == true {
			Log(transactionRequest.ID, "Genrate aggregated signature by alpha Oracle took :", time.Since(start5).Seconds())
		}

		// transactionRequest.Signature = shareToBytes

		if ShowTime == true {
			// Log(transactionRequest.ID, "Select RSIP groups from Tribe by deterministic number took", time.Since(start5))
		}

		start8 := time.Now()

		RSIPVotes := api.VoteResponses{}
		ch := make(chan []byte)

		// for j := 0; j < numberOfTribes+1; j++ {
		go TransactionToRSIP(uc.Node, *req, int(selectedRSIPGroup), ch, RSIPVotes, int(req.Target.SelectedTribe), transactionRequest, aggregatedSignatureInBytes, int(req.Target.SelectedNumber), leaderGroupNumber, req.BatchProposerLeader, uc.NodeBlock)

		system.Free()
		pairing.Free()
		params.Free()

		// }

		RSIPGroupCount := 1
		aggregatedSignatureFromRSIP := []byte{}
	Outer:
		for {
			response := <-ch
			RSIPGroupCount--

			if Debug == true {
				fmt.Println("Here In Oracle", response)
			}

			if RSIPGroupCount == 0 {
				if Debug == true {
					fmt.Println("Alpha to other beta's and alphas", aggregatedSignatureFromRSIP)
					fmt.Println("Back to Leader from Alpha Oracle")
				}
				if ShowTime == true {
					Log(transactionRequest.ID, "Transaction to proceed in Single Oracle (Oracle => RSIP, RSIP => Oracle(In Back Process) , Oracle => Leader(In Back Process)) took :", time.Since(start8).Seconds())
				}
				go func() {
					for elem := range ch {
						if Debug == true {
							fmt.Println(elem)
						}
					}
					close(ch)
				}()
				transactionRequest = api.BatchTransaction{}
				RSIPVotes = api.VoteResponses{}
				res.Data = []byte("Done")
				break Outer
			}
		}

	}

	/*
	   Calles on beta Oracles, generate its shares and sends to alpha oracles
	*/
	if req.Action == "Transaction-to-beta" {
		// start := time.Now()
		// err := godotenv.Load()
		// if err != nil {
		// 	fmt.Println("Error loading .env file")
		// }

		hashIn32 := [32]byte{}
		for index := 0; index < len(req.Hash); index++ {
			hashIn32[index] = req.Hash[index]
		}

		// Verify batch usign selected leader group

		// keyInByte := "bls_leader_group_" + req.Param + "_public"
		keyInByte := uc.NodeBlock.LeadersPublicKey[req.Param]

		verified := VerifyBLSBatchSignature(uc.Node, hashIn32, req.Signature, keyInByte)

		if verified != "" {
			// fmt.Println("Failed to verify signature__________________________________________________________________________ BETA")
		}
		// if ShowTime == true {
		// 	Log(req.Data, "Verify signature by  ALPHAs and Betas ORACLE took :", time.Since(start).Seconds())
		// }

		//else {
		if Debug == true {
			fmt.Println("VERIFIED", verified)
		}
		// }

		// Select RSIP group based on signature and hash determined value
		RSIPGroup := DeterministicValue(req.Hash[:], req.Signature)

		RSIPDecimalData := api.RSIPDecimals{}
		decimalWithGroup, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"_Tribe_"+strconv.Itoa(int(req.Target.SelectedTribe))+"_Decimal_DKG"), nil)
		proto.Unmarshal(decimalWithGroup, &RSIPDecimalData)

		decimalWithGroup = nil

		getSelectedGroup := *RSIPDecimalData.RSIPDecimal[RSIPGroup]
		selectedRSIPGroup := getSelectedGroup.Group

		shareToBytes := []byte{}

		if selectedRSIPGroup == req.RSIPGroup {

			// Generate Share of Aggregates Signature by Other Leaders
			paraBytes, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"bls_params"), nil)
			params, _ := ParamsFromBytes(paraBytes)

			pairing := GenPairing(params)
			sysBytes, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"bls_system"), nil)
			system, _ := SystemFromBytes(pairing, sysBytes)
			share := Signature{}

			key := "bls_oracle_group_secret"
			share = GenerateShares(uc.Node, hashIn32, key, system)

			shareToBytes = system.SigToBytes(share)
			if ShowTime == true {
				// Log(req.BatchID, "Verify selected RSIP group and signature by single beta oracle took :", time.Since(start).Seconds())
			}

		} else {
			fmt.Println("RSIP GROUP IS NOT SAME")
		}
		RSIPDecimalData = api.RSIPDecimals{}
		res.Data = shareToBytes

	}

	/*
	 Calls on RSIP alpha,
	 RSIP alpha will verify shares from Oracles
	*/
	if req.Action == "Transaction-To-Proceed-In-RSIP-ALPHA" {

		// This service is used to proceed batch in Alpha RSIP, RSIP will send batch to other alpha and

		// start := time.Now()
		transactionRequest := &api.BatchTransaction{}

		//start15 := time.Now()
		proto.Unmarshal(req.Data, transactionRequest)

		if ShowTime == true {
			// Log(transactionRequest.ID, "Time to unmarshal REQUEST", time.Since(start15))
		}
		RSIPNodes := &api.Contacts{}

		// getListOfRSIPNodes, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"-Tribe_"+strconv.Itoa(int(req.Target.SelectedTribe))+"_RSIP_GROUP_"+strconv.Itoa(int(req.RSIPGroup))), nil)
		// proto.Unmarshal(getListOfRSIPNodes, RSIPNodes)

		for index := 0; index < len(uc.NodeBlock.RSIPList[strconv.Itoa((int(req.Target.SelectedTribe)))]); index++ {
			if uc.NodeBlock.RSIPList[strconv.Itoa(int(req.Target.SelectedTribe))][index].Group == strconv.Itoa(int(req.RSIPGroup)) {
				for l := 0; l < len(uc.NodeBlock.RSIPList[strconv.Itoa(int(req.Target.SelectedTribe))][index].Contacts.Contact); l++ {
					RSIPNodes.Contact = append(RSIPNodes.Contact, uc.NodeBlock.RSIPList[strconv.Itoa(int(req.Target.SelectedTribe))][index].Contacts.Contact[l])
				}
			}
		}

		fmt.Println("RSIP NODES----->>>>>>", len(RSIPNodes.Contact))

		// getListOfRSIPNodes = nil

		clients := api.Clients{}
		response, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(uc.Node.NodeID[:])+"_Clients"), nil)

		proto.Unmarshal(response, &clients)

		response = nil

		start := time.Now()
		// Verify batch usign selected oracle group public key
		hash := sha256.Sum256(req.Data)
		// keyInByte := "bls_oracle_group_" + req.Param + "_public"
		keyInByte := uc.NodeBlock.OraclesPublicKey[req.Param]

		verified := VerifyBLSBatchSignature(uc.Node, hash, req.Signature, keyInByte)

		// if ShowTime == true && req.Target.SelectedTribe == 0 && req.Index == 0 {
		if ShowTime == true {
			Log(transactionRequest.ID, "Verify signature by RSIP alpha recived from oracle took :", time.Since(start).Seconds())
		}

		if verified != "" {
			// fmt.Println("Failed to verify signature__________________________________________________________________________ RSIP ALPHA")
		} else {
			// if Debug == true {
			// fmt.Println("RSIP ALPHA VERIFIED")
			// }

			start2 := time.Now()
			for index := 0; index < len(transactionRequest.Data); index++ {
				voteResponse := api.VoteResponse{
					ID:            uc.Node.NodeID,
					Address:       req.Target.Address,
					TransactionID: transactionRequest.Data[index].ID,
				}
				verifiedBalance := VerifyAmount(uc.Node, *transactionRequest.Data[index], clients)
				if verifiedBalance != "" {
					voteResponse.Vote = "NO"
					voteResponse.Reason = verifiedBalance
				} else {
					voteResponse.Vote = "YES"
				}

				transactionRequest.Data[index].RSIP.VoteResponse = append(transactionRequest.Data[index].RSIP.VoteResponse, &voteResponse)
				voteResponse = api.VoteResponse{}
			}

			// if ShowTime == true && req.Target.SelectedTribe == 0 && req.Index == 0 {
			if ShowTime == true {
				Log(transactionRequest.ID, "Vote by single RSIP alpha on Transactions after verification took :", time.Since(start2).Seconds())
			}

			thresholdOfRSIP, _ := strconv.Atoi(os.Getenv("THRESHOLD_OF_RSIPS"))
			// fmt.Println(len(RSIPNodes.Contact), thresholdOfRSIP)
			// Generate Share of Aggregates Signature by Batch Proposer
			memberIds := rand.Perm(len(RSIPNodes.Contact))[:thresholdOfRSIP]

			start3 := time.Now()

			paraBytes, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"bls_params"), nil)
			params, _ := ParamsFromBytes(paraBytes)

			pairing := GenPairing(params)
			sysBytes, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"bls_system"), nil)
			system, _ := SystemFromBytes(pairing, sysBytes)

			shares := []Signature{}

			key := "bls_RSIP_group_secret"
			transactionObjForBootNode, _ := proto.Marshal(transactionRequest)
			//newHash := sha256.Sum256(transactionObjForBootNode)

			shares = append(shares, GenerateShares(uc.Node, hash, key, system))

			// if ShowTime == true && req.Target.SelectedTribe == 0 && req.Index == 0 {
			if ShowTime == true {
				Log(transactionRequest.ID, "Generate share by single RSIP alpha took :", time.Since(start3).Seconds())
			}

			start4 := time.Now()

			betaRSIPResponseChan := make(chan []byte)

			selectedOracleGroup, _ := strconv.Atoi(req.Param)
			request := api.Request{
				Action:    "Transaction-To-Proceed",
				Data:      transactionObjForBootNode,
				Param:     "RSIP-NODE",
				RSIPGroup: int32(selectedOracleGroup),
				Signature: req.Signature,
				Hash:      hash[:],
			}

			for j := 0; j <= len(RSIPNodes.Contact)-1; j++ {
				if RSIPNodes.Contact[j].Address != req.Target.Address {
					go BootstrapSync(*req.Target, RSIPNodes.Contact[j], request, betaRSIPResponseChan, nil, uc.Node)
				}

			}

			betaOraclesCount := len(RSIPNodes.Contact) - 1

		OutsideLoopSecond:
			for {
				response := <-betaRSIPResponseChan
				betaOraclesCount--
				signShareFromOtherLeaders, _ := system.SigFromBytes(response)

				if len(shares) < thresholdOfRSIP {
					shares = append(shares, signShareFromOtherLeaders)
				}
				if betaOraclesCount == 0 {
					if ShowTime == true && req.Target.SelectedTribe == 0 && req.Index == 0 {
						Log(transactionRequest.ID, "Response after all other RSIP alpha and beta's took :", time.Since(start4).Seconds())
					}
					go func() {
						for elem := range betaRSIPResponseChan {
							if Debug == true {

								fmt.Println(elem)
							}
						}
						close(betaRSIPResponseChan)
					}()
					break OutsideLoopSecond
				}
			}

			start5 := time.Now()
			// Genrate aggregated signature by batch proposer
			signature, err := Threshold(shares, memberIds, system)

			for i := 0; i < len(shares); i++ {
				shares[i].Free()
			}
			if err != nil {
				fmt.Println(err, "Signature Generation Error")
			}
			// if ShowTime == true && req.Target.SelectedTribe == 0 && req.Index == 0 {
			if ShowTime == true {
				Log(transactionRequest.ID, "Generate signature by RSIP alpha took :", time.Since(start5).Seconds())
			}

			signatureInBytes := system.SigToBytes(signature)

			// shareToBytes := system.SigToBytes(signature)

			RSIPResponse := api.RSIPResponse{
				Hash:      hash[:],
				Signature: signatureInBytes,
				BatchID:   transactionRequest.ID,
			}
			RSIPResponseInBytes, _ := proto.Marshal(&RSIPResponse)

			if int(req.Index) == 0 {
				start := time.Now()
				firstRSIPalphaChan := make(chan []byte)

				// To all Oracles
				// listOfOraclesFromDb := GetValue(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_Epoch_1_Oracle_Group_"+req.Param))
				listOfOracles := uc.NodeBlock.OracleList[req.Param]

				// sent batch to all oracles with RSIP signature
				requestForOracles := api.Request{
					Action:              "Transaction-To-Proceed-In-All-Oracles",
					Data:                RSIPResponseInBytes,
					Param:               req.Param,
					LeaderGroup:         req.LeaderGroup,
					BatchProposerLeader: req.BatchProposerLeader,
				}

				for j := 0; j <= len(listOfOracles.Contact)-1; j++ {
					requestForOracles.RSIPGroup = int32(j) // To send index to Transaction-To-Proceed-In-All-Oracles
					listOfOracles.Contact[j].SelectedTribe = int32(req.Target.SelectedTribe)
					listOfOracles.Contact[j].SelectedNumber = int32(req.RSIPGroup)
					requestForOracles.Index = int32(j)
					go BootstrapSync(*req.Target, listOfOracles.Contact[j], requestForOracles, firstRSIPalphaChan, nil, uc.Node)

				}

				responseFromOracle := 1
			OuterLoopThird:
				for {
					response := <-firstRSIPalphaChan
					responseFromOracle--

					if response != nil {
						responseFromOracle = 0
					}

					if Debug == true {
						fmt.Println(response)
					}
					if responseFromOracle == 0 {
						// fmt.Println("Want print last----------------------------")

						go func() {
							for elem := range firstRSIPalphaChan {
								if Debug == true {

									fmt.Println(elem)
								}
							}
							close(firstRSIPalphaChan)
						}()

						res.Data = []byte("Done")

						transactionObjForBootNode = nil
						RSIPResponseInBytes = nil
						system.Free()
						pairing.Free()
						params.Free()
						// signature.Free()

						break OuterLoopThird
					}
				}

				if ShowTime == true {
					Log(transactionRequest.ID, "Single RSIP alpha process further (alpha RSIP => all Oracle, Oracles => Leaders) and send back to batch proposer took :", time.Since(start).Seconds())
				}
			}

		}

		RSIPNodes = nil
		clients = api.Clients{}

		// if ShowTime == true {
		// 	Log(transactionRequest.ID, "Signature generation in RSIP group", time.Since(start8))
		// 	Log(transactionRequest.ID, "single alpha RSIP with other alpha and beta RSIP votes took", time.Since(start))
		// }

	}

	/*
	 Calls on all ORACLES
	*/
	if req.Action == "Transaction-To-Proceed-In-All-Oracles" {

		// To verify signature of RSIP and send it to ALPHA oracles
		RSIPResponse := api.RSIPResponse{}
		proto.Unmarshal(req.Data, &RSIPResponse)

		start := time.Now()

		hash := RSIPResponse.Hash
		signature := RSIPResponse.Signature
		batchId := RSIPResponse.BatchID

		hashIn32 := [32]byte{}
		for index := 0; index < len(hash); index++ {
			hashIn32[index] = hash[index]
		}

		keyToStoreVerifiedRSIPGroup := "_batch_" + batchId + "_RSIP_GROUP_" + strconv.Itoa(int(req.Target.SelectedTribe))
		// keyInByte := "bls_tribe_" + strconv.Itoa(int(req.Target.SelectedTribe)) + "_RSIP_group_" + strconv.Itoa(int(req.Target.SelectedNumber))
		keyInByte := []byte{}
		for _, v := range uc.NodeBlock.RSIPPublicKey[strconv.Itoa(int(req.Target.SelectedTribe))] {
			if v.Group == strconv.Itoa(int(req.Target.SelectedNumber)) {
				keyInByte = v.PublicKey
			}
		}

		verified := VerifyBLSBatchSignature(uc.Node, hashIn32, signature, keyInByte)

		if ShowTime == true {
			Log(RSIPResponse.BatchID, "Verify signature by oracle received from RSIP (Tribe-"+strconv.Itoa(int(req.Target.SelectedTribe))+")in back process took :", time.Since(start).Seconds())
		}

		paraBytes, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"bls_params"), nil)
		params, _ := ParamsFromBytes(paraBytes)

		pairing := GenPairing(params)
		sysBytes, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"bls_system"), nil)
		system, _ := SystemFromBytes(pairing, sysBytes)

		if verified != "" {
			// fmt.Println("Failed to verify signature__________________________________________________________________________ All Oracles")
		} else {

			// fmt.Println("VERIFIED IN ALL ORACLES")
			uc.Node.ValuesDB.Put([]byte(hex.EncodeToString(uc.Node.NodeID[:])+keyToStoreVerifiedRSIPGroup), []byte("verified"), nil)

			uc.Node.ValuesDB.Put([]byte(hex.EncodeToString(uc.Node.NodeID[:])+keyToStoreVerifiedRSIPGroup+"signature"), signature, nil)

			results := []string{}
			signaturesInBytes := api.Shares{}
			for index := 0; index < (5); index++ {
				keyToStoreVerifiedRSIPGroup := "_batch_" + batchId + "_RSIP_GROUP_" + strconv.Itoa(index)
				data, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(uc.Node.NodeID[:])+keyToStoreVerifiedRSIPGroup), nil)

				signatureInBytes := api.Share{}
				bytes, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(uc.Node.NodeID[:])+keyToStoreVerifiedRSIPGroup+"signature"), nil)
				signatureInBytes.Share = bytes

				if string(data) == "verified" {
					results = append(results, string(data))
					signaturesInBytes.Share = append(signaturesInBytes.Share, &signatureInBytes)
				}

			}

			if len(results) == (5) {
				fmt.Println("Got All Results", len(signaturesInBytes.Share))

				allOracleChan := make(chan []byte)

				//start := time.Now()

				key := "bls_oracle_group_secret"

				share := GenerateShares(uc.Node, hashIn32, key, system)

				// if ShowTime == true {
				// 	Log(RSIPResponse.BatchID, "Generate share by single Oracle in back process took :", time.Since(start).Seconds())
				// }

				// listOfOraclesFromDb := GetValue(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_Epoch_1_Oracle_Group_"+req.Param))
				// listOfOracles := api.Contacts{}
				// proto.Unmarshal(listOfOraclesFromDb, &listOfOracles)

				listOfOracles := uc.NodeBlock.OracleList[req.Param]

				requestForOracles := api.Request{
					Action:               "Transaction-To-Proceed-In-Alpha-Oracles-On-Back-Process",
					Param:                req.Param,
					Hash:                 hashIn32[:],
					Signature:            system.SigToBytes(share),
					BatchID:              batchId,
					RSIPGroup:            req.RSIPGroup,
					LeaderGroup:          req.LeaderGroup,
					BatchProposerLeader:  req.BatchProposerLeader,
					AggregatedSignatures: &signaturesInBytes,
				}

				for j := 0; j <= len(listOfOracles.Contact)-1; j++ {
					if listOfOracles.Contact[j].Category == "alpha" {
						requestForOracles.Index = int32(j) // To send index to Transaction-To-Proceed-In-All-Oracles
						listOfOracles.Contact[j].SelectedTribe = req.Target.SelectedTribe
						go BootstrapSync(*req.Target, listOfOracles.Contact[j], requestForOracles, allOracleChan, nil, uc.Node)
					}

				}

				responseFromOracle := 1
			OuterLoopFourth:
				for {
					response := <-allOracleChan
					responseFromOracle--

					if response != nil {
						//fmt.Println(responseFromOracle, "responseFromOracle, Transaction-To-Proceed-In-Alpha-Oracles-On-Back-Process")
						responseFromOracle = 0
					}

					if Debug == true {
						fmt.Println(response)
					}
					if responseFromOracle == 0 {

						go func() {
							for elem := range allOracleChan {
								if Debug == true {

									fmt.Println(elem)
								}
							}
							close(allOracleChan)
						}()

						res.Data = []byte("Done")
						break OuterLoopFourth
					}
				}

				listOfOracles = &api.Contacts{}
				system.Free()
				pairing.Free()
				params.Free()

			}

			// if Debug == true {
			// fmt.Println("RSIP ALPHA VERIFIED")
			// }
		}

		RSIPResponse = api.RSIPResponse{}

		req.Data = nil
	}

	/*
	 Calls on alpha oracles in back process,
	 Once batch returns from the RSIP alpha,
	 This will verify the shares of RSIP alpha
	*/
	if req.Action == "Transaction-To-Proceed-In-Alpha-Oracles-On-Back-Process" {

		// Alpha oracle will gether other alphas and betas partial signatures
		// Then It will be sent to all leaders from one alpha oracle who gether the first
		start := time.Now()

		totalOracles, _ := strconv.Atoi(os.Getenv("NUMBER_OF_ORACLES"))
		totalOracleGroups, _ := strconv.Atoi(os.Getenv("NUMBER_OF_ORACLE_GROUPS"))
		numberOfOracles := (totalOracles / (totalOracleGroups + 1))
		thresholdOfOracle, _ := strconv.Atoi(os.Getenv("THRESHOLD_OF_ORACLES"))

		sharesInBytes := [][]byte{}
		keyToStoreVerifiedRSIPGroup := "_batch_" + req.BatchID + "_RSIP_GROUP_Share_Alpha_Oracle_" + strconv.Itoa(int(req.RSIPGroup)) + strconv.Itoa(int(req.Index))
		uc.Node.ValuesDB.Put([]byte(hex.EncodeToString(req.Target.ID[:])+keyToStoreVerifiedRSIPGroup), req.Signature, nil)

		for index := 0; index < thresholdOfOracle; index++ {
			keyToStoreVerifiedRSIPGroup := "_batch_" + req.BatchID + "_RSIP_GROUP_Share_Alpha_Oracle_" + strconv.Itoa(int(index)) + strconv.Itoa(int(req.Index))
			getShare, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+keyToStoreVerifiedRSIPGroup), nil)
			if getShare != nil {
				if len(sharesInBytes) <= thresholdOfOracle {
					sharesInBytes = append(sharesInBytes, getShare)
				}
			}
		}
		// if ShowTime == true && req.Index == 0 && req.Target.SelectedTribe == 0 {
		if ShowTime == true {
			Log(req.BatchID, "Generate share by all other alpha Oracle in back process took  :", time.Since(start).Seconds())
		}

		// keyToStoreVerifiedRSIPGroup = "_batch_processed_in_alpha_oracle"
		// isProccessed, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+keyToStoreVerifiedRSIPGroup), nil)

		if len(sharesInBytes) == thresholdOfOracle {
			if req.Index == 0 {
				start2 := time.Now()

				//fmt.Println("GOT ALL SHARES IN ORACLE ALPHA", sharesInBytes)
				listOfLeadersChan := make(chan []byte)

				// Generate Share of Aggregates Signature by Batch Proposer
				memberIds := rand.Perm(numberOfOracles)[:thresholdOfOracle]
				paraBytes, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"bls_params"), nil)
				params, _ := ParamsFromBytes(paraBytes)

				pairing := GenPairing(params)
				sysBytes, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"bls_system"), nil)
				system, _ := SystemFromBytes(pairing, sysBytes)

				shares := []Signature{}
				for index := 0; index < len(sharesInBytes); index++ {
					partialShares, _ := system.SigFromBytes(sharesInBytes[index])
					shares = append(shares, partialShares)
				}

				signature, err := Threshold(shares, memberIds, system)

				for i := 0; i < len(sharesInBytes); i++ {
					shares[i].Free()
				}

				if ShowTime == true {
					Log(req.BatchID, "Generate signature by alpha oracle after retrieving required shares took  :", time.Since(start2).Seconds())
				}
				if err != nil {
					fmt.Println(err, "Signature Generation Error")
				}

				signatureInBytes := system.SigToBytes(signature)

				//fmt.Println("+++++++++++++++++++++++++++=START")
				// Transaction to Leaders
				listOfLeaders := &api.Contacts{}
				// listOfLeaders := GetValuesById(uc.Node, []byte(hex.EncodeToString(uc.Node.NodeID[:])+"_Leaders_Epoch_1_Group_"+strconv.Itoa(int(req.LeaderGroup))))
				listOfLeaders.Contact = append(uc.NodeBlock.LeaderList[strconv.Itoa(int(req.LeaderGroup))].Contact[:0:0], uc.NodeBlock.LeaderList[strconv.Itoa(int(req.LeaderGroup))].Contact...)

				request := api.Request{
					Action:               "Transaction-To-Proceed-In-All-Leaders",
					Signature:            signatureInBytes,
					Hash:                 req.Hash,
					LeaderGroup:          req.LeaderGroup,
					BatchID:              req.BatchID,
					Param:                req.Param,
					BatchProposerLeader:  req.BatchProposerLeader,
					AggregatedSignatures: req.AggregatedSignatures,
				}

				for i := 0; i <= len(listOfLeaders.Contact)-1; i++ {
					request.Index = int32(i)
					go BootstrapSync(*req.Target, listOfLeaders.Contact[i], request, listOfLeadersChan, nil, uc.Node)
				}

				responseFromLeader := len(listOfLeaders.Contact)
			OuterLoopFifth:
				for {
					response := <-listOfLeadersChan
					responseFromLeader--

					if Debug == true {
						fmt.Println(response)
					}
					if responseFromLeader == 0 {

						//fmt.Println("+++++++++++++++++++++++++++=END============")
						go func() {
							for elem := range listOfLeadersChan {
								if Debug == true {

									fmt.Println(elem)
								}
							}
							close(listOfLeadersChan)
						}()
						listOfLeaders = &api.Contacts{}
						res.Data = []byte("Done")

						system.Free()
						pairing.Free()
						params.Free()
						// signature.Free()

						break OuterLoopFifth
					}
				}

			}
			// if isProccessed == nil {
			// 	listOfOraclesFromDb := GetValue(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_Epoch_1_Oracle_Group_"+req.Param))
			// 	listOfOracles := api.Contacts{}
			// 	proto.Unmarshal(listOfOraclesFromDb, &listOfOracles)

			// 	request := api.Request{
			// 		Action: keyToStoreVerifiedRSIPGroup,
			// 		Data:   []byte("yes"),
			// 	}

			// 	for j := 0; j <= len(listOfOracles.Contact)-1; j++ {
			// 		if listOfOracles.Contact[j].Category == "alpha" {
			// 			go BootstrapSync(*req.Target, listOfOracles.Contact[j], request, nil, nil)
			// 		}
			// 	}

			// 	fmt.Println("PROCCED IN ALPHA ORACLE", req.Target.Address)
			// }

		}

		// if getShare != nil {
		// 	sharesInBytes = append(sharesInBytes, getShare)
		// }

	}

	/*
	 Calls on leader in back process,
	 Once batch returns from the oracle alpha,
	 This will verify the shares of oracles
	*/
	if req.Action == "Transaction-To-Proceed-In-All-Leaders" {

		start := time.Now()

		// keyInByte := "bls_oracle_group_" + req.Param + "_public"
		keyInByte := uc.NodeBlock.OraclesPublicKey[req.Param]

		hashIn32 := [32]byte{}
		for index := 0; index < len(req.Hash); index++ {
			hashIn32[index] = req.Hash[index]
		}
		verified := VerifyBLSBatchSignature(uc.Node, hashIn32, req.Signature, keyInByte)

		// if ShowTime == true && req.Index == 0 {
		if ShowTime == true {
			Log(req.BatchID, "Verify signature by leader recived from oracle in back process took :", time.Since(start).Seconds())
		}

		leaderChan := make(chan []byte)

		if verified != "" {
			// fmt.Println("Failed to verify signature__________________________________________________________________________ All Oracles")
		} else {
			// fmt.Println("Verified In Leader", req.Target.Address)

			start2 := time.Now()

			paraBytes, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"bls_params"), nil)
			params, _ := ParamsFromBytes(paraBytes)

			pairing := GenPairing(params)
			sysBytes, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"bls_system"), nil)
			system, _ := SystemFromBytes(pairing, sysBytes)
			key := "bls_leader_group_secret"

			share := GenerateShares(uc.Node, hashIn32, key, system)

			if ShowTime == true && req.Index == 0 {
				Log(req.BatchID, "Generate share by single leader in back process took :", time.Since(start2).Seconds())
			}

			listOfLeaders := &api.Contacts{}
			listOfLeaders.Contact = append(uc.NodeBlock.LeaderList[strconv.Itoa(int(req.LeaderGroup))].Contact[:0:0], uc.NodeBlock.LeaderList[strconv.Itoa(int(req.LeaderGroup))].Contact...)

			// listOfLeaders := GetValuesById(uc.Node, []byte(hex.EncodeToString(uc.Node.NodeID[:])+"_Leaders_Epoch_1_Group_"+strconv.Itoa(int(req.LeaderGroup))))

			// start3 := time.Now()

			request := api.Request{
				Action:               "Transaction-To-Batch-Proposer",
				Signature:            system.SigToBytes(share),
				Hash:                 req.Hash,
				LeaderGroup:          req.LeaderGroup,
				BatchID:              req.BatchID,
				Param:                req.Param,
				Index:                req.Index,
				AggregatedSignatures: req.AggregatedSignatures,
			}
			for index := 0; index < len(listOfLeaders.Contact); index++ {
				if listOfLeaders.Contact[index].Address == req.BatchProposerLeader {
					go BootstrapSync(*req.Target, listOfLeaders.Contact[index], request, leaderChan, nil, uc.Node)
				}

			}

			responseFromLeader := 1
		OuterLoopSixth:
			for {
				response := <-leaderChan
				responseFromLeader--

				if Debug == true {
					fmt.Println(response)
				}
				if responseFromLeader == 0 {
					// if ShowTime == true {
					// 	Log(req.BatchID, "Generate share and store by all leaders in back process took :", time.Since(start3).Seconds())
					// }

					go func() {
						for elem := range leaderChan {
							if Debug == true {

								fmt.Println(elem)
							}
						}
						close(leaderChan)
					}()
					listOfLeaders = &api.Contacts{}
					res.Data = []byte("Done")
					share.Free()
					system.Free()
					pairing.Free()
					params.Free()
					break OuterLoopSixth
				}
			}

		}

	}

	/*
	  Stored all leaders shares on batch Proposer
	*/
	if req.Action == "Transaction-To-Batch-Proposer" {
		// fmt.Println("GOT SHARES AT BATCH PROPOSER", req.Signature)
		leaderShares := "Leader_shares_" + req.BatchID + strconv.Itoa(int(req.Index))

		RSIpSignatures := "RSIP_Groups_Signatures" + req.BatchID

		signaturesInBytes, _ := proto.Marshal(req.AggregatedSignatures)

		uc.Node.ValuesDB.Put([]byte(hex.EncodeToString(req.Target.ID[:])+leaderShares), req.Signature, nil)
		uc.Node.ValuesDB.Put([]byte(hex.EncodeToString(req.Target.ID[:])+RSIpSignatures), signaturesInBytes, nil)
		uc.Node.ValuesDB.Put([]byte(hex.EncodeToString(req.Target.ID[:])+"_HASH"), req.Hash, nil)

		res.Data = []byte("Done")

		// else {
		// 	sharesInBytes := [][]byte{}

		// 	//listOfLeaders := GetValuesById(uc.Node, []byte(hex.EncodeToString(uc.Node.NodeID[:])+"_Leaders_Epoch_1_Group_"+strconv.Itoa(int(req.LeaderGroup))))

		// 	for j := 0; j < 5; j++ {
		// 		leaderShares := "Leader_shares" + strconv.Itoa(j)
		// 		getShare, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+leaderShares), nil)
		// 		// if getShare != nil {
		// 		// 	sharesInBytes = append(sharesInBytes, getShare)
		// 		// }

		// 	}
		// }

	}

	/*
	  Update clients on network
	*/
	if req.Action == "Store-Updated-Client-Across-The-Network" {
		err := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_Clients"), req.Data)
		if err != nil {
			fmt.Println(err)
		}
	}

	/*
	  Store transactions accross the network
	*/
	if req.Action == "Store-Transaction-Across-The-Network" {

		// var latestTransactions []interface{}

		// json.Unmarshal(req.Data, &latestTransactions)
		// fmt.Println("length at second", len(latestTransactions))

		// start := time.Now()

		// Mdb.BulkAdd(latestTransactions)

		// // Log(latestTransactions[0].BatchID, "Store updated client all nodes took :", len(latestTransactions))
		// // fmt.Println(latestTransactions)
		// // res.Data = []byte("abc")
		// fmt.Println("Store batch at single node", time.Since(start))

		start := time.Now()
		err := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_Transactions_"+req.Param), req.Data)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println("Store batch at single node", time.Since(start))
	}

	return &res, nil
}

/*
  This function takes
  total number of chunks
  size of batch in bytes
  chunks
  And return the constructed batch using reedsolomon
*/
func ReconstructBatch(lenChunks, byteSize int, chunks []ChunkWithId) (reconstructedBatch []interface{}) {

	var data [][]byte
	for j := 0; j < lenChunks; j++ {

		length1 := len(data)

		for i := 0; i < len(chunks); i++ {

			if (j + 1) == int(chunks[i].ID) {
				data = append(data, chunks[i].Data)
			}
		}

		length2 := len(data)

		if length1 == length2 {
			data = append(data, nil)
		}

	}

	var enc reedsolomon.Encoder

	enc, _ = reedsolomon.New(3, 1)

	err := enc.ReconstructData(data)
	CheckErr(err)

	file, _ := os.Create("erasure.json")
	err, response := enc.Join(file, data, byteSize)

	json.Unmarshal(response, &reconstructedBatch)

	fmt.Println("\n BATCH:", len(reconstructedBatch))
	return
}
