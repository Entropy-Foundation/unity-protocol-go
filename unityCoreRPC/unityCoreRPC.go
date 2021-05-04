package unityCoreRPC

import (
	"context"
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

	. "github.com/dfinity/go-dfinity-crypto/groupsig"
	"github.com/golang/protobuf/proto"

	// guuid "github.com/google/uuid"
	"google.golang.org/grpc"

	"github.com/unity-go/api"
	. "github.com/unity-go/dkg"
	. "github.com/unity-go/findValues"
	. "github.com/unity-go/unityNode"
	. "github.com/unity-go/util"
)

type UnityCore struct {
	Node *UnityNode
}

type NodeTargetId struct {
	node NodeID
}

type FindNodeRequest struct {
	RPCHeader
	Target              api.Contact
	Action              string
	Data                []byte
	OracleGroup         int
	ChannelName         chan []byte
	SelectedOracleGroup int
	txBetweenOracles    []byte
}

type RPCHeader struct {
	Sender api.Contact
}

type FindNodeResponse struct {
	RPCHeader
	Contacts, Value api.Contacts
	Vote            string
	Verified        string
	Data            []byte
	Data2           VoteResponse
	IsTransection   bool
}

type FindNodeResponseForPong struct {
	Res      string
	Vote     string
	Verified string
}

type NodeIdDecimal struct {
	ID     uint64
	NodeID NodeID
}

func BatchingOfTx(self api.Contact, target *api.Contact, action string, data []byte) ([]byte, error) {

	client, err := DialContact(*target)
	if err != nil {
		fmt.Println(err)
	}
	req := NewFindNodeRequest(self, *target, action, data)
	res := FindNodeResponse{}
	defer client.Close()

	err = client.Call("UnityCore.StoreShards", &req, &res)
	if err != nil {
		fmt.Println(err)
	}

	if res.Data != nil {
		return res.Data, nil
	} else {
		return []byte{}, err
	}
}

