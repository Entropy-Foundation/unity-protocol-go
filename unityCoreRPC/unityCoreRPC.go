package unityCoreRPC

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	_ "net/http/pprof"
	"os"
	// "runtime"
	"strconv"
	"sync"
	"time"

	// . "github.com/dfinity/go-dfinity-crypto/groupsig"
	"github.com/golang/protobuf/proto"

	// guuid "github.com/google/uuid"
	"google.golang.org/grpc"

	"github.com/unity-go/api"
	// . "github.com/unity-go/dkg"
	. "github.com/unity-go/findValues"
	. "github.com/unity-go/go-bls"
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

// For ERASURE
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

// For ERASURE

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

// Initialize gRPC request
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

// This function will check the amount of client
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

/*
  This function calls on batch proposer
  It will send batch to oracles, leaders and RSIPs
  And collection the responses
*/
func QueueHandleTransaction(transactionsRequest api.BatchTransaction, node *UnityNode, selfContact *api.Contact, leaderGroupNumber int, channel chan api.BatchTransaction, nodeBlock *NodeBlock) {

	ch := make(chan []byte)
	listOfLeaders := api.Contacts{}
	listOfLeaders.Contact = append(nodeBlock.LeaderList[strconv.Itoa(leaderGroupNumber)].Contact[:0:0], nodeBlock.LeaderList[strconv.Itoa(leaderGroupNumber)].Contact...)

	paraBytes, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(node.NodeID[:])+"bls_params"), nil)
	params, _ := ParamsFromBytes(paraBytes)

	pairing := GenPairing(params)
	sysBytes, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(node.NodeID[:])+"bls_system"), nil)
	system, _ := SystemFromBytes(pairing, sysBytes)

	start2 := time.Now()
	go Transactions(node, selfContact, transactionsRequest, ch, leaderGroupNumber, listOfLeaders, system)
	responseFromLeader := 1
OuterFirst:
	for {
		response := <-ch
		responseFromLeader--
		voteResponse := api.BatchTransaction{}
		proto.Unmarshal(response, &voteResponse)
		transactionsRequest = voteResponse
		voteResponse = api.BatchTransaction{}
		if responseFromLeader == 0 {
			if ShowTime == true {
				Log(transactionsRequest.ID, "Response from all leaders took :", time.Since(start2).Seconds())
			}
			break OuterFirst
		}
	}

	batchResponse := api.BatchTransaction{
		Data:      transactionsRequest.Data,
		ID:        transactionsRequest.ID,
		Signature: transactionsRequest.Signature,
	}

	// // ======================================      TRANSACTION TO ORACLE         ============================================================
	oracleCh := make(chan []byte)

	start4 := time.Now()
	go TransactionToOracle(node, selfContact, batchResponse, oracleCh, leaderGroupNumber, nodeBlock)

	responseFromOracle := 1
	finalDataResponse := api.BatchTransaction{}
	// voteResponse := BatchTransaction{}

OuterSecond:
	for {
		response := <-oracleCh
		responseFromOracle--
		voteResponse := api.BatchTransaction{}
		proto.Unmarshal(response, &voteResponse)
		batchResponse = voteResponse
		voteResponse = api.BatchTransaction{}
		if responseFromOracle == 0 {
			if ShowTime == true {
				Log(batchResponse.ID, "Transaction to proceed in All Oracle (Oracle => RSIP, RSIP => Oracle(In Back Process) , Oracle => Leader(In Back Process)) took :", time.Since(start4).Seconds())
			}
			break OuterSecond
			// finalDataResponse.Signature = transactionsResponseFinal

		}

	}

	// // ======================================      TRANSACTION RESPONSE         ============================================================
	finalCh := make(chan []byte)

	go finalResponse(node, selfContact, finalCh, leaderGroupNumber, transactionsRequest.ID, listOfLeaders, system)

	finalResponseFromOracle := 1
