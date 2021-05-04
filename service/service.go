package service

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/joho/godotenv"
	"github.com/unity-go/api"
	. "github.com/unity-go/dkg"
	. "github.com/unity-go/findValues"
	. "github.com/unity-go/unityCoreRPC"
	. "github.com/unity-go/unityNode"
	. "github.com/unity-go/util"

	. "github.com/dfinity/go-dfinity-crypto/groupsig"
	guuid "github.com/google/uuid"
)

// Server represents the gRPC server
// type Server struct {
// }

type UnityCore2 struct {
	Node *UnityNode
}

func (uc *UnityCore2) SayHello(ctx context.Context, in *api.PingMessage) (*api.PingMessage, error) {

	log.Printf("Receive message %s", in.Greeting)

	return &api.PingMessage{Greeting: "bar"}, nil
}

func (uc *UnityCore2) Sync(ctx context.Context, request *api.Request) (*api.BootstarpSyncResponse, error) {

	req := request
	res := api.BootstarpSyncResponse{}

	value := GetValuesById(uc.Node, []byte(string(req.Target.ID[:])))

	value.Contact = append(value.Contact, req.Sender)
	data := Unique(&value)

	sender, _ := proto.Marshal(data)
	errorFromGet := PutValuesById(uc.Node, []byte(string(req.Target.ID[:])), sender)
	if errorFromGet != nil {
		fmt.Println(errorFromGet)
	}

	res.Contacts = data
	// resInBytes := PrepareResponse(data, nil)

	return &res, nil

}

func PrepareRequest(data []byte) api.Request {
	req := api.Request{}
	proto.Unmarshal(data, &req)

	// log.Println("Receive message", req)

	return req
}

func PrepareResponse(contacts *api.Contacts, data []byte) []byte {
	res := api.BootstarpSyncResponse{
		Contacts: contacts,
		Data:     data,
	}
	resInBytes, _ := proto.Marshal(&res)

	return resInBytes
}