func (uc *UnityCore) StoreShards(req FindNodeRequest, res *FindNodeResponse) error {
	if Debug == true {
		fmt.Println("Request", req.RPCHeader.Sender.Address, "\n->", req.Target.Address, "\n<<<", req.Data, ">>>\n")
	}

	if req.Action == "Distribute-shards-among-leaders" {
		errorFromSave := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_tx_shards_"+strconv.Itoa(1)), req.Data)
		if errorFromSave != nil {
			fmt.Println(errorFromSave)
		}
		res.Data = GetValue(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_tx_shards_"+strconv.Itoa(1)))

		return nil
	}

	if req.Action == "Circulate-shards-among-leaders" {

		// get all 5 leaders
		listOfLeaders := GetValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_Leaders_Epoch_1"))

		ownTxChunk := GetValue(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_tx_shards_"+strconv.Itoa(1)))

		msg := api.Byte{Msg: ownTxChunk}
		ownTxChunkInBytes, _ := proto.Marshal(&msg)

		// Circulate TX-Chunks among remaining 3 leaders.
		for i := 0; i < len(listOfLeaders.Contact); i++ {
			if listOfLeaders.Contact[i].Address != req.RPCHeader.Sender.Address && listOfLeaders.Contact[i].Address != req.Target.Address {
				_, err := BatchingOfTx(req.Target, listOfLeaders.Contact[i], "Circulate-Process-Start", ownTxChunkInBytes)
				if err != nil {
					fmt.Println(err)
				}
			}
		}
		return nil
	}

	// if req.Action == "Circulate-Process-Start" {

	// 	TxChunks := TransactionChunks{}
	// 	allShards := GetValue(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_tx_shards"))
	// 	TxChunks = append(TxChunks, allShards)

	// 	senderShard := TransactionChunk{}
	// 	proto.Unmarshal(req.Data, &senderShard)
	// 	TxChunks = append(TxChunks, senderShard)

	// 	TxChunksInByte, _ := proto.Marshal(TxChunks)

	// 	errorFromSave := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_tx_shards"), TxChunksInByte)
	// 	if errorFromSave != nil {
	// 		fmt.Println(errorFromSave)
	// 	}
	// }

	if req.Action == "Recover-transaction-among-leaders" {
		res.Data = GetValue(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_tx_shards"))
	}

	return nil
}

func (uc *UnityCore) FindNodeRPC(req api.Request, res *FindNodeResponse) error {

	if req.Action == "store-data-in-leaders" {

		transactionsQueueFromDB := &api.TransactionWithSignatures{}
		upendData := &api.TransactionWithSignatures{}

		response, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"_tx_queue"), nil)

		proto.Unmarshal(response, transactionsQueueFromDB)
		proto.Unmarshal(req.Data, upendData)

		transactionsQueueFromDB.TransactionWithSignature = append(transactionsQueueFromDB.TransactionWithSignature, upendData.TransactionWithSignature...)

		transactionQueueByte, _ := proto.Marshal(transactionsQueueFromDB)

		uc.Node.ValuesDB.Put([]byte(hex.EncodeToString(req.Target.ID[:])+"_tx_queue"), transactionQueueByte, nil)

		return nil

	}

	if req.Action == "Oracle-List-Save" {
		if Debug == true {
			// fmt.Println(hex.EncodeToString(req.Target.ID[:]) + "_Oracles_Epoch_1")
		}

		errorFromSave := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_Oracles_Epoch_1"), req.Data)
		if errorFromSave != nil {
			fmt.Println(errorFromSave)
		}

		// listOfOracles := GetValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_Oracles_Epoch_1"))
		// if Debug == true {
		// 	fmt.Println("")
		// 	fmt.Println("-------------------All Oracles of Epoch 1------------------")
		// 	fmt.Println("")
		// }

		if Debug == true {
			// for i := 0; i <= len(listOfOracles.Contact)-1; i++ {
			// 	fmt.Println(listOfOracles.Contact[i])
			// }
		}

		return nil
	}

	if req.Action == "Leaders-List-Save" {
		errorFromSave := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_Leaders_Epoch_1"), req.Data)
		if errorFromSave != nil {
			fmt.Println(errorFromSave)
		}

		// listOfLeaders := GetValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_Leaders_Epoch_1"))
		// fmt.Println("")
		// fmt.Println("-------------------All Leaders of Epoch 1------------------")
		// fmt.Println("")
		// for i := 0; i <= len(listOfLeaders)-1; i++ {
		// 	fmt.Println(listOfLeaders[i])
		// }
		return nil
	}

	if req.Action == "Shared-Secret-Key-Leader" {
		errorFromSave := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"dkg_Secret_Leaders"), req.Data)
		if errorFromSave != nil {
			fmt.Println(errorFromSave)
		}
		return nil
	}

	if req.Action == "Shared-Public-Key-Leader" {
		err := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"dkg_Public_Leaders"), req.Data)
		if err != nil {
			fmt.Println(err)
		} else {
			if Debug == true {
				fmt.Println("key saved", req.Target.ID, req.Target.Address)
			}
		}
		return nil
	}

	if strings.Contains(req.Action, "dkg_Secret_Oracles") {
		errorFromSave := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"dkg_Secret_Oracles"), req.Data)
		if errorFromSave != nil {
			fmt.Println(errorFromSave)
		}
		return nil
	}

	if strings.Contains(req.Action, "dkg_Public_Oracles") {
		err := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"dkg_Public_Oracles"), req.Data)
		if err != nil {
			fmt.Println(err)
		}
		return nil
	}

	if strings.Contains(req.Action, "dkg_Secret_RSIP_Group") {
		err := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+req.Action), req.Data)
		if err != nil {
			fmt.Println(err)
		}
		return nil
	}

	if strings.Contains(req.Action, "dkg_Public_RSIP_Group_") {
		err := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+req.Action), req.Data)
		if err != nil {
			fmt.Println(err)
		}
		return nil
	}

	if strings.Contains(req.Action, "_Epoch_1_Oracle_Group_") {
		err := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+req.Action), req.Data)
		if err != nil {
			fmt.Println(err)
		}
		return nil
	}

	if strings.Contains(req.Action, "_Tribe_Group_") {
		err := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+req.Action), req.Data)
		if err != nil {
			fmt.Println(err)
		}
		return nil
	}

	if strings.Contains(req.Action, "-Tribe_") {
		err := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+req.Action), req.Data)
		if err != nil {
			fmt.Println(err)
		}
		return nil
	}

	if req.Action == "Retrieve-RSIP-Group-Seckey" {

		key := api.Key{}

		proto.Unmarshal(req.Data, &key)

		secKeyInBytes, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+key.Key), nil)

		dkgSecKey := &Seckey{}
		dkgSecKey.SetHexString(string(secKeyInBytes))

		decimalKey := dkgSecKey.GetDecimalString()
		res.Data, _ = json.Marshal(decimalKey)
		return nil
	}

	if strings.Contains(req.Action, "_Decimal_DKG") {
		errorFromSave := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+req.Action), req.Data)
		if errorFromSave != nil {
			fmt.Println(errorFromSave)
		}
		// return nil
	}

	// if req.Action == "Start-Queued-Batch" {
	// 	res.IsTransection = true

	// 	start := time.Now()
	// 	// fmt.Println("\nSTART QUEUE function------------------------------------------>", req.Target.Address)

	// 	transactionsQueueFromDB, singleBatch, remainingBatch := &api.TransactionWithSignatures{}, &api.TransactionWithSignatures{}, &api.TransactionWithSignatures{}

	// 	// var mutex sync.Mutex
	// 	// mutex.Lock()

	// 	start2 := time.Now()
	// 	// retrieve all batch from queue
	// 	response, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"_tx_queue"), nil)
	// 	proto.Unmarshal(response, transactionsQueueFromDB)

	// 	lengthOfSingleBatch := len(transactionsQueueFromDB.TransactionWithSignature)

	// 	// retrieve first batch from queue
	// 	if len(transactionsQueueFromDB.TransactionWithSignature) >= lengthOfSingleBatch {
	// 		singleBatch.TransactionWithSignature = append(singleBatch.TransactionWithSignature, transactionsQueueFromDB.TransactionWithSignature[:lengthOfSingleBatch]...)
	// 		remainingBatch.TransactionWithSignature = append(remainingBatch.TransactionWithSignature, transactionsQueueFromDB.TransactionWithSignature[lengthOfSingleBatch:]...)
	// 	}

	// 	// remaining batch stored in queue again
	// 	remainingBatchInByte, _ := proto.Marshal(remainingBatch)
	// 	uc.Node.ValuesDB.Put([]byte(hex.EncodeToString(req.Target.ID[:])+"_tx_queue"), remainingBatchInByte, nil)

	// 	if ShowTime == true {
	// 		fmt.Println("\nGet one batch for one leader from database", time.Since(start2))
	// 	}

	// 	// mutex.Unlock()

	// 	individualBatch := &api.BatchTransaction{}
	// 	individualBatch.Data = singleBatch.TransactionWithSignature

	// 	for index := 0; index < len(individualBatch.Data); index++ {
	// 		uuid := guuid.New()
	// 		individualBatch.Data[index].ID = uuid.String()
	// 	}

	// 	// first batch goes for validation process
	// 	finalResponse := QueueHandleTransaction(*individualBatch, uc.Node, req.Target)
	// 	if ShowTime == true {
	// 		fmt.Println("\nSingle Batch Time", len(transactionsQueueFromDB.TransactionWithSignature), len(singleBatch.TransactionWithSignature), len(remainingBatch.TransactionWithSignature), time.Since(start))
	// 	}
	// 	res.Data = finalResponse

	// 	return nil
	// }

	if req.Action == "Transaction-To-Proceed" {

		res.IsTransection = true
		start134 := time.Now()

		voteResponses := &api.VoteResponses{}

		transactionRequest := &api.BatchTransaction{}

		proto.Unmarshal(req.Data, transactionRequest)
		// proto.Unmarshal(req.Data, &transactionRequest)

		clients := api.Clients{}
		response, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(uc.Node.NodeID[:])+"_Clients"), nil)

		proto.Unmarshal(response, &clients)

		for index := 0; index < len(transactionRequest.Data); index++ {
			voteResponse := api.VoteResponse{
				ID:            uc.Node.NodeID,
				Address:       req.Target.Address,
				TransactionID: transactionRequest.Data[index].ID,
			}

			verifiedBalance := VerifyAmount(uc.Node, *transactionRequest.Data[index], clients)
			// verifiedBalance := ""
			if verifiedBalance != "" {
				voteResponse.Vote = "NO"
				voteResponse.Reason = verifiedBalance
			} else {
				voteResponse.Vote = "YES"
			}

			voteResponses.VoteResponse = append(voteResponses.VoteResponse, &voteResponse)

		}

		res.Data, _ = proto.Marshal(voteResponses)
		// if ShowTime == true {

		fmt.Printf("%s took %v\n", "Verify and Vote from single node", time.Since(start134))
		// }
		return nil

	}

	// if req.Action == "to_update_data" {
	// 	requestedOracleID := api.NodeID{}
	// 	emptyID := api.NodeID{}

	// 	getdata, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"_Requested_Oracle"), nil)

	// 	proto.Unmarshal(getdata, &requestedOracleID)

	// 	if Equal(requestedOracleID, emptyID) {
	// 		newData, _ := proto.Marshal(req.RPCHeader.Sender.ID)

	// 		PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_Requested_Oracle"), newData)

	// 		// getdata, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"_Requested_Oracle"), nil)
	// 		// proto.Unmarshal(getdata, &requestedOracleID)

	// 	} else {
	// 		newData, _ := proto.Marshal(requestedOracleID)

	// 		PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_Requested_Oracle"), newData)

	// 	}
	// }

	if req.Action == "check_data" {

		res.Data, _ = uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"_Requested_Oracle"), nil)

		return nil
	}

	// if req.Action == "Transaction-To-Validate-And-Check-Threshold-From-Oracles-On-First-Oracle" {

	// 	start3 := time.Now()

	// 	res.IsTransection = true

	// 	start := time.Now()
	// 	err := godotenv.Load()
	// 	if err != nil {
	// 		fmt.Println("Error loading .env file")
	// 	}

	// 	if Debug == true {
	// 		fmt.Println("", "First Oracle------------------------------")
	// 	}

	// 	numberOfRSIPGroup := os.Getenv("NUMBER_OF_RSIP_GROUP")
	// 	numberOfTribesGroup := os.Getenv("NUMBER_OF_TRIBES")
	// 	numberOfGroup, _ := strconv.Atoi(numberOfRSIPGroup)
	// 	numberOfTribes, _ := strconv.Atoi(numberOfTribesGroup)

	// 	transactionRequest := BatchTransaction{}

	// 	proto.Unmarshal(req.Data, &transactionRequest)
	// 	verified := verifyGroupSignature(uc.Node, transactionRequest, "dkg_Public_Leaders", req.Target.ID)

	// 	fmt.Println("verified +++++++++++++++++++++", verified)

	// 	start6 := time.Now()
	// 	for k := 0; k < len(transactionRequest.Data); k++ {
	// 		voteResponse := VoteResponse{
	// 			ID:            req.Target.ID,
	// 			Address:       req.Target.Address,
	// 			TransactionID: transactionRequest.Data[k].ID,
	// 		}

	// 		threshold := 0
	// 		for j := 0; j <= len(transactionRequest.Data[k].Leaders)-1; j++ {
	// 			if transactionRequest.Data[k].Leaders[j].Vote == "YES" {
	// 				threshold++
	// 			}
	// 		}
	// 		voteResponse.Threshold = strconv.Itoa(threshold) + " out of " + strconv.Itoa(len(transactionRequest.Data[k].Leaders)) + " Leaders"
	// 		transactionRequest.Data[k].Oracles = append(transactionRequest.Data[k].Oracles, voteResponse)

	// 	}

	// 	if ShowTime == true {
	// 		fmt.Println("Single Alpha Oracle votes by itself", time.Since(start6))
	// 	}
	// 	start4 := time.Now()
	// 	oraclesFromDB := Contacts{}

	// 	oracles, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"_Epoch_1_Oracle_Group_"+strconv.Itoa(req.Target.SelectedNumber)), nil)
	// 	proto.Unmarshal(oracles, &oraclesFromDB)

	// 	if Debug == true {
	// 		fmt.Printf("%s took %v\n", "Verify and check threshold for leader from first oracle", time.Since(start))
	// 	}

	// 	betaOraclesCount := len(oraclesFromDB) - 1
	// 	betaOraclesResponseChan := make(chan []byte)
	// 	transactionRequestInByteForRSIP, _ := proto.Marshal(transactionRequest)

	// 	for j := 0; j <= len(oraclesFromDB)-1; j++ {
	// 		if oraclesFromDB[j].ID != req.Target.ID {
	// 			if Debug == true {
	// 				fmt.Println("BETA", j)
	// 			}
	// 			go Bootstrap(oraclesFromDB[j], req.Target, "Transaction-to-beta", transactionRequestInByteForRSIP, betaOraclesResponseChan, "From-beta-to-alpha-oracle")
	// 		}

	// 	}

	// 	for {
	// 		select {
	// 		case response := <-betaOraclesResponseChan:
	// 			betaOraclesCount--
	// 			voteResponses := VoteResponses{}
	// 			proto.Unmarshal(response, &voteResponses)
	// 			if Debug == true {
	// 			}
	// 			for index := 0; index < len(transactionRequest.Data); index++ {
	// 				for j := 0; j < len(voteResponses); j++ {
	// 					if transactionRequest.Data[index].ID == voteResponses[j].TransactionID {
	// 						transactionRequest.Data[index].Oracles = append(transactionRequest.Data[index].Oracles, voteResponses[j])
	// 					}
	// 				}
	// 			}
	// 		}
	// 		if betaOraclesCount == 0 {
	// 			if ShowTime == true {
	// 				fmt.Printf("%s took %v\n", "Alpha oracle to other beta's and alphas oracles with vote response", time.Since(start4))
	// 			}
	// 			break
	// 		}
	// 	}

	// 	start5 := time.Now()
	// 	secret := &Seckey{}
	// 	if Debug == true {
	// 		fmt.Println(strings.Replace(transactionRequest.Signature, " ", "", -1))
	// 	}
	// 	secret.SetHexString(strings.Replace(transactionRequest.Signature, "1 ", "", -1))
	// 	if Debug == true {
	// 		fmt.Println("sec.DecimalString: ", secret.GetDecimalString())
	// 	}
	// 	signatureInDecimalString := secret.GetDecimalString()
	// 	signatureInInt, _ := strconv.Atoi(signatureInDecimalString)

	// 	numberOfSelectedGroup := signatureInInt % numberOfGroup

	// 	// RSIPNodes := Contacts{}

	// 	key := hex.EncodeToString(req.Target.ID[:]) + "_Secret"
	// 	getSecretKey, _ := uc.Node.ValuesDB.Get([]byte(key), nil)
	// 	secretKeyOracle := &Seckey{}
	// 	secretKeyOracle.SetHexString(string(getSecretKey))
	// 	if Debug == true {
	// 		fmt.Println("sec.DecimalString: ", secretKeyOracle.GetDecimalString())
	// 	}

	// 	if ShowTime == true {
	// 		fmt.Printf("%s took %v\n", "Select RSIP groups from Tribe by deterministic number", time.Since(start5))
	// 	}
	// 	//	decimalKey := secretKeyOracle.GetDecimalString()
	// 	//	decimalKeyForSelectTribe, _ := strconv.Atoi(decimalKey)
	// 	//	numberOfSelectedTribe := decimalKeyForSelectTribe % numberOfTribes

	// 	// RSIPUpdateGroup := Contacts{}

	// 	// for index := 0; index < numberOfTribes+1; index++ {

	// 	// 	tribeGroup, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"_Tribe_Group_"+strconv.Itoa(index)), nil)
	// 	// 	proto.Unmarshal(tribeGroup, &RSIPUpdateGroup)

	// 	// 	for j := 0; j < len(RSIPUpdateGroup); j++ {
	// 	// 		Bootstrap(RSIPUpdateGroup[j], req.Target, "to_update_data", nil, nil, "")
	// 	// 	}

	// 	// }

	// 	start8 := time.Now()

	// 	RSIPVotes := VoteResponses{}
	// 	signaturesRSIP := []string{}
	// 	// channelForRSIP := make(ch []byte)
	// 	// newTxBatch.Data = transactionRequest.Data
	// 	ch := make(chan []byte)

	// 	for j := 0; j < numberOfTribes+1; j++ {
	// 		go TransactionToRSIP(uc, req, numberOfSelectedGroup, ch, RSIPVotes, signaturesRSIP, j, transactionRequest)

	// 	}

	// 	RSIPGroupCount := 1
	// 	for {
	// 		select {
	// 		case response := <-ch:
	// 			RSIPGroupCount--
	// 			voteResponses := VoteResponses{}
	// 			proto.Unmarshal(response, &voteResponses)

	// 			for k := 0; k < len(transactionRequest.Data); k++ {
	// 				transactionRequest.Data[k].RSIP = append(transactionRequest.Data[k].RSIP, voteResponses...)
	// 			}

	// 			thresholdOfRSIP := 0

	// 			for index := 0; index < len(transactionRequest.Data); index++ {
	// 				for l := 0; l < len(transactionRequest.Data[index].RSIP); l++ {
	// 					if transactionRequest.Data[index].RSIP[l].Vote == "YES" {
	// 						thresholdOfRSIP++
	// 					}
	// 				}
	// 			}

	// 			for index := 0; index < len(transactionRequest.Data); index++ {
	// 				for j := 0; j <= len(transactionRequest.Data[index].Oracles)-1; j++ {
	// 					transactionRequest.Data[index].Oracles[j].ThresholdOfRSIP = strconv.Itoa(thresholdOfRSIP) + " out of " + strconv.Itoa(len(transactionRequest.Data[index].RSIP)) + " RSIP"
	// 				}
	// 			}

	// 		}
	// 		if RSIPGroupCount == 0 {
	// 			// fmt.Printf("%s took %v\n", "Alpha to other beta's and alphas", time.Since(start4))

	// 			break
	// 		}
	// 	}

	// 	if ShowTime == true {
	// 		fmt.Printf("%s took %v\n", "From RSIP votes from all Tribes", time.Since(start8))
	// 		fmt.Printf("%s took %v\n", "Single Alpha Oracle after sending tx to all other oracles and RSIPs", time.Since(start3))
	// 	}
	// 	fmt.Println(len(signaturesRSIP))

	// 	transactionRequest.SignatureOfRSIP = signaturesRSIP
	// 	transactionBatchImBytes, _ := proto.Marshal(transactionRequest)
	// 	res.Data = transactionBatchImBytes

	// 	return nil

	// }

	if req.Action == "Transaction-To-Proceed-In-RSIP" {

		// This service is used to proceed batch in Alpha RSIP, RSIP will send batch to other alpha and

		start := time.Now()
		// line to add data in channle
		res.IsTransection = true

		transactionRequest := &api.BatchTransaction{}

		start15 := time.Now()
		proto.Unmarshal(req.Data, transactionRequest)
		if ShowTime == true {
			fmt.Println("Time to unmarshal REQUEST", time.Since(start15))
		}
		RSIPNodes := &api.Contacts{}

		start1 := time.Now()
		getListOfRSIPNodes, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"-Tribe_"+strconv.Itoa(int(req.Target.SelectedTribe))+"_RSIP_GROUP_"+strconv.Itoa(int(req.Target.SelectedNumber))), nil)
		proto.Unmarshal(getListOfRSIPNodes, RSIPNodes)
		if ShowTime == true {
			fmt.Println("Get list of RSIP nodes", time.Since(start1))
		}
		clients := api.Clients{}
		response, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(uc.Node.NodeID[:])+"_Clients"), nil)

		proto.Unmarshal(response, &clients)

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
		}
		if ShowTime == true {
			fmt.Println("Self RSIP alpha votes", time.Since(start2))
		}
		start3 := time.Now()

		transactionRequestInByteForRSIP, _ := proto.Marshal(transactionRequest)
		// transactionRequestInByteForRSIP, _ := proto.Marshal(transactionRequest)

		betaRSIPResponseChan := make(chan []byte)
		for j := 0; j <= len(RSIPNodes.Contact)-1; j++ {
			if Equal(RSIPNodes.Contact[j].ID, req.Target.ID) {
				go Bootstrap(RSIPNodes.Contact[j], req.Target, "Transaction-To-Proceed", transactionRequestInByteForRSIP, betaRSIPResponseChan, "")
			}

		}

		betaOraclesCount := len(RSIPNodes.Contact) - 1

		voteResponses := api.VoteResponses{}
		for {
			select {
			case response := <-betaRSIPResponseChan:
				betaOraclesCount--
				proto.Unmarshal(response, &voteResponses)

				for index := 0; index < len(transactionRequest.Data); index++ {
					for j := 0; j < len(voteResponses.VoteResponse); j++ {
						if transactionRequest.Data[index].ID == voteResponses.VoteResponse[j].TransactionID {
							transactionRequest.Data[index].RSIP.VoteResponse = append(transactionRequest.Data[index].RSIP.VoteResponse, voteResponses.VoteResponse[j])
						}
					}
				}
			}
			if betaOraclesCount == 0 {
				// if ShowTime == true {
				fmt.Printf("%s took %v\n", "Each RSIP alpha getting response from all other RSIP alphas and betas", time.Since(start3))
				// }
				break
			}
		}

		start5 := time.Now()

		key := "_Tribe_" + strconv.Itoa(int(req.Target.SelectedTribe)) + "_dkg_Secret_RSIP_Group_" + strconv.Itoa(int(req.Target.SelectedNumber))

		// fmt.Println(key, "Private")
		sign := GenerateBatchSignature(uc.Node, key, uc.Node.NodeID, *transactionRequest)
		// sign := ""

		transactionRequest.Signature = sign
		res.Data, _ = proto.Marshal(transactionRequest)

		if ShowTime == true {
			fmt.Println("Signature generation from RSIP group ", time.Since(start5), "::size", len(res.Data))
			fmt.Println("Single RSIP alpha to process whole batch ", time.Since(start), "::size", len(res.Data))
		}
		return nil

	}

	// if req.Action == "Transaction-to-beta" {

	// 	res.IsTransection = true

	// 	err := godotenv.Load()
	// 	if err != nil {
	// 		fmt.Println("Error loading .env file")
	// 	}

	// 	transactionRequest := BatchTransaction{}
	// 	voteResponses := VoteResponses{}

	// 	proto.Unmarshal(req.Data, &transactionRequest)

	// 	for k := 0; k < len(transactionRequest.Data); k++ {
	// 		voteResponse := VoteResponse{
	// 			ID:            req.Target.ID,
	// 			Address:       req.Target.Address,
	// 			TransactionID: transactionRequest.Data[k].ID,
	// 		}

	// 		threshold := 0
	// 		for j := 0; j <= len(transactionRequest.Data[k].Leaders)-1; j++ {
	// 			if transactionRequest.Data[k].Leaders[j].Vote == "YES" {
	// 				threshold++
	// 			}
	// 		}
	// 		voteResponse.Threshold = strconv.Itoa(threshold) + " out of " + strconv.Itoa(len(transactionRequest.Data[k].Leaders)) + " Leaders"
	// 		voteResponses = append(voteResponses, voteResponse)

	// 	}

	// 	res.Data, _ = proto.Marshal(voteResponses)

	// }

	if req.Action == "Transaction-To-Validate-And-Proceed-To-All-Oracles" {
		start3 := time.Now()
		if Debug == true {
			fmt.Println(req.Target.ID, req.Target.Address, "Oracles all")
		}

		// numberOfRSIPGroup := os.Getenv("NUMBER_OF_RSIP_GROUP")
		transactionRequestInByteForRSIP := []byte{}
		transactionRequest := api.TransactionWithSignature{}
		voteResponse := VoteResponse{
			ID:      req.Target.ID,
			Address: req.Target.Address,
		}

		proto.Unmarshal(req.Data, &transactionRequest)
		signature := transactionRequest.Signature

		Oracles := transactionRequest.Oracles
		RSIP := transactionRequest.RSIP

		transactionRequest.Oracles = &api.VoteResponses{}
		transactionRequest.RSIP = &api.VoteResponses{}
		verified := ""
		transactionRequest.Oracles = Oracles
		transactionRequest.RSIP = RSIP

		if verified != "" {
			if Debug == true {
				fmt.Println(verified)
			}

		} else {
			if Debug == true {
				fmt.Println("Signature Verified")
			}

			transactionRequest.Signature = signature

			threshold := 0
			for j := 0; j <= len(transactionRequest.Leaders.VoteResponse)-1; j++ {
				if transactionRequest.Leaders.VoteResponse[j].Vote == "YES" {
					threshold++
				}
			}
			voteResponse.Threshold = strconv.Itoa(threshold) + " out of " + strconv.Itoa(len(transactionRequest.Leaders.VoteResponse)) + " Leaders"
			// transactionRequest.Oracles = append(transactionRequest.Oracles, voteResponse)

			// start4 := time.Now()

			// numberOfGroup, _ := strconv.Atoi(numberOfRSIPGroup)
			// min := 0
			// max := numberOfGroup

			// randNumber := rand.Intn(max-min) + min
			// RSIPNodes := Contacts{}
			// getListOfRSIP, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"-Tribe_"+strconv.Itoa(req.OracleGroup)+"_RSIP_GROUP_"+strconv.Itoa(randNumber)), nil)
			// proto.Unmarshal(getListOfRSIP, &RSIPNodes)

			// Oracles := transactionRequest.Oracles
			// RSIP := transactionRequest.RSIP
			// transactionRequest.Oracles = VoteResponses{}
			// transactionRequest.RSIP = VoteResponses{}
			// transactionObjForBootNode, _ := proto.Marshal(transactionRequest)

			// sign := GenerateSignature(uc.Node, transactionObjForBootNode, "dkg_Secret_Oracles", req.Target.ID, transactionRequest)
			// transactionRequest.Oracles = Oracles
			// transactionRequest.RSIP = RSIP
			// transactionRequest.Signature = sign.GetHexString()

			// transactionRequestInByteForRSIP, _ = proto.Marshal(transactionRequest)
			// req.OracleGroup = randNumber

			// if Debug == true {
			// 	fmt.Println("Selected RSIP Group", randNumber, transactionRequest.Signature)
			// }

			// for j := 0; j <= len(RSIPNodes)-1; j++ {
			// 	_, newTransaction, errFromOracles := Bootstrap(RSIPNodes[j], req.Target, "Transaction-To-RSIP-Nodes", transactionRequestInByteForRSIP, nil)
			// 	// Oracles := transactionRequest.Oracles
			// 	// RSIP := transactionRequest.RSIP
			// 	// transactionRequest.Oracles = VoteResponses{}
			// 	// transactionRequest.RSIP = VoteResponses{}

			// 	// verified := verifyGroupSignature(uc.Node, transactionRequest, "dkg_Public_RSIP_Group_"+strconv.Itoa(randNumber), req.Target.ID)
			// 	// fmt.Println(verified, "++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++")
			// 	// transactionRequest.Oracles = Oracles
			// 	// transactionRequest.RSIP = RSIP
			// 	// transactionRequest.Signature = sign.GetHexString()
			// 	// transactionRequestInByteForRSIP, _ = proto.Marshal(transactionRequest)
			// 	transactionRequestInByteForRSIP = newTransaction
			// 	if errFromOracles != nil {
			// 		fmt.Println("Oracle Error:", errFromOracles)
			// 	}

			// }
			// if Debug == true {
			// 	fmt.Printf("%s took %v\n", "Select Random RSIP group and Procced tx from RSIP for single Oracle", time.Since(start4))
			// }
			// proto.Unmarshal(transactionRequestInByteForRSIP, &transactionRequest)
			// thresholdOfRSIP := 0
			// for k := 0; k <= len(transactionRequest.RSIP)-1; k++ {
			// 	if transactionRequest.RSIP[k].Oracle == req.Target.ID {
			// 		if transactionRequest.RSIP[k].Vote == "YES" {
			// 			thresholdOfRSIP++
			// 		}
			// 	}
			// }

			// for j := 0; j <= len(transactionRequest.Oracles)-1; j++ {
			// 	if transactionRequest.Oracles[j].ID == req.Target.ID {
			// 		transactionRequest.Oracles[j].ThresholdOfRSIP = strconv.Itoa(thresholdOfRSIP) + " Out Of " + strconv.Itoa(len(RSIPNodes)) + " RSIP"
			// 	}
			// }
			transactionRequestInByteForRSIP, _ = proto.Marshal(&transactionRequest)

			if Debug == true {
				fmt.Printf("%s took %v\n", "Verify transaction and process from it's RSIP from Single Oracle", time.Since(start3))
			}
			res.Data = transactionRequestInByteForRSIP
		}

		return nil
	}

	// if req.Action == "Transaction-To-RSIP-Nodes" {
	// 	res.IsTransection = true
	// 	start4 := time.Now()
	// 	transactionRequest := TransactionWithSignature{}
	// 	clients := []Client{}
	// 	response, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(uc.Node.NodeID[:])+"_Clients"), nil)

	// 	proto.Unmarshal(response, &clients)
	// 	proto.Unmarshal(req.Data, &transactionRequest)
	// 	voteResponse := VoteResponse{
	// 		ID:      req.Target.ID,
	// 		Address: req.Target.Address,
	// 	}

	// 	signature := transactionRequest.Signature
	// 	Oracles := transactionRequest.Oracles
	// 	RSIP := transactionRequest.RSIP
	// 	transactionRequest.Oracles = VoteResponses{}
	// 	transactionRequest.RSIP = VoteResponses{}
	// 	// verified := verifyGroupSignature(uc.Node, transactionRequest, "dkg_Public_Oracles", req.Target.ID)
	// 	verified := ""
	// 	transactionRequest.Oracles = Oracles
	// 	transactionRequest.RSIP = RSIP
	// 	verifiedBalance := VerifyAmount(uc.Node, transactionRequest, clients)
	// 	// voteResponse.Oracle = req.RPCHeader.Sender.ID
	// 	if verified != "" {
	// 		if Debug == true {
	// 			fmt.Println(verified, "not verified in RSIP")
	// 		}

	// 		voteResponse.Reason = verified
	// 		voteResponse.Vote = "NO"
	// 	} else if verifiedBalance != "" {
	// 		voteResponse.Vote = "NO"
	// 		voteResponse.Reason = verifiedBalance
	// 	} else {
	// 		voteResponse.Vote = "YES"
	// 		// transactionObjInByte, _ := proto.Marshal(transactionRequest)
	// 		// transactionRequest.Oracles = VoteResponses{}
	// 		// transactionRequest.RSIP = VoteResponses{}
	// 		// // key := hex.EncodeToString(req.Target.ID[:]) + "dkg_Secret_RSIP_Group_" + strconv.Itoa(req.OracleGroup)
	// 		// sign := GenerateSignature(uc.Node, transactionObjInByte, "dkg_Secret_RSIP_Group_"+strconv.Itoa(req.OracleGroup), req.Target.ID, transactionRequest)
	// 		// transactionRequest.Signature = sign.GetHexString()
	// 		// transactionRequest.Oracles = Oracles
	// 		// transactionRequest.RSIP = RSIP
	// 		transactionRequest.Signature = signature
	// 		if Debug == true {
	// 			fmt.Println(verified, "verified in RSIP")
	// 		}

	// 	}
	// 	// transactionRequest.RSIP = append(transactionRequest.RSIP, voteResponse)
	// 	res.Data, _ = proto.Marshal(voteResponse)
	// 	if Debug == true {

	// 		fmt.Printf("%s took %v\n", "Procced transaction in a single RSIP", time.Since(start4))
	// 	}
	// 	return nil
	// }

	if req.Action == "Transaction-To-Leader-After-Verification-From-Oracle" {
		// uc.Node.Mutex.Lock()
		start := time.Now()
		fmt.Println("selected leader node", req.Target.Address)

		clients := api.Clients{}
		// client := Client{}
		transactionRequest := api.BatchTransaction{}
		proto.Unmarshal(req.Data, &transactionRequest)
		contactsFromOwnDb := GetValuesById(uc.Node, []byte(string(req.Target.ID[:])))
		response, _ := uc.Node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"_Clients"), nil)
		proto.Unmarshal(response, &clients)

		// verified := verifyGroupSignature(uc.Node, transactionRequest, "dkg_Public_Oracles", req.Target.ID)

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

			newBatchResponse.Data = append(newBatchResponse.Data, &latestTx)

			// Transactions = append(Transactions, transactionObject)
		}

		transactionInByte, _ := proto.Marshal(&newBatchResponse)

		if ShowTime == true {
			fmt.Println("Time to check amount and threshold of single batch on LEADER", time.Since(start), len(transactionInByte))
		}
		start0 := time.Now()

		for i := 0; i <= len(contactsFromOwnDb.Contact)-1; i++ {
			// if contactsFromOwnDb[i].ID != req.Target.ID {
			request := api.Request{
				Action: "Store-Updated-Client-Across-The-Network",
				Data:   byteValue,
			}

			BootstrapSync(*req.Target, contactsFromOwnDb.Contact[i], request, nil, nil)

			request2 := api.Request{
				Action: "Store-Updated-Client-Across-The-Network",
				Data:   transactionInByte,
			}

			// }
			BootstrapSync(*req.Target, contactsFromOwnDb.Contact[i], request2, nil, nil)

		}
		// if ShowTime == true {
		fmt.Println("Store updated client and batch at all nodes", time.Since(start0))
		// }
		res.Data = transactionInByte

		return nil

	}

	if req.Action == "Store-Updated-Client-Across-The-Network" {
		err := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_Clients"), req.Data)
		if err != nil {
			return nil
		}
		return nil
	}
	if req.Action == "Store-Transaction-Across-The-Network" {

		// start := time.Now()
		err := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_Transactions"), req.Data)
		if err != nil {
			return nil
		}
		// fmt.Println("Store batch at single node", time.Since(start))
		return nil
	}

	// if req.Action == "Organizing-RSIP-Nodes-Group" {

	// 	numberOfRSIPGroup := os.Getenv("NUMBER_OF_RSIP_GROUP")
	// 	numberOfRSIPGroupInInt, _ := strconv.Atoi(numberOfRSIPGroup)
	// 	if Debug == true {
	// 		fmt.Println(req.Target.Address, "TO")
	// 	}

	// 	allNodes := &api.Contacts{}

	// 	// get list of oracles tribe
	// 	listOfOracles := GetValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_Tribe"))
	// 	min := 0
	// 	max := len(listOfOracles.Contact) - 1

	// 	// group nodes into RSIP
	// 	for i := 0; i <= numberOfRSIPGroupInInt; i++ {

	// 		randNumber := rand.Intn(max-min) + min
	// 		selectedNode := listOfOracles.Contact[randNumber]

	// 		listOfOracles.Contact = append(listOfOracles.Contact[:randNumber], listOfOracles.Contact[randNumber+1:]...)

	// 		listOfRSIPNodes := GetValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_RSIP_GROUP_"+strconv.Itoa(i)))
	// 		listOfRSIPNodes.Contact = append(listOfRSIPNodes.Contact, selectedNode)
	// 		listOfOraclesInByte, _ := proto.Marshal(&listOfRSIPNodes)

	// 		errorFromPut := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_RSIP_GROUP_"+strconv.Itoa(i)), listOfOraclesInByte)
	// 		if errorFromPut != nil {
	// 			fmt.Println(errorFromPut)
	// 		}
	// 		max--
	// 	}

	// 	for len(listOfOracles.Contact) > 0 {
	// 		for i := 0; i <= numberOfRSIPGroupInInt; i++ {
	// 			randNumber := 0
	// 			min := 0
	// 			max := len(listOfOracles.Contact)
	// 			if max > 0 {
	// 				randNumber = rand.Intn(max-min) + min
	// 			}
	// 			if len(listOfOracles.Contact) != 0 {
	// 				selectedNode := listOfOracles.Contact[randNumber]
	// 				listOfOracles.Contact = append(listOfOracles.Contact[:randNumber], listOfOracles.Contact[randNumber+1:]...)

	// 				listOfRSIPNodes := GetValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_RSIP_GROUP_"+strconv.Itoa(i)))

	// 				listOfRSIPNodes.Contact = append(listOfRSIPNodes.Contact, selectedNode)
	// 				listOfOraclesInByte, _ := proto.Marshal(&listOfRSIPNodes)

	// 				errorFromPut := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_RSIP_GROUP_"+strconv.Itoa(i)), listOfOraclesInByte)
	// 				if errorFromPut != nil {
	// 					fmt.Println(errorFromPut)
	// 				}
	// 				max--

	// 			}
	// 		}
	// 	}

	// 	// Generate DKF for each Group
	// 	for i := 0; i <= numberOfRSIPGroupInInt; i++ {
	// 		// if i == 0 {
	// 		secretKey := ""
	// 		listOfRSIPNodes := GetValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_RSIP_GROUP_"+strconv.Itoa(i)))
	// 		if Debug == true {
	// 			fmt.Println("------------- Group one -------------\n", len(listOfRSIPNodes.Contact))
	// 		}

	// 		for j := 0; j <= len(listOfRSIPNodes.Contact)-1; j++ {
	// 			if j != 0 {
	// 				key := hex.EncodeToString(req.Target.ID[:]) + "dkg_Secret_RSIP_Group_" + strconv.Itoa(i)
	// 				getSecretKey, _ := uc.Node.ValuesDB.Get([]byte(key), nil)
	// 				secretKey, _ = PingNodeForDKG(listOfRSIPNodes.Contact[j], listOfRSIPNodes.Contact[0], "dkg_Secret_RSIP_Group_"+strconv.Itoa(i), getSecretKey)

	// 			}
	// 		}

	// 		if Debug == true {
	// 			for j := 0; j <= len(listOfRSIPNodes.Contact)-1; j++ {
	// 				fmt.Println(listOfRSIPNodes.Contact[j])
	// 			}
	// 			fmt.Println("")
	// 		}

	// 		allNodesOfNetwork, _ := uc.Node.ValuesDB.Get(req.Target.ID[:], nil)
	// 		proto.Unmarshal(allNodesOfNetwork, allNodes)

	// 		// store group's secret key among group
	// 		for j := 0; j <= len(listOfRSIPNodes.Contact)-1; j++ {
	// 			if j != 0 {
	// 				_, _, err := Bootstrap(listOfRSIPNodes.Contact[j], req.Target, "dkg_Secret_RSIP_Group_"+strconv.Itoa(i), []byte(secretKey), nil, "")
	// 				if err != nil {
	// 					fmt.Println("Share Secret Key Of Group Error:", err)
	// 				}
	// 			}
	// 		}

	// 		groupPublicKey := GenerateSharedPublicKey(secretKey)
	// 		if Debug == true {
	// 			fmt.Println(groupPublicKey, "groupPublicKey", i)
	// 		}

	// 		// store group's public key across the network
	// 		for j := 0; j <= len(allNodes.Contact)-1; j++ {
	// 			_, _, err := Bootstrap(allNodes.Contact[j], req.Target, "dkg_Public_RSIP_Group_"+strconv.Itoa(i), []byte(groupPublicKey), nil, "")
	// 			if err != nil {
	// 				fmt.Println("Share Secret Key Of Group Error:", err)
	// 			}
	// 		}

	// 	}

	// 	return nil
	// }

	data := &api.Contacts{}

	value := GetValuesById(uc.Node, []byte(string(req.Target.ID[:])))
	if req.Action != "" {
		proto.Unmarshal([]byte(req.Data), data)
		data = Unique(data)
	} else {
		value.Contact = append(value.Contact, req.RPCHeader.Sender)
		data = Unique(&value)
	}

	sender, _ := proto.Marshal(data)
	errorFromGet := PutValuesById(uc.Node, []byte(string(req.Target.ID[:])), sender)
	if errorFromGet != nil {
		fmt.Println(errorFromGet)
	}

	// if value != api.Contact{} {
	res.Contacts = *data
	// }

	return nil
}