OuterThird:
	for {
		response := <-finalCh
		finalResponseFromOracle--
		proto.Unmarshal(response, &finalDataResponse)
		// finalDataResponse = voteResponse
		if finalResponseFromOracle == 0 {
			if ShowTime == true {
				// Log(transactionsRequest.ID, "Response after updating to DB", time.Since(start6))
			}

			break OuterThird
		}
	}

	// To Do
	GenerateAggregateSignatureFromRSIPSignatures(node, transactionsRequest.ID, system, leaderGroupNumber, nodeBlock)
	//

	system.Free()
	pairing.Free()
	params.Free()

	listOfLeaders = api.Contacts{}
	channel <- finalDataResponse
	finalDataResponse = api.BatchTransaction{}
}

// This function will generate BLS signature
func GenerateBLSBatchSignature(node *UnityNode, selfContact *api.Contact, transactionRequest api.BatchTransaction, key []byte, Contact api.Contacts, n int, t int) []byte {

	memberIds := rand.Perm(n)[:t]

	paraBytes, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(node.NodeID[:])+"bls_params"), nil)
	params, _ := ParamsFromBytes(paraBytes)

	pairing := GenPairing(params)
	sysBytes, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(node.NodeID[:])+"bls_system"), nil)
	system, _ := SystemFromBytes(pairing, sysBytes)

	transactionRequest.Signature = []byte{}
	transactionObjForBootNode, _ := proto.Marshal(&transactionRequest)
	hash := sha256.Sum256(transactionObjForBootNode)

	shares := make([]Signature, t)

	for i := 0; i < len(memberIds); i++ {
		request := api.Request{
			Action: "retrive_bls_secret",
			Data:   key,
		}
		response := BootstrapSync(*selfContact, Contact.Contact[memberIds[i]], request, nil, nil, node)
		LeaderPrivateKey := system.PrivateFromBytes(response.Data)
		shares[i] = Sign(hash, LeaderPrivateKey)
	}

	transactionObjForBootNode = nil
	transactionRequest = api.BatchTransaction{}

	signature, _ := Threshold(shares, memberIds, system)

	sig := system.SigToBytes(signature)

	// signature.Free()
	return sig
}

// This function will verify BLS signature
func VerifyBLSBatchSignature(node *UnityNode, hash [32]byte, sign []byte, key []byte) string {
	paraBytes, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(node.NodeID[:])+"bls_params"), nil)
	params, _ := ParamsFromBytes(paraBytes)

	pairing := GenPairing(params)
	sysBytes, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(node.NodeID[:])+"bls_system"), nil)
	system, _ := SystemFromBytes(pairing, sysBytes)

	// getPublicKey, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(node.NodeID[:])+key), nil)
	getPublicKey := key

	LeaderGroupPublicKeyKey, _ := system.PublicFromBytes(getPublicKey)
	signature, _ := system.SigFromBytes(sign)
	if !Verify(signature, hash, LeaderGroupPublicKeyKey) {
		return "Failed to verify signature"
	}

	// signature.Free()
	LeaderGroupPublicKeyKey.Free()
	system.Free()
	pairing.Free()
	params.Free()

	return ""
}

// This function will return deterministic values
func DeterministicValue(hash, sign []byte) uint32 {
	decimalOfHash := binary.BigEndian.Uint32(hash)
	decimalOfSign := binary.BigEndian.Uint32(sign)

	return (decimalOfHash + decimalOfSign) % 4
}