func (uc *UnityCore2) SyncAction(ctx context.Context, request *api.Request) (*api.BootstarpSyncResponse, error) {

	req := request
	res := api.BootstarpSyncResponse{}

	if req.Action == "store-data-in-leaders" {

		transactionsQueueFromDB := &api.TransactionWithSignatures{}
		upendData := &api.TransactionWithSignatures{}

		// response, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"_tx_queue"), nil)

		// err := proto.Unmarshal(response, transactionsQueueFromDB)
		// if Debug == true {
		// 	fmt.Println("errr", err)
		// }
		proto.Unmarshal(req.Data, upendData)

		transactionsQueueFromDB.TransactionWithSignature = append(transactionsQueueFromDB.TransactionWithSignature, upendData.TransactionWithSignature...)

		transactionQueueByte, _ := proto.Marshal(transactionsQueueFromDB)

		if req.TransactionType == "manual" {
			uc.Node.ValuesDB.Put([]byte(hex.EncodeToString(req.Target.ID[:])+"_tx_queue_"+req.Param), transactionQueueByte, nil)
		} else {
			uc.Node.ValuesDB.Put([]byte(hex.EncodeToString(req.Target.ID[:])+"_tx_queue"), transactionQueueByte, nil)
		}

		fmt.Println("Stored")
		res.Data = []byte("stored")
		// return nil, nil

	}

	// Election Process
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

	if strings.Contains(req.Action, "dkg_Secret_Oracles") {
		errorFromSave := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"dkg_Secret_Oracles"), req.Data)
		if errorFromSave != nil {
			fmt.Println(errorFromSave)
		}
	}

	if strings.Contains(req.Action, "dkg_Public_Oracles") {
		err := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"dkg_Public_Oracles"), req.Data)
		if err != nil {
			fmt.Println(err)
		}
	}

	if strings.Contains(req.Action, "_Epoch_1_Oracle_Group_") {
		err := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+req.Action), req.Data)
		if err != nil {
			fmt.Println(err)
		}
	}

	if strings.Contains(req.Action, "_Tribe_Group_") {
		err := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+req.Action), req.Data)
		if err != nil {
			fmt.Println(err)
		}
	}

	if req.Action == "Leaders-List-Save" {
		errorFromSave := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_"+req.Param), req.Data)
		if errorFromSave != nil {
			fmt.Println(errorFromSave)
		}
	}

	if req.Action == "Shared-Secret-Key-Leader" {
		errorFromSave := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+req.Param), req.Data)
		if errorFromSave != nil {
			fmt.Println(errorFromSave)
		}
	}

	if strings.Contains(req.Action, "_Decimal_DKG") {
		errorFromSave := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+req.Action), req.Data)
		if errorFromSave != nil {
			fmt.Println(errorFromSave)
		}
	}

	if strings.Contains(req.Action, "-Tribe_") {
		err := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+req.Action), req.Data)
		if err != nil {
			fmt.Println(err)
		}
	}

	if strings.Contains(req.Action, "dkg_Secret_RSIP_Group") {
		err := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+req.Action), req.Data)
		if err != nil {
			fmt.Println(err)
		}
	}

	if strings.Contains(req.Action, "dkg_Public_RSIP_Group_") {
		err := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+req.Action), req.Data)
		if err != nil {
			fmt.Println(err)
		}
	}

	if req.Action == "Shared-Public-Key-Leader" {
		err := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+req.Param), req.Data)
		if err != nil {
			fmt.Println(err)
		} else {
			if Debug == true {
				fmt.Println("key saved", req.Target.ID, req.Target.Address)
			}
		}
	}

	if req.Action == "Retrieve-RSIP-Group-Seckey" {

		key := ""
		json.Unmarshal(req.Data, &key)

		secKeyInBytes, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+key), nil)

		dkgSecKey := &Seckey{}
		dkgSecKey.SetHexString(string(secKeyInBytes))

		decimalKey := dkgSecKey.GetDecimalString()

		if req.Param == "secret-key" {
			res.Data = secKeyInBytes
		} else {
			res.Data, _ = json.Marshal(decimalKey)
		}
	}

	if req.IsDKG == true {
		key := ""

		if req.Action == "Leaders" {
			key = hex.EncodeToString(req.Target.ID[:]) + "dkg_Secret_Leaders"
		} else if req.Action == "Oracles" {
			key = hex.EncodeToString(req.Target.ID[:]) + "dkg_Secret_Oracles"
		} else {
			key = hex.EncodeToString(req.Target.ID[:]) + req.Action
		}

		getSecretKey, _ := uc.Node.ValuesDB.Get([]byte(key), nil)

		dkgSecretShare := AggregateSeckeysForDKG(getSecretKey, req.Data)
		errorFromGet := PutValuesById(uc.Node, []byte(key), []byte(dkgSecretShare))
		if errorFromGet != nil {
			fmt.Println(errorFromGet)
		}

		resInBytes, _ := json.Marshal(dkgSecretShare)
		res.Data = resInBytes
	}

	// Transactions
	if req.Action == "Start-Queued-Batch" {
		start := time.Now()
		leaderGroupNumber, _ := strconv.Atoi(req.Param)

		transactionType := string(req.TransactionType)

		// fmt.Println("\nSTART QUEUE function------------------------------------------>", req.Target.Address)

		transactionsQueueFromDB, singleBatch, remainingBatch := api.TransactionWithSignatures{}, api.TransactionWithSignatures{}, api.TransactionWithSignatures{}

		batchChannel := make(chan api.BatchTransaction)

		batchResponses := api.BatchTransactions{}

		transactionToken := req.TransactionToken
		response := []byte{}
		if transactionType == "auto" {
			response, _ = uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"_tx_queue"), nil)
		} else {
			response, _ = uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"_tx_queue_"+transactionToken), nil)
		}
		proto.Unmarshal(response, &transactionsQueueFromDB)
		// var mutex sync.Mutex
		// mutex.Lock()
		var numberOfBatches int

		if req.NumberOfBatches == 0 {
			numberOfBatches = 1
		} else {
			numberOfBatches = int(req.NumberOfBatches)
		}

		for index := 0; index < numberOfBatches; index++ {

			start2 := time.Now()

			if transactionType == "auto" {
				// retrieve all batch from queue
				lengthOfSingleBatch := int(req.NumberOfTransactionInSingleBatch)

				// retrieve first batch from queue
				if len(transactionsQueueFromDB.TransactionWithSignature) >= lengthOfSingleBatch {
					singleBatch.TransactionWithSignature = append(singleBatch.TransactionWithSignature, transactionsQueueFromDB.TransactionWithSignature[:lengthOfSingleBatch]...)
					remainingBatch.TransactionWithSignature = append(remainingBatch.TransactionWithSignature, transactionsQueueFromDB.TransactionWithSignature[lengthOfSingleBatch:]...)
				}

				transactionsQueueFromDB.TransactionWithSignature = remainingBatch.TransactionWithSignature
			} else {
				fmt.Println(len(transactionsQueueFromDB.TransactionWithSignature), "Len")
				singleBatch.TransactionWithSignature = transactionsQueueFromDB.TransactionWithSignature
			}

			fmt.Println(len(singleBatch.TransactionWithSignature), "Single Batch Len")

			individualBatch := api.BatchTransaction{}
			individualBatch.ID = guuid.New().String()
			individualBatch.Data = singleBatch.TransactionWithSignature

			fmt.Println("INDIVIDUAL BATCH", len(individualBatch.Data))

			dataInBytes, _ := proto.Marshal(&individualBatch)

			if ShowTime == true {
				Log(individualBatch.ID, "Get one batch for one leader from database took :", time.Since(start2).Seconds(), "::size", len(dataInBytes))
			}

			for index := 0; index < len(individualBatch.Data); index++ {
				uuid := guuid.New()
				individualBatch.Data[index].ID = uuid.String()
			}

			singleBatch = api.TransactionWithSignatures{}

			remainingBatch = api.TransactionWithSignatures{}

			// first batch goes for validation process
			go QueueHandleTransaction(individualBatch, uc.Node, req.Target, leaderGroupNumber, batchChannel)

			if transactionType == "auto" && index != (numberOfBatches-1) {

				timeToSleep, _ := strconv.Atoi(string(req.Value))

				time.Sleep(time.Millisecond * time.Duration(timeToSleep+500))

				fmt.Println("Time DUration in tribe", time.Millisecond*time.Duration(timeToSleep+500))
			}

		}

		for {
			select {
			case response := <-batchChannel:
				// vote
				numberOfBatches--
				batchResponses.BatchTransaction = append(batchResponses.BatchTransaction, &response)
				if ShowTime == true {
					batchResponsesInBytesForSize, _ := proto.Marshal(&batchResponses)
					Log(response.ID, "Total time for single batch from LEADER - ORACLES, ORACLES - RSIP, RSIP - ORACLES and ORACLES - LEADERS :", time.Since(start).Seconds(), "::size", len(batchResponsesInBytesForSize))
				}

				uc.Node.ValuesDB.Put([]byte(hex.EncodeToString(req.Target.ID[:])+"_tx_queue"), []byte{}, nil)
			}
			if numberOfBatches == 0 {

				break
			}
		}

		batchResponsesInBytes, _ := proto.Marshal(&batchResponses)

		res.Data = batchResponsesInBytes
	}

	if req.Action == "Transaction-To-Proceed" {

		//	start134 := time.Now()

		voteResponses := api.VoteResponses{}

		transactionRequest := api.BatchTransaction{}

		//start15 := time.Now()

		proto.Unmarshal(req.Data, &transactionRequest)

		if ShowTime == true {
			if req.Param == "RSIP-NODE" {
				// Log(transactionRequest.ID, "Time to unmarshal REQUEST in RSIP", time.Since(start15))
			}
		}
		clients := api.Clients{}
		response, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(uc.Node.NodeID[:])+"_Clients"), nil)

		proto.Unmarshal(response, &clients)

		// start16 := time.Now()
		verified := ""
		if req.Param == "RSIP-NODE" {
			verified = VerifyGroupSignature(uc.Node, transactionRequest, "dkg_Public_Oracles", req.Target.ID)
			// verified = ""
		}
		if ShowTime == true {
			if req.Param == "RSIP-NODE" {
				// Log(transactionRequest.ID, "time to verify signature in RSIP", time.Since(start16))
			}
		}
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
			} else if verified != "" {
				voteResponse.Vote = "NO"
				voteResponse.Reason = verified
			} else {
				voteResponse.Vote = "YES"
			}

			voteResponses.VoteResponse = append(voteResponses.VoteResponse, &voteResponse)

		}

		if ShowTime == true {
			// Log(transactionRequest.ID, "Verify and Vote from single node took", time.Since(start134))
		}

		res.Data, _ = proto.Marshal(&voteResponses)

	}

	if req.Action == "Transaction-To-Validate-And-Check-Threshold-From-Oracles-On-First-Oracle" {

		leaderGroupNumber := req.Param
		start3 := time.Now()

		start := time.Now()
		err := godotenv.Load()
		if err != nil {
			fmt.Println("Error loading .env file")
		}

		if Debug == true {
			fmt.Println("", "First Oracle------------------------------")
		}

		numberOfRSIPGroup := os.Getenv("NUMBER_OF_RSIP_GROUP")
		//	numberOfTribesGroup := os.Getenv("NUMBER_OF_TRIBES")
		numberOfGroup, _ := strconv.Atoi(numberOfRSIPGroup)
		//	numberOfTribes, _ := strconv.Atoi(numberOfTribesGroup)

		transactionRequest := api.BatchTransaction{}

		proto.Unmarshal(req.Data, &transactionRequest)
		verified := VerifyGroupSignature(uc.Node, transactionRequest, "dkg_Public_Leaders_Group_"+leaderGroupNumber, req.Target.ID)
		// verified := ""

		if ShowTime == true {
			fmt.Println("verified +++++++++++++++++++++", verified)
		}

		//start6 := time.Now()
		for k := 0; k < len(transactionRequest.Data); k++ {
			voteResponse := api.VoteResponse{
				ID:            req.Target.ID,
				Address:       req.Target.Address,
				TransactionID: transactionRequest.Data[k].ID,
			}

			threshold := 0
			for j := 0; j <= len(transactionRequest.Data[k].Leaders.VoteResponse)-1; j++ {
				if transactionRequest.Data[k].Leaders.VoteResponse[j].Vote == "YES" {
					threshold++
				}
			}
			voteResponse.Threshold = strconv.Itoa(threshold) + " out of " + strconv.Itoa(len(transactionRequest.Data[k].Leaders.VoteResponse)) + " Leaders"
			transactionRequest.Data[k].Oracles.VoteResponse = append(transactionRequest.Data[k].Oracles.VoteResponse, &voteResponse)

		}

		if ShowTime == true {
			// Log(transactionRequest.ID, "Single Alpha Oracle votes by itself", time.Since(start6))
		}
		start4 := time.Now()
		oraclesFromDB := &api.Contacts{}

		oracles, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"_Epoch_1_Oracle_Group_"+strconv.Itoa(int(req.Target.SelectedNumber))), nil)
		proto.Unmarshal(oracles, oraclesFromDB)

		// if Debug == true {
		// }

		betaOraclesCount := len(oraclesFromDB.Contact) - 1
		betaOraclesResponseChan := make(chan []byte)
		transactionRequestInByteForRSIP, _ := proto.Marshal(&transactionRequest)
		Log(transactionRequest.ID, "Verify and check threshold for leader from first oracle took :", time.Since(start).Seconds(), "::size", len(transactionRequestInByteForRSIP))

		request := api.Request{
			Action: "Transaction-to-beta",
			Data:   transactionRequestInByteForRSIP,
		}

		for j := 0; j <= len(oraclesFromDB.Contact)-1; j++ {

			if !Equal(oraclesFromDB.Contact[j].ID, req.Target.ID) {
				if Debug == true {
					fmt.Println("BETA", j)
				}

				go BootstrapSync(*req.Target, oraclesFromDB.Contact[j], request, betaOraclesResponseChan, nil)

			}

		}
		dataInBytes := []byte{}
		for {
			select {
			case response := <-betaOraclesResponseChan:
				betaOraclesCount--
				voteResponses := api.VoteResponses{}
				dataInBytes = response
				proto.Unmarshal(response, &voteResponses)
				if Debug == true {
				}
				for index := 0; index < len(transactionRequest.Data); index++ {
					for j := 0; j < len(voteResponses.VoteResponse); j++ {
						if transactionRequest.Data[index].ID == voteResponses.VoteResponse[j].TransactionID {
							transactionRequest.Data[index].Oracles.VoteResponse = append(transactionRequest.Data[index].Oracles.VoteResponse, voteResponses.VoteResponse[j])
						}
					}
				}
			}
			if betaOraclesCount == 0 {

				break
			}
		}

		if ShowTime == true {
			Log(transactionRequest.ID, "Each alpha ORACLE getting response from all other ALPHAs and Betas ORACLE took :", time.Since(start4).Seconds(), "::size", len(dataInBytes))
		}

		start5 := time.Now()
		secret := &Seckey{}
		if Debug == true {
			fmt.Println(strings.Replace(transactionRequest.Signature, " ", "", -1))
		}
		secret.SetHexString(strings.Replace(transactionRequest.Signature, "1 ", "", -1))
		if Debug == true {
			fmt.Println("sec.DecimalString: ", secret.GetDecimalString())
		}
		signatureInDecimalString := secret.GetDecimalString()
		signatureInInt, _ := strconv.Atoi(signatureInDecimalString)

		numberOfSelectedGroup := signatureInInt % numberOfGroup

		// RSIPNodes := Contacts{}

		key := hex.EncodeToString(req.Target.ID[:]) + "_Secret"
		getSecretKey, _ := uc.Node.ValuesDB.Get([]byte(key), nil)
		secretKeyOracle := &Seckey{}
		secretKeyOracle.SetHexString(string(getSecretKey))
		if Debug == true {
			fmt.Println("sec.DecimalString: ", secretKeyOracle.GetDecimalString())
		}

		if ShowTime == true {
			Log(transactionRequest.ID, "Select RSIP groups from Tribe by deterministic number took :", time.Since(start5).Seconds())
		}
		//	decimalKey := secretKeyOracle.GetDecimalString()
		//	decimalKeyForSelectTribe, _ := strconv.Atoi(decimalKey)
		//	numberOfSelectedTribe := decimalKeyForSelectTribe % numberOfTribes

		// RSIPUpdateGroup := Contacts{}

		// for index := 0; index < numberOfTribes+1; index++ {

		// 	tribeGroup, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"_Tribe_Group_"+strconv.Itoa(index)), nil)
		// 	proto.Unmarshal(tribeGroup, &RSIPUpdateGroup)

		// 	for j := 0; j < len(RSIPUpdateGroup); j++ {
		// 		Bootstrap(RSIPUpdateGroup[j], req.Target, "to_update_data", nil, nil, "")
		// 	}

		// }

		// start8 := time.Now()

		RSIPVotes := api.VoteResponses{}
		signaturesRSIP := []string{}
		// channelForRSIP := make(ch []byte)
		// newTxBatch.Data = transactionRequest.Data
		ch := make(chan []byte)

		// for j := 0; j < numberOfTribes+1; j++ {
		go TransactionToRSIP(uc.Node, *req, numberOfSelectedGroup, ch, RSIPVotes, signaturesRSIP, int(req.Target.SelectedTribe), transactionRequest, req.Data)

		// }

		RSIPGroupCount := 1
		for {
			select {
			case response := <-ch:
				RSIPGroupCount--
				dataInBytes = response
				voteResponses := api.VoteResponses{}
				proto.Unmarshal(response, &voteResponses)

				for k := 0; k < len(transactionRequest.Data); k++ {
					transactionRequest.Data[k].RSIP.VoteResponse = append(transactionRequest.Data[k].RSIP.VoteResponse, voteResponses.VoteResponse...)
				}

				thresholdOfRSIP := 0

				for index := 0; index < len(transactionRequest.Data); index++ {
					for l := 0; l < len(transactionRequest.Data[index].RSIP.VoteResponse); l++ {
						if transactionRequest.Data[index].RSIP.VoteResponse[l].Vote == "YES" {
							thresholdOfRSIP++
						}
					}
				}

				for index := 0; index < len(transactionRequest.Data); index++ {
					for j := 0; j <= len(transactionRequest.Data[index].Oracles.VoteResponse)-1; j++ {
						transactionRequest.Data[index].Oracles.VoteResponse[j].ThresholdOfRSIP = strconv.Itoa(thresholdOfRSIP) + " out of " + strconv.Itoa(len(transactionRequest.Data[index].RSIP.VoteResponse)) + " RSIP"
					}
				}

			}
			if RSIPGroupCount == 0 {
				// fmt.Printf("%s took %v\n", "Alpha to other beta's and alphas", time.Since(start4))

				break
			}
		}

		if ShowTime == true {
			// Log(transactionRequest.ID, "Total time", time.Since(start8), len(dataInBytes))
		}
		//transactionRequest.SignatureOfRSIP = signaturesRSIP
		transactionBatchImBytes, _ := proto.Marshal(&transactionRequest)
		Log(transactionRequest.ID, "Total time taken by SINGLE alpha ORACLE to proceed(INCLUDED gossiping to other ORACLES and time take by RSIP group for this alpha ORACLE) :", time.Since(start3).Seconds(), "::size", len(dataInBytes))

		res.Data = transactionBatchImBytes

	}

	if req.Action == "Transaction-to-beta" {

		err := godotenv.Load()
		if err != nil {
			fmt.Println("Error loading .env file")
		}

		transactionRequest := api.BatchTransaction{}
		voteResponses := api.VoteResponses{}

		proto.Unmarshal(req.Data, &transactionRequest)

		for k := 0; k < len(transactionRequest.Data); k++ {
			voteResponse := api.VoteResponse{
				ID:            req.Target.ID,
				Address:       req.Target.Address,
				TransactionID: transactionRequest.Data[k].ID,
			}

			threshold := 0
			for j := 0; j <= len(transactionRequest.Data[k].Leaders.VoteResponse)-1; j++ {
				if transactionRequest.Data[k].Leaders.VoteResponse[j].Vote == "YES" {
					threshold++
				}
			}
			voteResponse.Threshold = strconv.Itoa(threshold) + " out of " + strconv.Itoa(len(transactionRequest.Data[k].Leaders.VoteResponse)) + " Leaders"
			voteResponses.VoteResponse = append(voteResponses.VoteResponse, &voteResponse)

		}

		res.Data, _ = proto.Marshal(&voteResponses)

	}

	if req.Action == "Transaction-To-Proceed-In-RSIP" {

		dataInBYtes := []byte{}
		// This service is used to proceed batch in Alpha RSIP, RSIP will send batch to other alpha and

		// start := time.Now()
		// line to add data in channle
		// res.IsTransection = true

		transactionRequest := &api.BatchTransaction{}

		//start15 := time.Now()
		proto.Unmarshal(req.Data, transactionRequest)

		if ShowTime == true {
			// Log(transactionRequest.ID, "Time to unmarshal REQUEST", time.Since(start15))
		}
		RSIPNodes := &api.Contacts{}

		getListOfRSIPNodes, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"-Tribe_"+strconv.Itoa(int(req.Target.SelectedTribe))+"_RSIP_GROUP_"+strconv.Itoa(int(req.Target.SelectedNumber))), nil)
		proto.Unmarshal(getListOfRSIPNodes, RSIPNodes)

		clients := api.Clients{}
		response, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(uc.Node.NodeID[:])+"_Clients"), nil)

		proto.Unmarshal(response, &clients)

		//	start2 := time.Now()
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
		}
		if ShowTime == true {
			// Log(transactionRequest.ID, "Self RSIP alpha votes", time.Since(start2))
		}
		start3 := time.Now()

		//start4 := time.Now()

		//start5 := time.Now()
		transactionRequestInByteForRSIP, _ := proto.Marshal(transactionRequest)
		// transactionRequestInByteForRSIP, _ := proto.Marshal(transactionRequest)

		if ShowTime == true {
			// Log(transactionRequest.ID, "Time to mashal request at RSIP", time.Since(start5))
		}
		betaRSIPResponseChan := make(chan []byte)

		request := api.Request{
			Action: "Transaction-To-Proceed",
			Data:   transactionRequestInByteForRSIP,
			Param:  "RSIP-NODE",
		}

		for j := 0; j <= len(RSIPNodes.Contact)-1; j++ {
			if RSIPNodes.Contact[j].Address != req.Target.Address {
				go BootstrapSync(*req.Target, RSIPNodes.Contact[j], request, betaRSIPResponseChan, nil)
			}

		}

		betaOraclesCount := len(RSIPNodes.Contact) - 1

		voteResponses := api.VoteResponses{}
		for {
			select {
			case response := <-betaRSIPResponseChan:
				dataInBYtes = response
				betaOraclesCount--
				if ShowTime == true {
					// Log(transactionRequest.ID, "SINGLE RSIP RESPONSE", time.Since(start4))
					// fmt.Println(time.Now().Minute(), time.Now().Second(), time.Now(), "After getting RESPONSE OUTSIDE")
				}
				//	start6 := time.Now()
				proto.Unmarshal(response, &voteResponses)
				if ShowTime == true {
					// Log(transactionRequest.ID, "Time to unmashal single RSIP alpha", time.Since(start6))
				}

				//	start7 := time.Now()
				// for index := 0; index < len(transactionRequest.Data); index++ {
				// 	for j := 0; j < len(voteResponses.VoteResponse); j++ {
				// 		if transactionRequest.Data[index].ID == voteResponses.VoteResponse[j].TransactionID {
				// 			transactionRequest.Data[index].RSIP.VoteResponse = append(transactionRequest.Data[index].RSIP.VoteResponse, voteResponses.VoteResponse[j])
				// 		}
				// 	}
				// }
				if ShowTime == true {
					// Log(transactionRequest.ID, "Time to loop on single RSIP alha node", time.Since(start7))
				}
			}
			if betaOraclesCount == 0 {

				break
			}
		}

		if ShowTime == true {
			Log(transactionRequest.ID, "Alpha RSIP to other beta's and alphas RSIP took :", time.Since(start3).Seconds(), "::size", len(dataInBYtes))
		}

		start8 := time.Now()

		key := "_Tribe_" + strconv.Itoa(int(req.Target.SelectedTribe)) + "_dkg_Secret_RSIP_Group_" + strconv.Itoa(int(req.Target.SelectedNumber))

		// fmt.Println(key, "Private")
		sign := GenerateBatchSignature(uc.Node, key, uc.Node.NodeID, *transactionRequest)
		// sign := ""

		transactionRequest.Signature = sign
		res.Data, _ = proto.Marshal(transactionRequest)

		if ShowTime == true {
			Log(transactionRequest.ID, "Signature generation in RSIP group took :", time.Since(start8).Seconds(), "::size", len(res.Data))
			// Log(transactionRequest.ID, "single alpha RSIP with other alpha and beta RSIP votes took", time.Since(start))
		}

	}

	if req.Action == "Transaction-To-Leader-After-Verification-From-Oracle" {
		// uc.Node.Mutex.Lock()
		start := time.Now()
		if ShowTime == true {
			fmt.Println("selected leader node", req.Target.Address)
		}
		clients := api.Clients{}
		// client := Client{}
		transactionRequest := api.BatchTransaction{}
		proto.Unmarshal(req.Data, &transactionRequest)
		contactsFromOwnDb := GetValuesById(uc.Node, []byte(string(req.Target.ID[:])))
		response, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"_Clients"), nil)
		proto.Unmarshal(response, &clients)

		// verified := VerifyGroupSignature(uc.Node, transactionRequest, "dkg_Public_Oracles", req.Target.ID)

		verified := ""
		for k := 0; k < len(transactionRequest.Data); k++ {
			if verified != "" {
				if Debug == true {
					fmt.Println(verified, "not Verified on leader at last")
				}

				transactionRequest.Data[k].Status = "FAIL"
				transactionRequest.Data[k].Reason = []string{verified}
				// transactionInByte, _ := proto.Marshal(transactionRequest)
				// transactionObjInByte, _ := proto.Marshal(transactionRequest)
				// sign := GenerateSignature(uc.Node, transactionObjInByte, "dkg_Secret_Leaders", req.Target.ID, transactionRequest)
				// transactionRequest.Signature = sign.GetHexString()
				transactionRequest.Data[k].Signature = ""

				// for i := 0; i <= len(contactsFromOwnDb)-1; i++ {
				// 	_, _, err := Bootstrap(contactsFromOwnDb[i], req.Target, "Store-Transaction-Across-The-Network", transactionInByte, nil, "")
				// 	if err != nil {
				// 		fmt.Println("Bootstrap error:", err)
				// 	}
				// }

			} else {

				for j := 0; j <= len(transactionRequest.Data[k].Oracles.VoteResponse)-1; j++ {
					checkThreshold := strings.Split(transactionRequest.Data[k].Oracles.VoteResponse[j].Threshold, " ")
					checkThresholdForRSIP := strings.Split(transactionRequest.Data[k].Oracles.VoteResponse[j].Threshold, " ")
					appliedThreshold, _ := strconv.Atoi(checkThreshold[0])
					totalThreshold, _ := strconv.Atoi(checkThreshold[3])
					appliedThresholdForRSIP, _ := strconv.Atoi(checkThresholdForRSIP[0])
					totalThresholdForRSIP, _ := strconv.Atoi(checkThresholdForRSIP[3])
					if appliedThreshold < totalThreshold {
						transactionRequest.Data[k].Reason = append(transactionRequest.Data[k].Reason, "Not achieved enough threshold at Oracle for Leaders "+transactionRequest.Data[k].Oracles.VoteResponse[j].Address)
					}
					if appliedThresholdForRSIP < totalThresholdForRSIP {
						transactionRequest.Data[k].Reason = append(transactionRequest.Data[k].Reason, "Not achieved enough threshold at Oracle For RSIP "+transactionRequest.Data[k].Oracles.VoteResponse[j].Address)
					}
				}

				if len(transactionRequest.Data[k].Reason) == 0 {
					for j := 0; j <= len(clients.Client)-1; j++ {
						if clients.Client[j].Address == transactionRequest.Data[k].From {
							clients.Client[j].Amount = clients.Client[j].Amount - transactionRequest.Data[k].Amount
						}
						if clients.Client[j].Address == transactionRequest.Data[k].To {
							clients.Client[j].Amount = clients.Client[j].Amount + transactionRequest.Data[k].Amount
						}

					}
					transactionRequest.Data[k].Status = "SUCCESS"

					// err := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_Clients"), byteValue)
					// if err != nil {
					// 	fmt.Println(err, "Error")
					// 	return nil
					// }

					// sign := GenerateSignature(uc.Node, transactionInByte, "dkg_Secret_Leaders", req.Target.ID, transactionRequest)
					// transactionRequest.Signature = sign.GetHexString()
					// transactionRequest.Signature = ""
					// transactionInByte, _ := proto.Marshal(transactionRequest)

					// for i := 0; i <= len(contactsFromOwnDb)-1; i++ {
					// 	// if contactsFromOwnDb[i].ID != req.Target.ID {
					// 	_, _, errFromClients := Bootstrap(contactsFromOwnDb[i], req.Target, "Store-Updated-Client-Across-The-Network", byteValue, nil, "")
					// 	if errFromClients != nil {
					// 		fmt.Println("Bootstrap error:", errFromClients)
					// 	}
					// 	// }
					// 	_, _, err := Bootstrap(contactsFromOwnDb[i], req.Target, "Store-Transaction-Across-The-Network", transactionInByte, nil, "")
					// 	if err != nil {
					// 		fmt.Println("Bootstrap error:", err)
					// 	}
					// }
					// fmt.Printf("%s took %v\n", "Update clients and store transaction object across the network", time.Since(start1))

					// res.Data = transactionInByte

				} else {
					transactionRequest.Data[k].Status = "FAIL"
					// transactionInByte, _ := proto.Marshal(transactionRequest)
					// sign := GenerateSignature(uc.Node, transactionInByte, "dkg_Secret_Leaders", req.Target.ID, transactionRequest)
					// transactionRequest.Signature = sign.GetHexString()
					transactionRequest.Data[k].Signature = ""
					// transactionInByte, _ = proto.Marshal(transactionRequest)

					// for i := 0; i <= len(contactsFromOwnDb)-1; i++ {
					// 	_, _, err := Bootstrap(contactsFromOwnDb[i], req.Target, "Store-Transaction-Across-The-Network", transactionInByte, nil, "")
					// 	if err != nil {
					// 		fmt.Println("Bootstrap error:", err)
					// 	}
					// }

					// res.Data = transactionInByte
				}

				if Debug == true {
					// fmt.Printf("%s took %v\n", "Transaction-To-Leader-After-Verification-From-Oracle", time.Since(start))
				}
			}
		}
		// uc.Node.Mutex.Unlock()
		byteValue, _ := proto.Marshal(&clients)

		newBatchResponse := api.BatchTransaction{}
		newBatchResponse.ID = transactionRequest.ID

		for k := 0; k < len(transactionRequest.Data); k++ {
			// transactions, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"_Transactions"), nil)
			// proto.Unmarshal(transactions, &Transactions)
			latestTx := api.TransactionWithSignature{}
			latestTx.ID = transactionRequest.Data[k].ID
			latestTx.From = transactionRequest.Data[k].From
			latestTx.To = transactionRequest.Data[k].To
			latestTx.Amount = int32(transactionRequest.Data[k].Amount)
			latestTx.Status = transactionRequest.Data[k].Status
			latestTx.Signature = transactionRequest.Data[k].Signature
			latestTx.Message = transactionRequest.Data[k].Message

			newBatchResponse.Data = append(newBatchResponse.Data, &latestTx)

			// Transactions = append(Transactions, transactionObject)
		}

		timeForDB := time.Now()
		// Store in mongoDB

		for i := 0; i < len(newBatchResponse.Data); i++ {
			newBatchResponse.Data[i].BatchID = newBatchResponse.ID
			Mdb.Add(*newBatchResponse.Data[i])
		}
		fmt.Println("Time taken by each batch to store data in mongoDB:", time.Since(timeForDB))

		transactionInByte, _ := proto.Marshal(&newBatchResponse)

		if ShowTime == true {
			Log(newBatchResponse.ID, "After Checking amount for batch :", time.Since(start).Seconds(), "::size", len(transactionInByte))
		}
		start0 := time.Now()

		chanName := make(chan []byte)

		start4 := time.Now()

		fmt.Println("lengthOfClients", len(byteValue))
		fmt.Println("lengthOfBatch", len(transactionInByte))

		go func() {
			for i := 0; i <= len(contactsFromOwnDb.Contact)-1; i++ {
				// if contactsFromOwnDb[i].ID != req.Target.ID {

				request := api.Request{
					Action: "Store-Updated-Client-Across-The-Network",
					Data:   byteValue,
				}

				BootstrapSync(*req.Target, contactsFromOwnDb.Contact[i], request, chanName, nil)

				// request2 := api.Request{
				// 	Action: "Store-Transaction-Across-The-Network",
				// 	Data:   transactionInByte,
				// }

				// // }
				// BootstrapSync(*req.Target, contactsFromOwnDb.Contact[i], request2, chanName, nil)
			}
		}()

		count := len(contactsFromOwnDb.Contact)
		for {
			select {
			case response := <-chanName:
				count--
				msg := api.MyData{}
				proto.Unmarshal(response, &msg)
			}
			if count == 0 {
				// fmt.Printf("%s took %v\n", "Alpha to other beta's and alphas", time.Since(start4))

				break
			}
		}

		fmt.Println("STORE TIME", time.Since(start4))

		// if ShowTime == true {
		Log(newBatchResponse.ID, "Store updated client all nodes :", time.Since(start0).Seconds(), "::size", len(transactionInByte))
		// }
		res.Data = transactionInByte

	}

	if req.Action == "Store-Updated-Client-Across-The-Network" {
		err := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_Clients"), req.Data)
		if err != nil {
			fmt.Println(err)
		}
		res.Data = []byte("abc")
	}

	if req.Action == "Store-Transaction-Across-The-Network" {

		transactionBatch := api.BatchTransaction{}
		proto.Unmarshal(req.Data, &transactionBatch)
		// start := time.Now()
		err := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_Transactions_"+transactionBatch.ID), req.Data)
		if err != nil {
			fmt.Println(err)
		}
		res.Data = []byte("abc")
		// fmt.Println("Store batch at single node", time.Since(start))
	}

	// resInBytes := PrepareResponse(res.Contacts, res.Data)

	return &res, nil
}