func NewFindNodeRequest(self, target api.Contact, action string, data []byte) FindNodeRequest {
	return FindNodeRequest{
		RPCHeader: RPCHeader{
			Sender: self,
		},
		Target: target,
		Action: action,
		Data:   data,
	}
}

func Unique(intSlice *api.Contacts) *api.Contacts {
	keys := make(map[string]bool)
	list := &api.Contacts{}
	for _, entry := range intSlice.Contact {
		if _, value := keys[entry.Address]; !value {
			keys[entry.Address] = true
			list.Contact = append(list.Contact, entry)
		}
	}
	return list
}

// func (uc *UnityCore) SendMessageToTargetNode(req api.Request, res api.FindNodeResponse) error {
// 	if Debug == true {
// 		fmt.Println(uc.Node.NodeID)
// 	}

// 	respondedNodesIDs := []NodeIdDecimal{}
// 	respondedNodes := &api.Contacts{}

// 	listOfOracles := GetValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_Tribe"))
// 	if Debug == true {
// 		fmt.Println("first", req.Action != "PING", req.Action)
// 	}

// 	if req.Action != "PING" {
// 		req.RPCHeader.Sender.Oracle = req.Target.ID
// 		listOfOracles.Contact = append(listOfOracles.Contact, req.RPCHeader.Sender)
// 		if Debug == true {
// 			fmt.Println("")
// 			fmt.Println("-------------------Tribe Of Oracle ->", req.Target.Address, "-----------------")
// 			fmt.Println("")
// 			for i := 0; i <= len(listOfOracles.Contact)-1; i++ {
// 				fmt.Println(listOfOracles.Contact[i])
// 			}
// 		}