// This will validate batch on ORACLES
func TransactionToOracle(node *UnityNode, selfContact *api.Contact, batchResponse api.BatchTransaction, ch chan []byte, leaderGroupNumber int, nodeBlock *NodeBlock) {

	// fmt.Println("SHOULD BE PRINT ONLY ONCE")

	//	start3 := time.Now()
	numberOfOracleGroup := os.Getenv("NUMBER_OF_ORACLE_GROUPS")
	numberOfGroup, _ := strconv.Atoi(numberOfOracleGroup)

	// thresholdOfOracle := 7

	//
	// memberIds := rand.Perm(thresholdOfOracle)[:thresholdOfOracle]
	// paraBytes, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(node.NodeID[:])+"bls_params"), nil)
	// params, _ := ParamsFromBytes(paraBytes)

	// pairing := GenPairing(params)
	// sysBytes, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(node.NodeID[:])+"bls_system"), nil)
	// system, _ := SystemFromBytes(pairing, sysBytes)

	//

	minNo := 0
	maxNo := numberOfGroup + 1
	selectRandNumber := rand.Intn(maxNo-minNo) + minNo
	// oraclesFromDB := GetValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_Epoch_1_Oracle_Group_"+strconv.Itoa(selectRandNumber)))
	oraclesFromDB := nodeBlock.OracleList[strconv.Itoa(selectRandNumber)]
	sign := batchResponse.Signature
	batchResponse.Signature = []byte{}
	transactionRequestInByte, _ := proto.Marshal(&batchResponse)

	handleData2 := make(chan []byte)
	numberOfResponseFromOracle := 1
	// SignaturesFromOracle := []Signature{}
	// hash := [32]byte{}

	request := api.Request{
		Action:              "Transaction-To-Validate-And-Check-Threshold-From-Oracles-On-First-Oracle",
		Data:                transactionRequestInByte,
		Param:               strconv.Itoa(leaderGroupNumber),
		Signature:           sign,
		BatchProposerLeader: selfContact.Address,
	}
	for j := 0; j <= len(oraclesFromDB.Contact)-1; j++ {
		if oraclesFromDB.Contact[j].Category == "alpha" {
			oraclesFromDB.Contact[j].SelectedNumber = int32(selectRandNumber)
			oraclesFromDB.Contact[j].SelectedTribe = int32(j)

			go BootstrapSync(*selfContact, oraclesFromDB.Contact[j], request, handleData2, nil, node)

		}
	}

OuterFourth:
	for {
		response := <-handleData2

		fmt.Println("Back In Leader", response)

		numberOfResponseFromOracle--
		if numberOfResponseFromOracle == 0 {
			transactionRequestInByte = nil
			oraclesFromDB = &api.Contacts{}
			break OuterFourth
		}
	}

	// if ShowTime == true {
	// 	Log(batchResponse.ID, "Transaction to proceed in All Oracle (Oracle => RSIP, RSIP => Oracle(In Back Process) , Oracle => Leader(In Back Process)) took :", time.Since(start3).Seconds())
	// }

	ch <- []byte("Done")

}

// This will call on batch proposer at last
func finalResponse(node *UnityNode, selfContact *api.Contact, ch chan []byte, leaderGroupNumber int, batchId string, listOfLeaders api.Contacts, system System) {

	//start := time.Now()
	shares := []Signature{}

	thresholdOfLeaders, _ := strconv.Atoi(os.Getenv("NUMBER_OF_LEADERS"))
	batch := api.BatchTransaction{}

	memberIds := rand.Perm(len(listOfLeaders.Contact))[:thresholdOfLeaders]
	for index := 0; index < len(listOfLeaders.Contact); index++ {
		leaderShares := "Leader_shares_" + batchId + strconv.Itoa(index)
		sharesOfLeaders, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(node.NodeID[:])+leaderShares), nil)
		partialSign, _ := system.SigFromBytes(sharesOfLeaders)
		shares = append(shares, partialSign)
	}

	// if ShowTime == true {
	// 	Log(batchId, "Time to take all shares from leader took :", time.Since(start).Seconds())
	// }

	start2 := time.Now()

	signature, err := Threshold(shares, memberIds, system)

	for i := 0; i < len(shares); i++ {
		shares[i].Free()
	}

	if ShowTime == true {
		Log(batchId, "Generate Signature by leader took :", time.Since(start2).Seconds())
	}

	if err != nil {
		fmt.Println(err, "Signature Generation Error")
	}

	shareToBytes := system.SigToBytes(signature)
	batchFromProposer, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(node.NodeID[:])+"_batch_"+batchId), nil)
	proto.Unmarshal(batchFromProposer, &batch)
	batch.Signature = shareToBytes
	// fmt.Println(len(batch.Data), batch.Signature)

	batchInBytes, _ := proto.Marshal(&batch)

	// update client balance to all nodes
	// start3 := time.Now()

	// var wg2 sync.WaitGroup

	// // go func() {
	// for i := 0; i <= len(contactsFromOwnDb.Contact)-1; i++ {
	// 	wg2.Add(1)
	// 	// if contactsFromOwnDb[i].ID != req.Target.ID {

	// 	request := api.Request{
	// 		Action: "Store-Transaction-Across-The-Network",
	// 		Data:   transactionInByte,
	// 		Param:  batch.ID,
	// 	}

	// 	go BootstrapSync(*selfContact, contactsFromOwnDb.Contact[i], request, nil, &wg2, node)
	// }
	// wg2.Wait()

	// var wg2 sync.WaitGroup

	// loopIndex := len(latestTransactions) / 500

	// for index := 0; index < loopIndex; index++ {
	// 	var latestTransactionsToSend []interface{}
	// 	latestTransactionsToSend = append(latestTransactionsToSend, latestTransactions[:500]...)
	// 	latestTransactions = append(latestTransactions, latestTransactions[500:]...)
	// 	transactionInByte, _ := json.Marshal(&latestTransactionsToSend)

	// 	fmt.Println("latestTransactions", len(latestTransactions))

	// 	for i := 0; i <= len(contactsFromOwnDb.Contact)-1; i++ {
	// 		wg2.Add(1)
	// 		request2 := api.Request{
	// 			Action: "Store-Transaction-Across-The-Network",
	// 			Data:   transactionInByte,
	// 		}

	// 		// }
	// 		go BootstrapSync(*selfContact, contactsFromOwnDb.Contact[i], request2, nil, &wg2, node)
	// 	}

	// }
	// wg2.Wait()

	// }()

	// count := len(contactsFromOwnDb.Contact) * 2
	// for {
	// 	response := <-chanName
	// 	count--
	// 	msg := api.MyData{}
	// 	proto.Unmarshal(response, &msg)
	// 	if count == 0 {

	// 		break
	// 	}
	// }

	batch = api.BatchTransaction{}
	// contactsFromOwnDb = api.Contacts{}
	// clients = api.Clients{}

	ch <- batchInBytes
}

// This functions will call on batch proposer to generate and send shares to oracles
func Transactions(node *UnityNode, selfContact *api.Contact, transactionRequest api.BatchTransaction, ch chan []byte, leaderGroupNumber int, listOfLeaders api.Contacts, system System) {

	start := time.Now()

	transactionRequestInByte, _ := proto.Marshal(&transactionRequest)

	// numberOfOracleGroup := os.Getenv("NUMBER_OF_ORACLE_GROUPS")
	// numberOfGroup, _ := strconv.Atoi(numberOfOracleGroup)

	// minNo := 0
	// maxNo := numberOfGroup + 1
	// selectRandNumber := rand.Intn(maxNo-minNo) + minNo

	clients := &api.Clients{}
	// listOfLeaders := GetValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_Leaders_Epoch_1_Group_2"))

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

			// Remove Invalid Transaction By Batch Proposer
			// transactionRequest.Data = append(transactionRequest.Data[:index], transactionRequest.Data[index+1:]...)

		} else {
			voteResponse.Vote = "YES"
		}

		transactionRequest.Data[index].Leaders.VoteResponse = append(transactionRequest.Data[index].Leaders.VoteResponse, &voteResponse)

	}

	if ShowTime == true {
		Log(transactionRequest.ID, "Vote by single LEADER on Transactions took :", time.Since(start).Seconds(), "::size", len(transactionRequestInByte))
		// Log(transactionRequest.ID, "Vote by single LEADER on Transactions took :", time.Since(start).Seconds(), len(transactionRequestInByte))
	}

	// transactionRequestInByte, _ = proto.Marshal(&transactionRequest)

	start2 := time.Now()
	thresholdOfLeaders := len(listOfLeaders.Contact)
	// Generate Share of Aggregates Signature by Batch Proposer
	memberIds := rand.Perm(thresholdOfLeaders)[:thresholdOfLeaders]

	shares := []Signature{}

	key := "bls_leader_group_secret"
	transactionRequest.Signature = []byte{}
	transactionObjForBootNode, _ := proto.Marshal(&transactionRequest)
	hash := sha256.Sum256(transactionObjForBootNode)
	if ShowTime == true {
		Log(transactionRequest.ID, "Generate hash by single LEADER (First Leader) took :", time.Since(start2).Seconds(), "::size", len(hash))
	}

	// Store batch in levelDB
	node.ValuesDB.Put([]byte(hex.EncodeToString(node.NodeID[:])+"_batch_"+transactionRequest.ID), transactionObjForBootNode, nil)

	shares = append(shares, GenerateShares(node, hash, key, system))

	if ShowTime == true {
		Log(transactionRequest.ID, "Generate share by single LEADER took :", time.Since(start2).Seconds(), "::size", len(shares))
	}

	getShares := make(chan []byte)
	leadersCount := len(listOfLeaders.Contact) - 1

	request := api.Request{
		Action: "Transaction-To-Proceed",
		Data:   transactionObjForBootNode,
	}

	for i := 0; i <= len(listOfLeaders.Contact)-1; i++ {

		if !Equal(listOfLeaders.Contact[i].ID, node.NodeID) {
			go BootstrapSync(*selfContact, listOfLeaders.Contact[i], request, getShares, nil, node)
		}
	}