// 		sender, _ := proto.Marshal(listOfOracles)

// 		errorFromSave := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_Tribe"), sender)
// 		if errorFromSave != nil {
// 			fmt.Println(errorFromSave)
// 		}

// 		proto.Unmarshal(sender, &listOfOracles)
// 		res.Value = &listOfOracles
// 		return nil
// 	} else {
// 		for i := 0; i <= len(listOfOracles.Contact)-1; i++ {
// 			PONG, _ := Ping(req.RPCHeader.Sender, listOfOracles.Contact[i])

// 			if PONG == "PONG" {
// 				respondedNodes.Contact = append(respondedNodes.Contact, listOfOracles.Contact[i])
// 			}
// 		}

// 		for i := 0; i <= len(listOfOracles.Contact)-1; i++ {
// 			var mySlice = []byte(string(listOfOracles.Contact[i].ID[:]))
// 			data := binary.BigEndian.Uint64(mySlice)
// 			respondedNodesIDs = append(respondedNodesIDs, NodeIdDecimal{
// 				data, listOfOracles.Contact[i].ID,
// 			})
// 		}
// 		sort.Slice(respondedNodesIDs, func(i, j int) bool { return respondedNodesIDs[i].ID < respondedNodesIDs[j].ID })
// 		largest := respondedNodesIDs[len(respondedNodesIDs)-1]
// 		if Debug == true {
// 			fmt.Println("")
// 			fmt.Println("Responded to ->", req.RPCHeader.Sender.Address, "Oracle")
// 			fmt.Println("")
// 		}