OuterSixth:
	for {
		response := <-getShares
		leadersCount--

		signShareFromOtherLeaders, _ := system.SigFromBytes(response)

		shares = append(shares, signShareFromOtherLeaders)
		if leadersCount == 0 {
			break OuterSixth
		}
	}

	// Genrate aggregated signature by batch proposer
	start3 := time.Now()

	signature, err := Threshold(shares, memberIds, system)

	for i := 0; i < leadersCount; i++ {
		shares[i].Free()
	}

	if ShowTime == true {
		Log(transactionRequest.ID, "Generate aggregated signature by batch proposer took :", time.Since(start3).Seconds())
	}

	if err != nil {
		fmt.Println(err, "Signature Generation Error")
	}

	shareToBytes := system.SigToBytes(signature)
	transactionRequest.Signature = shareToBytes

	transactionRequestInByte, _ = proto.Marshal(&transactionRequest)

	transactionRequest = api.BatchTransaction{}
	transactionObjForBootNode = nil
	// signature.Free()

	clients = nil

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

func GenerateShares(node *UnityNode, hash [32]byte, key string, system System) Signature {

	getSecretKey, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(node.NodeID[:])+key), nil)

	LeaderGroupPrivateKey := system.PrivateFromBytes(getSecretKey)
	return Sign(hash, LeaderGroupPrivateKey)

}