// 		selectedLeader := &api.Contact{}
// 		for i := 0; i <= len(respondedNodes.Contact)-1; i++ {
// 			if Equal(respondedNodes.Contact[i].ID, largest.NodeID) {
// 				selectedLeader = respondedNodes.Contact[i]
// 			}
// 		}

// 		selectedLeader.IsLeader = true

// 		for j := 0; j <= len(listOfOracles.Contact)-1; j++ {
// 			if listOfOracles.Contact[j].Address == selectedLeader.Address {
// 				listOfOracles.Contact = append(listOfOracles.Contact[:j], listOfOracles.Contact[j+1:]...)
// 			}
// 		}

// 		listOfOraclesInByte, _ := proto.Marshal(listOfOracles)

// 		errorFromSave := PutValuesById(uc.Node, []byte(hex.EncodeToString(req.Target.ID[:])+"_Tribe"), listOfOraclesInByte)
// 		if errorFromSave != nil {
// 			fmt.Println(errorFromSave)
// 		}

// 		selectedLeaders := &api.Contacts{}
// 		selectedLeaders.Contact = append(selectedLeaders.Contact, selectedLeader)
// 		res.Value = selectedLeaders
// 		return nil
// 	}

// }

func (uc *UnityCore) RespondToOracle(req FindNodeRequest, res *FindNodeResponseForPong) error {
	res.Res = "PONG"
	return nil
}

func Ping(self, target *api.Contact) (string, error) {
	client, err := DialContact(*target)
	if err != nil {
		return "", err
	}
	req := NewFindNodeRequest(*self, *target, "", []byte{})
	res := FindNodeResponseForPong{}
	defer client.Close()

	err = client.Call("UnityCore.RespondToOracle", &req, &res)
	if err != nil {
		return "", err
	}

	return res.Res, nil
}

func (uc *UnityCore) DKG(req FindNodeRequest, res *FindNodeResponseForPong) error {
	if Debug == true {
		fmt.Println(req.Target.Address, req.RPCHeader.Sender.Address)
	}

	key := ""

	if req.Action == "Leaders" {
		key = hex.EncodeToString(req.Target.ID[:]) + "dkg_Secret_Leaders"
	} else if req.Action == "Oracles" {
		key = hex.EncodeToString(req.Target.ID[:]) + "dkg_Secret_Oracles"
	} else {
		key = hex.EncodeToString(req.Target.ID[:]) + req.Action
	}

	getSecretKey, _ := uc.Node.ValuesDB.Get([]byte(key), nil)
	if Debug == true {
		fmt.Println(getSecretKey, "getSecretKey")
	}

	dkgSecretShare := AggregateSeckeysForDKG(getSecretKey, req.Data)
	errorFromGet := PutValuesById(uc.Node, []byte(key), []byte(dkgSecretShare))
	if errorFromGet != nil {
		fmt.Println(errorFromGet)
	}

	res.Res = dkgSecretShare
	return nil
}

func (uc *UnityCore) SendMessageToTargetNodeForDkg(req api.Request, res *FindNodeResponseForPong) error {

	var getSecretKey []byte

	firstSelectedLeader := &api.Contact{}

	proto.Unmarshal(req.Data, firstSelectedLeader)
	key := ""

	if req.Action == "Leaders" {
		key = hex.EncodeToString(req.Target.ID[:]) + "dkg_Secret_Leaders"
	} else if req.Action == "Oracles" {
		key = hex.EncodeToString(req.Target.ID[:]) + "dkg_Secret_Oracles"
	} else {
		key = hex.EncodeToString(req.Target.ID[:]) + req.Action
	}

	getSecretKey, _ = uc.Node.ValuesDB.Get([]byte(key), nil)

	if Debug == true {
		fmt.Println(getSecretKey, "getSecretKey", key)
	}

	response, errFromListUpdate := PingNodeForDKG(req.Target, firstSelectedLeader, req.Action, getSecretKey)
	if errFromListUpdate != nil {
		fmt.Println("List Update Error:", errFromListUpdate)
	}
	res.Res = response
	return nil
}

func PingNodeForDKG(self, target *api.Contact, action string, data []byte) (string, error) {
	client, err := DialContact(*target)
	if err != nil {
		return "", err
	}
	req := NewFindNodeRequest(*self, *target, action, data)
	res := FindNodeResponseForPong{}
	defer client.Close()

	err = client.Call("UnityCore.DKG", &req, &res)
	if err != nil {
		return "", err
	}

	return res.Res, nil
}

func Bootstrap(target, self *api.Contact, action string, data []byte, handleData chan []byte, param string) (api.Contacts, []byte, error) {
	if action == "" {
		if Debug == true {
			fmt.Println("bootstraping to", target.Address)
		}
	}
	client, err := DialContact(*target)
	if err != nil {
		return api.Contacts{}, []byte{}, err
	}
	req := NewFindNodeRequest(*self, *target, action, data)
	res := FindNodeResponse{}

	err = client.Call("UnityCore.FindNodeRPC", &req, &res)
	defer client.Close()

	if res.IsTransection == true {
		handleData <- res.Data
	}

	if err != nil {
		return api.Contacts{}, []byte{}, err
	}

	if res.Data != nil {
		return api.Contacts{}, res.Data, nil
	} else {
		return res.Contacts, []byte{}, nil
	}

}

func Bootstrap2(target, self api.Contact, action string, data []byte, handleData chan []byte, param string) (api.Contacts, []byte, error) {
	if action == "" {
		if Debug == true {
			fmt.Println("bootstraping to", target.Address)
		}
	}
	client, err := DialContact(target)
	if err != nil {
		return api.Contacts{}, []byte{}, err
	}
	req := NewFindNodeRequest(self, target, action, data)
	res := FindNodeResponse{}

	// m.Lock()
	err = client.Call("UnityCore.FindNodeRPC", &req, &res)
	// m.Unlock()
	defer client.Close()

	if res.IsTransection == true {
		handleData <- res.Data
	}

	if err != nil {
		return api.Contacts{}, []byte{}, err
	}

	if res.Data != nil {
		return api.Contacts{}, res.Data, nil
	} else {
		return res.Contacts, []byte{}, nil
	}

}

func VerifySign(node *UnityNode, transactionRequest api.TransactionWithSignature) string {
	clients := api.Clients{}
	client := api.Client{}
	response, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(node.NodeID[:])+"_Clients"), nil)

	proto.Unmarshal(response, &clients)

	for j := 0; j <= len(clients.Client)-1; j++ {
		if clients.Client[j].Address == transactionRequest.From {
			client = *clients.Client[j]
		}
	}

	// if (api.Client{}) == client {
	// 	return "Client details not found"
	// }

	transactionRequestWithoutSign := transactionRequest

	transactionRequestInByte, _ := proto.Marshal(&transactionRequestWithoutSign)

	publicKeyOfClient := &Pubkey{}
	signature := &Signature{}
	signature.SetHexString(transactionRequest.Signature)
	publicKeyOfClient.SetHexString(client.PublicKey)

	if Debug == true {
		fmt.Println(publicKeyOfClient, signature, transactionRequestInByte)

		fmt.Println(VerifySig(*publicKeyOfClient, transactionRequestInByte, *signature), "signature", transactionRequest.Signature, string(transactionRequestInByte))
	}

	if !VerifySig(*publicKeyOfClient, transactionRequestInByte, *signature) {
		return "Signature verification error"
	}
	return ""
}

func VerifyAmount(node *UnityNode, transactionRequest api.TransactionWithSignature, clients api.Clients) string {

	client := api.Client{}

	for j := 0; j <= len(clients.Client)-1; j++ {
		if clients.Client[j].Address == transactionRequest.From {
			client = *clients.Client[j]
		}
	}

	// if (Client{}) == client {
	// 	return "Client details not found"
	// }

	if client.Amount < transactionRequest.Amount {
		return "Client not have sufficient balance"
	}
	return ""
}

func FindNodeForDkg(self, target api.Contact, action string, data []byte) (string, error) {
	client, err := DialContact(target)
	if err != nil {
		return "", err
	}
	req := NewFindNodeRequest(self, target, action, data)
	res := FindNodeResponseForPong{}
	defer client.Close()

	err = client.Call("UnityCore.SendMessageToTargetNodeForDkg", &req, &res)
	if err != nil {
		return "", err
	}

	return res.Res, nil
}

func VerifyGroupSignature(node *UnityNode, transactionRequest api.BatchTransaction, key string, id NodeID) string {
	publicKey := &Pubkey{}

	getPublicKey, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(id[:])+key), nil)
	publicKey.SetHexString(string(getPublicKey))
	// signature := transactionRequest.Signature

	sign := Signature{}
	sign.SetHexString(transactionRequest.Signature)

	transactionRequest.Signature = ""

	transactionRequestInByte, _ := proto.Marshal(&transactionRequest)

	asig := AggregateSigs([]Signature{sign})

	if !VerifyAggregateSig([]Pubkey{*publicKey}, transactionRequestInByte, asig) {
		return "Aggregated signature does not verify"
	}
	return ""
}

func GenerateSignature(node *UnityNode, transactionObjForBootNode []byte, key string, id NodeID, transactionRequest api.TransactionWithSignature) Signature {
	secret := &Seckey{}

	getSecretKey, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(id[:])+key), nil)

	proto.Unmarshal(transactionObjForBootNode, &transactionRequest)

	transactionRequest.Signature = ""

	transactionObjForBootNode, _ = proto.Marshal(&transactionRequest)

	secret.SetHexString(string(getSecretKey))
	sig := Sign(*secret, transactionObjForBootNode)
	return sig
}

/********************************************************************QUEUE-MESSAGE*******************************************************************/