// This function will send transaction to RSIP alphas
func TransactionToRSIP(node *UnityNode, req api.Request, numberOfSelectedGroup int, ch chan []byte, RSIPVotes api.VoteResponses, k int, transactionRequest api.BatchTransaction, aggregatedSignatureInBytes []byte, selectedOracleGroup int, selectedLeaderGroup string, BatchProposerLeader string, nodeBlock *NodeBlock) {

	start := time.Now()
	// RSIPDecimalData := api.RSIPDecimals{}totalOracles, _ := strconv.Atoi(os.Getenv("NUMBER_OF_ORACLES"))
	// totalOracles, _ := strconv.Atoi(os.Getenv("NUMBER_OF_ORACLES"))
	// totalOracleGroups, _ := strconv.Atoi(os.Getenv("NUMBER_OF_ORACLE_GROUPS"))
	// numberOfRSIPGroups, _ := strconv.Atoi(os.Getenv("NUMBER_OF_RSIP_GROUP"))
	// thresholdOfOracle, _ := strconv.Atoi(os.Getenv("THRESHOLD_OF_ORACLES"))
	// numberOfOracles := thresholdOfOracle * 5
	numberOfOracles := 1

	getSelectedGroup := api.RSIPDecimal{}
	RSIPNodes := &api.Contacts{}
	channelForRSIP := make(chan []byte)

	// channel := make(chan []byte)
	// aggregateSigntureFromRSIP := []byte{}

	// TO-DO
	// decimalWithGroup, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"_Tribe_"+strconv.Itoa(k)+"_Decimal_DKG"), nil)
	// proto.Unmarshal(decimalWithGroup, &RSIPDecimalData)

	// getSelectedGroup = *RSIPDecimalData.RSIPDecimal[numberOfSelectedGroup]

	// getListOfRSIPNodes := &api.Contacts{}
	// getListOfRSIPNodes, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(req.Target.ID[:])+"-Tribe_"+strconv.Itoa(k)+"_RSIP_GROUP_"+strconv.Itoa(int(numberOfSelectedGroup))), nil)

	for index := 0; index < len(nodeBlock.RSIPList[strconv.Itoa(k)]); index++ {
		if nodeBlock.RSIPList[strconv.Itoa(k)][index].Group == strconv.Itoa(int(numberOfSelectedGroup)) {
			for l := 0; l < len(nodeBlock.RSIPList[strconv.Itoa(k)][index].Contacts.Contact); l++ {
				RSIPNodes.Contact = append(RSIPNodes.Contact, nodeBlock.RSIPList[strconv.Itoa(k)][index].Contacts.Contact[l])
			}
		}
	}

	// proto.Unmarshal(getListOfRSIPNodes, RSIPNodes)
	//signaturesFromOracle := GenerateBatchSignature(node, "dkg_Secret_Oracles", node.NodeID, transactionRequest)

	// listOfOracles := GetValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_Epoch_1_Oracle_Group_"+strconv.Itoa(selectedOracleGroup)))
	// n := 7
	// t := 5

	// batchData := api.BatchTransaction{}
	// proto.Unmarshal(transactionRequest, &batchData)

	// keyInByte, _ := json.Marshal("bls_oracle_secret")
	// sign := GenerateBLSBatchSignature(node, req.Sender, batchData, keyInByte, listOfOracles, n, t)
	// //END

	// transactionRequest.Signature = sign

	transactionRequestInByteForRSIP, _ := proto.Marshal(&transactionRequest)
	// transactionRequestInByteForRSIP := oldTransactionRequest

	// start12 := time.Now()
	// start := time.Now()

	LeaderGroup, _ := strconv.Atoi(selectedLeaderGroup)
	request := api.Request{
		Action:              "Transaction-To-Proceed-In-RSIP-ALPHA",
		Data:                transactionRequestInByteForRSIP,
		Param:               strconv.Itoa(int(req.Target.SelectedNumber)),
		Signature:           aggregatedSignatureInBytes,
		RSIPGroup:           int32(numberOfSelectedGroup),
		LeaderGroup:         int32(LeaderGroup),
		BatchProposerLeader: BatchProposerLeader,
	}

	for j := 0; j <= len(RSIPNodes.Contact)-1; j++ {
		if RSIPNodes.Contact[j].Category == "alpha" {
			request.Index = int32(j)
			RSIPNodes.Contact[j].SelectedNumber = int32(getSelectedGroup.Group)
			RSIPNodes.Contact[j].SelectedTribe = int32(k)

			go BootstrapSync(*req.Target, RSIPNodes.Contact[j], request, channelForRSIP, nil, node)
		}
	}

	responseFromChannel := numberOfOracles
OuterSeven:
	for {
		response := <-channelForRSIP
		responseFromChannel--

		if response != nil {
			fmt.Println(responseFromChannel, "Transaction-To-Proceed-In-RSIP-ALPHA")
			responseFromChannel = 0
		}
		if responseFromChannel == 0 {

			if ShowTime == true {
				Log(transactionRequest.ID, "Transaction to proceed in All RSIP took :", time.Since(start).Seconds())
			}

			RSIPNodes = nil
			transactionRequestInByteForRSIP = nil
			transactionRequest = api.BatchTransaction{}

			ch <- []byte("Done")
			break OuterSeven
		}
	}

	// dataInBytes := aggregateSigntureFromRSIP
	// ch <- dataInBytes
	// }

	// voteResponse.ThresholdOfRSIP = strconv.Itoa(thresholdOfRSIP) + " out of " + strconv.Itoa(len(transactionRequest.RSIP)) + " RSIP"
}