func QueueHandleTransaction(transactionsRequest api.BatchTransaction, node *UnityNode, selfContact *api.Contact, leaderGroupNumber int, channel chan api.BatchTransaction) {

	start := time.Now()
	start12 := time.Now()
	// fmt.Println("\nPROCESSOR BATCH START--------------------------------------------->", selfContact.Address)

	// transactionsResponse := BatchTransaction{}
	ch := make(chan []byte)

	// ======================================      TRANSACTION TO ORACLE         ============================================================
	// for i := 0; i < len(transactionsRequest); i++ {
	// 	if Debug == true {
	// 		fmt.Println("Transaction", i)
	// 	}
	start2 := time.Now()
	go Transactions(node, selfContact, transactionsRequest, ch, leaderGroupNumber)
	// }

	// for i := 0; i < len(transactionsRequest); i++ {
	// tx := BatchTransaction{}
	// proto.Unmarshal(<-ch, &tx)
	// }

	dataInBytes := []byte{}
	responseFromLeader := 1
	for {
		select {
		case response := <-ch:
			responseFromLeader--
			dataInBytes = response
			voteResponse := api.BatchTransaction{}
			proto.Unmarshal(response, &voteResponse)
			transactionsRequest = voteResponse
		}
		if responseFromLeader == 0 {
			if ShowTime == true {
				Log(transactionsRequest.ID, "Votes by all LEADERS took :", time.Since(start2).Seconds(), "::size", len(dataInBytes))
			}
			break
		}
	}

	start3 := time.Now()
	sign := GenerateBatchSignature(node, "dkg_Secret_Leaders_Group_"+strconv.Itoa(leaderGroupNumber), node.NodeID, transactionsRequest)
	// sign := ""
	if ShowTime == true {
		Log(transactionsRequest.ID, "Signature Generation from LEADER group took :", time.Since(start3).Seconds(), "::size", len(dataInBytes))
	}
	batchResponse := api.BatchTransaction{
		Data:      transactionsRequest.Data,
		ID:        transactionsRequest.ID,
		Signature: sign,
	}

	// // ======================================      TRANSACTION TO ORACLE         ============================================================
	oracleCh := make(chan []byte)
	// transactionsResponseOracle := []BatchTransaction{}
	start4 := time.Now()
	go TransactionToOracle(node, selfContact, batchResponse, oracleCh, leaderGroupNumber)

	// for i := 0; i < len(batchResponse.Data); i++ {
	// 	tx := BatchTransaction{}
	// 	proto.Unmarshal(<-oracleCh, &tx)
	// 	transactionsResponseOracle = append(transactionsResponseOracle, tx)
	// }

	responseFromOracle := 1
	finalDataResponse := api.BatchTransaction{}
	// voteResponse := BatchTransaction{}

	for {
		select {
		case response := <-oracleCh:
			responseFromOracle--
			voteResponse := api.BatchTransaction{}
			proto.Unmarshal(response, &voteResponse)
			batchResponse = voteResponse
			dataInBytes = response
		}
		if responseFromOracle == 0 {
			if ShowTime == true {
				Log(transactionsRequest.ID, "Response from alpha oracle took :", time.Since(start4).Seconds(), "::size", len(dataInBytes))
			}
			break
			// finalDataResponse.Signature = transactionsResponseFinal

		}

	}

	if ShowTime == true {
		Log(transactionsRequest.ID, "Time taken for LEADERS - ORACLES, ORACLES - RSIP, RSIP - ORACLES :", time.Since(start12).Seconds(), "::size", len(dataInBytes))
	}
	start6 := time.Now()
	signOracle := ""
	batchResponseOracle := api.BatchTransaction{
		Data:      batchResponse.Data,
		ID:        batchResponse.ID,
		Signature: signOracle,
	}
	// ======================================      TRANSACTION RESPONSE         ============================================================
	finalCh := make(chan []byte)

	go finalResponse(node, selfContact, batchResponseOracle, finalCh, leaderGroupNumber)

	// transactionsResponseFinal := BatchTransaction{}

	// for i := 0; i < len(batchResponseOracle.Data); i++ {
	// 	tx := TransactionWithSignature{}
	// 	proto.Unmarshal(<-finalCh, &tx)
	// 	transactionsResponseFinal = append(transactionsResponseFinal, tx)
	// }

	finalResponseFromOracle := 1
	for {
		select {
		case response := <-finalCh:
			finalResponseFromOracle--
			dataInBytes = response
			proto.Unmarshal(response, &finalDataResponse)
			// finalDataResponse = voteResponse
		}
		if finalResponseFromOracle == 0 {
			if ShowTime == true {
				Log(transactionsRequest.ID, "Total time taken previous + update client in DB :", time.Since(start6).Seconds(), "::size", len(dataInBytes))
			}
			break
		}
	}

	// res, _ := proto.Marshal(&finalDataResponse)
	if ShowTime == true {
		Log(transactionsRequest.ID, "Complete batch :", time.Since(start).Seconds(), "::size", len(dataInBytes))
	}

	channel <- finalDataResponse
	// return res

	// signOracle := GenerateBatchSignature(node, "dkg_Secret_Oracles", node.NodeID, batchResponse)

	// fmt.Println("VOTE RESPONSE VOTE RESPONE VOTE RESPONSE VOTE RESPONE", transactionsResponse)

	// fmt.Println(transactionsResponse, "----------", len(transactionsRequest))

	// fmt.Println("\n Final Result")

	//	return nil
}

func GenerateBatchSignature(node *UnityNode, key string, id NodeID, transactionRequest api.BatchTransaction) string {
	secret := &Seckey{}

	getSecretKey, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(id[:])+key), nil)

	transactionRequest.Signature = ""
	transactionObjForBootNode, _ := proto.Marshal(&transactionRequest)

	secret.SetHexString(string(getSecretKey))
	sig := Sign(*secret, transactionObjForBootNode)
	return sig.GetHexString()
}

func TransactionToOracle(node *UnityNode, selfContact *api.Contact, batchResponse api.BatchTransaction, ch chan []byte, leaderGroupNumber int) {

	// fmt.Println("SHOULD BE PRINT ONLY ONCE")

	start3 := time.Now()
	numberOfOracleGroup := os.Getenv("NUMBER_OF_ORACLE_GROUPS")
	numberOfGroup, _ := strconv.Atoi(numberOfOracleGroup)

	minNo := 0
	maxNo := numberOfGroup + 1
	selectRandNumber := rand.Intn(maxNo-minNo) + minNo
	oraclesFromDB := GetValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_Epoch_1_Oracle_Group_"+strconv.Itoa(selectRandNumber)))
	transactionRequestInByte, _ := proto.Marshal(&batchResponse)

	handleData2 := make(chan []byte)
	responseFromOracle := 5

	request := api.Request{
		Action: "Transaction-To-Validate-And-Check-Threshold-From-Oracles-On-First-Oracle",
		Data:   transactionRequestInByte,
		Param:  strconv.Itoa(leaderGroupNumber),
	}
	for j := 0; j <= len(oraclesFromDB.Contact)-1; j++ {
		if oraclesFromDB.Contact[j].Category == "alpha" {
			oraclesFromDB.Contact[j].SelectedNumber = int32(selectRandNumber)
			oraclesFromDB.Contact[j].SelectedTribe = int32(j)

			go BootstrapSync(*selfContact, oraclesFromDB.Contact[j], request, handleData2, nil)

		}
	}

	for {
		select {
		case response := <-handleData2:
			voteResponse := api.BatchTransaction{}
			proto.Unmarshal(response, &voteResponse)
			if responseFromOracle == 5 {
				batchResponse = voteResponse
			} else {
				for index := 0; index < len(batchResponse.Data); index++ {
					batchResponse.Data[index].RSIP.VoteResponse = append(batchResponse.Data[index].RSIP.VoteResponse, voteResponse.Data[index].RSIP.VoteResponse...)
				}
			}
			responseFromOracle--
		}
		if responseFromOracle == 0 {
			break
		}
	}

	transactionRequestInByte, _ = proto.Marshal(&batchResponse)
	if ShowTime == true {
		Log(batchResponse.ID, "Total time taken by all alpha ORACLES took :", "::size", time.Since(start3).Seconds())
	}
	ch <- transactionRequestInByte

}

func finalResponse(node *UnityNode, selfContact *api.Contact, batchResponse api.BatchTransaction, ch chan []byte, leaderGroupNumber int) {

	listOfLeaders := GetValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_Leaders_Epoch_1_Group_"+strconv.Itoa(leaderGroupNumber)))

	transactionRequestInByte, _ := proto.Marshal(&batchResponse)

	min := 0
	max := len(listOfLeaders.Contact) - 1

	randNumber := rand.Intn(max-min) + min

	start := time.Now()

	// _, latestTx, errFromBootstrap := Bootstrap(listOfLeaders[randNumber], selfContact, "Transaction-To-Leader-After-Verification-From-Oracle", transactionRequestInByte, nil, "")
	// if errFromBootstrap != nil {
	// 	fmt.Println("List Update Error:", errFromBootstrap)
	// }

	request := api.Request{
		Action: "Transaction-To-Leader-After-Verification-From-Oracle",
		Data:   transactionRequestInByte,
	}
	latestTx := BootstrapSync(*selfContact, listOfLeaders.Contact[randNumber], request, nil, nil)

	if ShowTime == false {
		Log(batchResponse.ID, "Transaction-To-Leader-After-Verification-From-Oracle took :", time.Since(start).Seconds(), "::size", len(transactionRequestInByte))
	}
	transactionObject := api.BatchTransaction{}
	proto.Unmarshal(latestTx.Data, &transactionObject)

	transactionRequestInByte, _ = proto.Marshal(&transactionObject)
	ch <- transactionRequestInByte
}