/*
 This function creates TCP connection between two nodes
 It will send request and handle response from server
*/
func BootstrapSync(self api.Contact, target *api.Contact, request api.Request, channel chan []byte, wg *sync.WaitGroup, node *UnityNode) *api.BootstarpSyncResponse {

	var conn *grpc.ClientConn
	conn = &grpc.ClientConn{}
	var err error

	// noConn := ClientConnection{}

	// node.

	node.MapMutex.Lock()
	if node.Connections[target.Address] != nil {
		conn = node.Connections[target.Address]
	} else {
		conn = nil
	}

	if conn == nil {

		// fmt.Println("New connection", len(node.Connections))

		// 	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		// defer cancel()

		conn, err = grpc.Dial(target.Address, grpc.WithInsecure(), grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(MaxMsgSize),
			grpc.MaxCallSendMsgSize(MaxMsgSize),
		))

		// // fmt.Println("else")
		if err != nil {
			log.Fatalf("did not connect: %v", err)
		}

		node.Connections[target.Address] = conn
	} else {
		// fmt.Println("Already connection exists")
	}
	node.MapMutex.Unlock()

	client := api.NewPingClient(conn)

	// defer conn.Close()

	request.Sender = &self
	request.Target = target

	// reqInBytes, _ := proto.Marshal(&request)

	response := &api.BootstarpSyncResponse{}
	if request.Action != "" {
		response, err = client.SyncAction(context.Background(), &request)
	} else {
		response, err = client.Sync(context.Background(), &request)
	}

	// request = api.Request{}
	if err != nil {
		log.Fatalf("Error when calling Sync: %s", err)
	}

	if wg != nil {
		wg.Done()
	}

	// res := api.BootstarpSyncResponse{}
	// proto.Unmarshal(response.Value, &res)

	if channel != nil && response.Data != nil {
		// fmt.Println("INSIDE")
		channel <- response.Data
		// response.Data = nil
		// fmt.Println("response.Data", response.Data)
	}

	// runtime.GC()

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

func GenerateAggregateSignatureFromRSIPSignatures(node *UnityNode, batchId string, system System, leaderGroupNumber int, nodeBlock *NodeBlock) {
	signaturesInBytes := api.Shares{}
	shares := []Signature{}

	RSIpSignatures := "RSIP_Groups_Signatures" + batchId
	signaturesOfRSIP, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(node.NodeID[:])+RSIpSignatures), nil)
	hash, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(node.NodeID[:])+"_HASH"), nil)

	proto.Unmarshal(signaturesOfRSIP, &signaturesInBytes)
	fmt.Println(len(signaturesInBytes.Share), "IN PROPOSER")

	for index := 0; index < len(signaturesInBytes.Share); index++ {
		sign, _ := system.SigFromBytes(signaturesInBytes.Share[index].Share)
		shares = append(shares, sign)
	}

	memberIds := rand.Perm(5)[:5]

	signature, _ := Threshold(shares, memberIds, system)
	shareToBytes := system.SigToBytes(signature)
	fmt.Println(signature, "Aggregate signature")

	keyInByte := nodeBlock.LeadersPublicKey[strconv.Itoa(leaderGroupNumber)]
	hashIn32 := [32]byte{}
	for index := 0; index < len(hash); index++ {
		hashIn32[index] = hash[index]
	}

	verified := VerifyBLSBatchSignature(node, hashIn32, shareToBytes, keyInByte)
	fmt.Println("VERIFY", verified)

}