func Transactions(node *UnityNode, selfContact *api.Contact, transactionRequest api.BatchTransaction, ch chan []byte, leaderGroupNumber int) {

	start := time.Now()

	transactionRequestInByte, _ := proto.Marshal(&transactionRequest)

	// numberOfOracleGroup := os.Getenv("NUMBER_OF_ORACLE_GROUPS")
	// numberOfGroup, _ := strconv.Atoi(numberOfOracleGroup)

	// minNo := 0
	// maxNo := numberOfGroup + 1
	// selectRandNumber := rand.Intn(maxNo-minNo) + minNo

	clients := &api.Clients{}
	listOfLeaders := GetValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_Leaders_Epoch_1_Group_"+strconv.Itoa(leaderGroupNumber)))
	response, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(node.NodeID[:])+"_Clients"), nil)

	proto.Unmarshal(response, clients)
	//oraclesFromDB := GetValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_Epoch_1_Oracle_Group_"+strconv.Itoa(selectRandNumber)))

	//transactionObjForBootNode := []byte{}

	for index := 0; index < len(transactionRequest.Data); index++ {
		voteResponse := api.VoteResponse{
			ID:            node.NodeID,
			Address:       selfContact.Address,
			TransactionID: transactionRequest.Data[index].ID,
		}
		verifiedBalance := VerifyAmount(node, *transactionRequest.Data[index], *clients)
		if verifiedBalance != "" {
			voteResponse.Vote = "NO"
			voteResponse.Reason = verifiedBalance
		} else {
			voteResponse.Vote = "YES"
		}

		transactionRequest.Data[index].Leaders.VoteResponse = append(transactionRequest.Data[index].Leaders.VoteResponse, &voteResponse)

	}

	transactionRequestInByte, _ = proto.Marshal(&transactionRequest)

	handleData := make(chan []byte)
	leadersCount := len(listOfLeaders.Contact) - 1

	request := api.Request{
		Action: "Transaction-To-Proceed",
		Data:   transactionRequestInByte,
	}

	for i := 0; i <= len(listOfLeaders.Contact)-1; i++ {

		if !Equal(listOfLeaders.Contact[i].ID, node.NodeID) {
			go BootstrapSync(*selfContact, listOfLeaders.Contact[i], request, handleData, nil)
		}
	}

	for {
		select {
		case response := <-handleData:
			leadersCount--
			voteResponses := api.VoteResponses{}
			proto.Unmarshal(response, &voteResponses)

			for index := 0; index < len(transactionRequest.Data); index++ {
				for j := 0; j < len(voteResponses.VoteResponse); j++ {
					if transactionRequest.Data[index].ID == voteResponses.VoteResponse[j].TransactionID {
						transactionRequest.Data[index].Leaders.VoteResponse = append(transactionRequest.Data[index].Leaders.VoteResponse, voteResponses.VoteResponse[j])
					}
				}
			}
		}
		if leadersCount == 0 {
			break
		}
	}

	transactionRequestInByte, _ = proto.Marshal(&transactionRequest)
	// if ShowTime == true {
	Log(transactionRequest.ID, "Vote by single LEADER took :", time.Since(start).Seconds(), "::size", len(transactionRequestInByte))
	// }
	ch <- transactionRequestInByte

	//fmt.Println(transactionRequest.Leaders)

	//fmt.Printf("%s took %v\n", "Transaction to send and proceed on all other leaders took", time.Since(start))

	// sign := GenerateSignature(node, transactionObjForBootNode, "dkg_Secret_Leaders", node.NodeID, transactionRequest)

	// proto.Unmarshal(transactionObjForBootNode, &transactionRequest)

	// transactionRequest.Signature = sign.GetHexString()
	// transactionObjForBootNode, _ = Json.MarshalIndent(transactionRequest, "", "  ")

	// if Debug == true {
	// 	fmt.Println("", "First Leader------------------------------")
	// 	fmt.Printf("%s took %v\n", "Complete transaction without storing in Db", time.Since(start))
	// }

	// transactionRequestInByte, _ = proto.Marshal(transactionRequest)
	// responseFromOracle := 1
	// handleData2 := make(chan []byte)
	// fmt.Println(len(oraclesFromDB), "oraclesFromDB")

	// var ws sync.WaitGroup
	// var m sync.Mutex

	// for j := 0; j <= len(oraclesFromDB)-1; j++ {
	// 	if oraclesFromDB[j].Category == "alpha" {
	// 		oraclesFromDB[j].SelectedNumber = selectRandNumber
	// 		if Debug == true {
	// 			fmt.Println("Is alpha oracles  ======:", oraclesFromDB[j])
	// 			fmt.Println(oraclesFromDB[j], "------------------>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>", j)
	// 		}
	// 		ws.Add(1)
	// 		//go Bootstrap(oraclesFromDB[j], selfContact, "Transaction-To-Validate-And-Check-Threshold-From-Oracles-On-First-Oracle", transactionRequestInByte, handleData2, "")

	// 		go Bootstrap2(oraclesFromDB[j], selfContact, "Transaction-To-Validate-And-Check-Threshold-From-Oracles-On-First-Oracle", transactionRequestInByte, handleData2, "", &ws, &m)
	// 	}
	// }
	// ws.Wait()

	// for {
	// 	select {
	// 	case response := <-handleData2:
	// 		responseFromOracle--
	// 		voteResponse := TransactionWithSignature{}
	// 		fmt.Println("=====================--------------===============*************", response)
	// 		proto.Unmarshal(response, &voteResponse)
	// 		transactionRequest = voteResponse
	// 	}
	// 	if responseFromOracle == 0 {
	// 		break
	// 	}
	// }

	// fmt.Println("&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&")
	// fmt.Println(transactionRequest)
	// fmt.Println("&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&")

	// transactionRequestInByte, _ = proto.Marshal(transactionRequest)
	// // signatureOfOraclesToSendLeader := GenerateSignature(node, transactionRequestInByte, "dkg_Secret_Oracles", req.Target.ID, transactionRequest)

	// transactionRequest.Signature = ""
	// transactionRequestInByteForRSIP, _ := proto.Marshal(transactionRequest)

	// min := 0
	// max := len(listOfLeaders) - 1

	// randNumber := rand.Intn(max-min) + min
	// _, latestTx, errFromBootstrap := Bootstrap(listOfLeaders[randNumber], selfContact, "Transaction-To-Leader-After-Verification-From-Oracle", transactionRequestInByteForRSIP, nil, "")
	// if errFromBootstrap != nil {
	// 	fmt.Println("List Update Error:", errFromBootstrap)
	// }

	// transactionObject := TransactionWithSignature{}
	// proto.Unmarshal(latestTx, &transactionObject)

	// newTransactionObject := TransactionWithSignature{}
	// newTransactionObject.ID = transactionObject.ID
	// newTransactionObject.From = transactionObject.From
	// newTransactionObject.To = transactionObject.To
	// newTransactionObject.Amount = transactionObject.Amount
	// newTransactionObject.Status = transactionObject.Status
	// newTransactionObject.Signature = transactionObject.Signature

	// fmt.Printf("%s took %v\n", "Finish transaction", time.Since(start))
	// transactionRequestInByte, _ = proto.Marshal(newTransactionObject)
	// ch <- transactionRequestInByte
}

/********************************************************************QUEUE-MESSAGE*******************************************************************/

func TransactionToRSIP(node *UnityNode, req api.Request, numberOfSelectedGroup int, ch chan []byte, RSIPVotes api.VoteResponses, signaturesRSIP []string, k int, transactionRequest api.BatchTransaction, oldTransactionRequest []byte) {

	// start := time.Now()
	RSIPDecimalData := api.RSIPDecimals{}
	getSelectedGroup := api.RSIPDecimal{}
	RSIPNodes := &api.Contacts{}
	channelForRSIP := make(chan []byte)

	decimalWithGroup, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"_Tribe_"+strconv.Itoa(k)+"_Decimal_DKG"), nil)
	proto.Unmarshal(decimalWithGroup, &RSIPDecimalData)

	getSelectedGroup = *RSIPDecimalData.RSIPDecimal[numberOfSelectedGroup]

	getListOfRSIPNodes, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"-Tribe_"+strconv.Itoa(k)+"_RSIP_GROUP_"+strconv.Itoa(int(getSelectedGroup.Group))), nil)

	proto.Unmarshal(getListOfRSIPNodes, RSIPNodes)
	signaturesFromOracle := GenerateBatchSignature(node, "dkg_Secret_Oracles", node.NodeID, transactionRequest)
	// signaturesFromOracle := ""
	transactionRequest.Signature = signaturesFromOracle

	// transactionRequestInByteForRSIP, _ := proto.Marshal(&transactionRequest)
	transactionRequestInByteForRSIP := oldTransactionRequest

	start12 := time.Now()

	request := api.Request{
		Action: "Transaction-To-Proceed-In-RSIP",
		Data:   transactionRequestInByteForRSIP,
	}
	for j := 0; j <= len(RSIPNodes.Contact)-1; j++ {
		if RSIPNodes.Contact[j].Category == "alpha" {
			RSIPNodes.Contact[j].SelectedNumber = int32(getSelectedGroup.Group)
			RSIPNodes.Contact[j].SelectedTribe = int32(k)

			go BootstrapSync(*req.Target, RSIPNodes.Contact[j], request, channelForRSIP, nil)
		}
	}

	responseFromChannel := 1
	for {
		select {
		case response := <-channelForRSIP:
			responseFromChannel--
			voteResponses := api.BatchTransaction{}
			proto.Unmarshal(response, &voteResponses)
			// transactionRequest.Data[index].RSIP = append(transactionRequest.Data[index].RSIP, voteResponses.Data)
			key := "_Tribe_" + strconv.Itoa(k) + "_dkg_Public_RSIP_Group_" + strconv.Itoa(int(getSelectedGroup.Group))

			// fmt.Println(key, "public")
			verified := VerifyGroupSignature(node, voteResponses, key, node.NodeID)
			// verified := ""

			if Debug == true {
				fmt.Println("VERIFIED", verified)
			}

			// signaturesRSIP = append(signaturesRSIP, voteResponses.SignatureOfRSIP...)
			for index := 0; index < len(voteResponses.Data); index++ {

				if index == 0 {
					RSIPVotes.VoteResponse = append(RSIPVotes.VoteResponse, voteResponses.Data[index].RSIP.VoteResponse...)
				}

			}

			if ShowTime == true {
				dataInBytesForSize, _ := proto.Marshal(&RSIPVotes)
				Log(transactionRequest.ID, "response for one RSIP group took :", time.Since(start12).Seconds(), "::size", len(dataInBytesForSize))

				// Log(transactionRequest.ID, "response after ", time.Since(start), len(dataInBytesForSize))
			}
		}
		if responseFromChannel == 0 {
			break
		}
	}

	dataInBytes, _ := proto.Marshal(&RSIPVotes)
	ch <- dataInBytes
	// }

	// voteResponse.ThresholdOfRSIP = strconv.Itoa(thresholdOfRSIP) + " out of " + strconv.Itoa(len(transactionRequest.RSIP)) + " RSIP"
}

func BootstrapSync(self api.Contact, target *api.Contact, request api.Request, channel chan []byte, wg *sync.WaitGroup) *api.BootstarpSyncResponse {

	conn, err := grpc.Dial(target.Address, grpc.WithInsecure(), grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(MaxMsgSize),
		grpc.MaxCallSendMsgSize(MaxMsgSize),
	))

	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	client := api.NewPingClient(conn)
	defer conn.Close()

	request.Sender = &self
	request.Target = target

	// reqInBytes, _ := proto.Marshal(&request)

	response := &api.BootstarpSyncResponse{}
	if request.Action != "" {
		response, err = client.SyncAction(context.Background(), &request)
	} else {
		response, err = client.Sync(context.Background(), &request)
	}
	if err != nil {
		log.Fatalf("Error when calling Sync: %s", err)
	}

	if wg != nil {
		wg.Done()
	}

	// res := api.BootstarpSyncResponse{}
	// proto.Unmarshal(response.Value, &res)

	if channel != nil {
		channel <- response.Data
	}

	return response
}

// Struct for g-RPC
type Request struct {
	Sender api.Contact
	Target api.Contact
	Action string
	Data   []byte
	IsDKG  bool
	Param  string
}

type Response struct {
	Contacts Contacts
	Data     []byte
}

func Equal(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func EqualString(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

// func EqualNodeID(a, b api.NodeID) bool {
// 	if len(a) != len(b) {
// 		return false
// 	}
// 	for i, v := range a {
// 		if v != b[i] {
// 			return false
// 		}
// 	}
// 	return true
// }
