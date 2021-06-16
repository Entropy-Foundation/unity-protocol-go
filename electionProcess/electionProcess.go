package electionProcess

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"sync"

	"github.com/golang/protobuf/proto"

	"github.com/unity-go/api"
	// . "github.com/unity-go/dkg"
	. "github.com/unity-go/findValues"

	// . "github.com/unity-go/transport"
	"math/rand"
	"regexp"
	"time"

	. "github.com/unity-go/go-bls"
	. "github.com/unity-go/unityCoreRPC"
	. "github.com/unity-go/unityNode"
	. "github.com/unity-go/util"
)

func Regex(s string) string {
	re := regexp.MustCompile(`\n`)
	return re.ReplaceAllString(s, "$1W")
}

var SetupTime time.Time

// This function handles whole election preocess
func Scheduler(node *UnityNode, selfContact api.Contact, nodeBlock *NodeBlock) {

	// if node.CurrentElectionBlock == 0 {
	node.CurrentElectionBlock = node.CurrentElectionBlock + 1
	nodeBlock.BlockNumber = nodeBlock.BlockNumber + 1
	// }

	nodeBlock.RSIPPublicKey = make(map[string][]GroupsPublicKeys)
	nodeBlock.RSIPList = make(map[string][]GroupList)
	fmt.Println("START-NETWORK-SETUP")
	SetupTime = time.Now()
	fmt.Println("Debug-Value", Debug)

	if Debug == true {
		fmt.Println("", "First Oracle------------------------------")
	}

	Oracles := os.Getenv("NUMBER_OF_ORACLES")
	OraclesGroups := os.Getenv("NUMBER_OF_ORACLE_GROUPS")
	numberOfAlphaOracles := os.Getenv("NUMBER_OF_ALPHA")
	// numberOfGroups, _ := strconv.Atoi(os.Getenv("NUMBER_OF_GROUPS_OF_NODES"))

	numberOfOracles, _ := strconv.Atoi(Oracles)
	numberOfOracleGroups, _ := strconv.Atoi(OraclesGroups)
	numberOfAlphaNodes, _ := strconv.Atoi(numberOfAlphaOracles)

	if Debug == true {
		fmt.Println("")
		fmt.Println("-------------------Starting Election Process Of Oracles------------------")
	}
	// Code for scheduling event
	ticker := time.NewTicker(50 * time.Millisecond)
	quit := make(chan bool)
	a := 0
	rand.Seed(21)
	contacts := append(nodeBlock.AllNodesInNetwork[:0:0], nodeBlock.AllNodesInNetwork...)
	contacts2 := append(nodeBlock.AllNodesInNetwork[:0:0], nodeBlock.AllNodesInNetwork...)
	listOfOracles := &api.Contacts{}
	contactsFromOwnDb := api.Contacts{
		Contact: contacts,
	}
	contactsFromOwnDbNew := api.Contacts{
		Contact: contacts2,
	}

	fmt.Println("contactsFromOwnDbNew", len(contactsFromOwnDbNew.Contact))

	// Notify for election process start

	var wgForStartingProcess sync.WaitGroup

	requestForStartingProccess := api.Request{
		Action: "STARTING_ELECTION_PROCESS",
		Index:  int32(nodeBlock.BlockNumber),
	}
	for l := 0; l < len(contactsFromOwnDbNew.Contact); l++ {

		if contactsFromOwnDbNew.Contact[l].Address != selfContact.Address {
			wgForStartingProcess.Add(1)

			go BootstrapSync(selfContact, contactsFromOwnDbNew.Contact[l], requestForStartingProccess, nil, &wgForStartingProcess, node)
		}

	}
	wgForStartingProcess.Wait()

	// For BLS grouping
	params := GenParamsTypeF(32)
	pairing := GenPairing(params)
	system, _ := GenSystem(pairing)

	paramsToBytesData, _ := params.ToBytes()
	request := api.Request{
		Action: "bls_params",
		Data:   paramsToBytesData,
	}

	systemToBytesData := system.ToBytes()
	request2 := api.Request{
		Action: "bls_system",
		Data:   systemToBytesData,
	}

	var wg sync.WaitGroup

	for i := 0; i <= len(contactsFromOwnDbNew.Contact)-1; i++ {
		wg.Add(2)
		go BootstrapSync(selfContact, contactsFromOwnDbNew.Contact[i], request, nil, &wg, node)
		go BootstrapSync(selfContact, contactsFromOwnDbNew.Contact[i], request2, nil, &wg, node)
	}
	wg.Wait()

	go func() {
		for {
			select {
			case <-ticker.C:
				startTime := time.Now()
				a++
				min := 0
				max := len(contactsFromOwnDbNew.Contact)
				randNumber := rand.Intn(max-min) + min
				// selectedContact := &contactsFromOwnDbNew[randNumber]
				// fmt.Println(*selectedContact, len(contactsFromOwnDbNew))
				// selectedContact.IsOracle = true
				selectedContactToChange := contactsFromOwnDbNew.Contact[randNumber]
				selectedContactToChange.IsOracle = true

				listOfOracles.Contact = append(listOfOracles.Contact, selectedContactToChange)
				contactsFromOwnDbNew.Contact = append(contactsFromOwnDbNew.Contact[:randNumber], contactsFromOwnDbNew.Contact[randNumber+1:]...)

				if Debug == true {
					fmt.Println("")
					fmt.Println(a, "selected node is", *selectedContactToChange, len(contactsFromOwnDb.Contact))
				}
				// dataToStoreInSelf, _ := proto.Marshal(&contactsFromOwnDb)
				// err := PutValuesById(node, []byte(string(node.NodeID[:])), dataToStoreInSelf)
				// if err != nil {
				// 	return
				// }
				// dataToStoreInSelf = nil

				if Debug == true {
					fmt.Println("----------------------Time to process and select single Oracle", time.Since(startTime))
				}
				if a == numberOfOracles {
					if Debug == true {
						fmt.Println("----------------------length of Oracles", len(listOfOracles.Contact))
					}
					dataForListOfOracles, _ := proto.Marshal(listOfOracles)
					request := api.Request{
						Action: "Oracle-List-Save",
						Data:   dataForListOfOracles,
					}
					for i := 0; i <= len(contactsFromOwnDb.Contact)-1; i++ {

						go BootstrapSync(selfContact, contactsFromOwnDb.Contact[i], request, nil, nil, node)

					}

					dataForListOfOracles = nil
					t := time.Now()
					t.Second()
					lengthOfGroup := len(listOfOracles.Contact) / (numberOfOracleGroups + 1)
					listOfOraclesToFormGroups := &api.Contacts{}
					for k := 0; k <= len(listOfOracles.Contact)-1; k++ {
						listOfOraclesToFormGroups.Contact = append(listOfOraclesToFormGroups.Contact, listOfOracles.Contact[k])
					}

					for len(listOfOraclesToFormGroups.Contact) > 0 {
						for i := 0; i <= numberOfOracleGroups; i++ {
							numberOfAlpha := numberOfAlphaNodes
							listOfNodesOfTribe := &api.Contacts{}

							for m := 0; m < lengthOfGroup; m++ {
								numberOfAlpha--
								randNum := 0
								minNew := 0
								maxNew := len(listOfOraclesToFormGroups.Contact)
								if maxNew > 0 {
									randNum = rand.Intn(maxNew-minNew) + minNew
								}
								if len(listOfOraclesToFormGroups.Contact) != 0 {
									if Debug == true {
										fmt.Println(randNum, "randNum")
										fmt.Println(numberOfAlpha, "numberOfAlpha")
									}
									selectedNode := listOfOraclesToFormGroups.Contact[randNum]
									if numberOfAlpha > 0 {
										selectedNode.Category = "alpha"
									} else {
										selectedNode.Category = "beta"
									}

									listOfOraclesToFormGroups.Contact = append(listOfOraclesToFormGroups.Contact[:randNum], listOfOraclesToFormGroups.Contact[randNum+1:]...)

									// listOfNodesOfTribe := GetValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_Epoch_1_Oracle_Group_"+strconv.Itoa(i)))
									listOfNodesOfTribe.Contact = append(listOfNodesOfTribe.Contact, selectedNode)

									maxNew--

								}
							}

							// update node type on nodeBlock

							var wgForNodeType sync.WaitGroup
							requestForNodeType := api.Request{
								Action: "NodeType",
								Param:  "IsOracle",
							}
							for j := 0; j <= len(listOfNodesOfTribe.Contact)-1; j++ {

								if listOfNodesOfTribe.Contact[j].Address != selfContact.Address {
									wgForNodeType.Add(1)

									go BootstrapSync(selfContact, listOfNodesOfTribe.Contact[j], requestForNodeType, nil, &wgForNodeType, node)
								}

							}
							wgForNodeType.Wait()

							listOfNodesOfTribeInByte, _ := proto.Marshal(listOfNodesOfTribe)

							errorFromPut := PutValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_Epoch_1_Oracle_Group_"+strconv.Itoa(i)), listOfNodesOfTribeInByte)

							// store on block
							nodeBlock.OracleList[strconv.Itoa(i)] = listOfNodesOfTribe

							if errorFromPut != nil {
								fmt.Println(errorFromPut)
							}
							var wg sync.WaitGroup
							request := api.Request{
								Action: "_Epoch_1_Oracle_Group_" + strconv.Itoa(i),
								Param:  strconv.Itoa(i),
								Data:   listOfNodesOfTribeInByte,
							}
							for j := 0; j <= len(contactsFromOwnDb.Contact)-1; j++ {

								if contactsFromOwnDb.Contact[j].Address != selfContact.Address {
									wg.Add(1)

									go BootstrapSync(selfContact, contactsFromOwnDb.Contact[j], request, nil, &wg, node)
								}

							}

							wg.Wait()
							// contactsFromOwnDb = api.Contacts{}

						}
					}

					// BLS FOR ORACLE START......

					thresholdOfOracles, _ := strconv.Atoi(os.Getenv("THRESHOLD_OF_ORACLES"))
					fmt.Println(thresholdOfOracles, lengthOfGroup, "ORACLE")

					go func() {
						for index := 0; index <= numberOfOracleGroups; index++ {

							// Generate key shares.
							// params := GenParamsTypeF(256)
							// pairing := GenPairing(params)
							// system, _ := GenSystem(pairing)

							fmt.Println("thresholdOfOracles, lengthOfGroup, system", thresholdOfOracles, lengthOfGroup, system)

							// Genetate Keys
							groupKey, memberKeys, groupSecret, memberSecrets, _ := GenKeyShares(thresholdOfOracles, lengthOfGroup, system)
							groupSecretToBytes := system.PrivateKeyToBytes(groupSecret)
							groupPublicToBytes := system.PublicKeyToBytes(groupKey)

							// store in block

							// nodeBlock.OracleList[strconv.Itoa(i)] = listOfNodesOfTribe

							if Debug == true {
								fmt.Println(groupKey, memberKeys, groupSecret, memberSecrets)
							}

							// oracleGroup := GetValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_Epoch_1_Oracle_Group_"+strconv.Itoa(index)))

							oracleGroup := nodeBlock.OracleList[strconv.Itoa(index)]

							fmt.Println("oracleGroup,Contact", len(oracleGroup.Contact))

							// Store oracle and oracle group's secret key among oracle

							var wgInner sync.WaitGroup

							for j := 0; j < len(oracleGroup.Contact); j++ {

								wgInner.Add(2)
								memberSecretsToBytes := system.PrivateKeyToBytes(memberSecrets[j])

								// Store oracle group member's secret key
								request1 := api.Request{
									Action: "bls_oracle_secret",
									Data:   memberSecretsToBytes,
								}

								go BootstrapSync(selfContact, oracleGroup.Contact[j], request1, nil, &wgInner, node)

								// Store oracle group's secret keys
								request2 := api.Request{
									Action: "bls_oracle_group_secret",
									Data:   groupSecretToBytes,
								}

								go BootstrapSync(selfContact, oracleGroup.Contact[j], request2, nil, &wgInner, node)

							}

							wgInner.Wait()

							var wgInnerSecond sync.WaitGroup

							// Store oracle and oracle group's public key among network
							for k := 0; k < len(contactsFromOwnDb.Contact); k++ {
								wgInnerSecond.Add(1)

								// memberPublicToBytes := system.PublicKeyToBytes(memberKeys[j])

								// // Store oracle public key across network
								// request1 := api.Request{
								// 	Action: "bls_oracle_public",
								// 	Data:   memberPublicToBytes,
								// }

								// go BootstrapSync(selfContact, contactsFromOwnDb.Contact[k], request1, nil, nil, node)

								nodeBlock.OraclesPublicKey[strconv.Itoa(index)] = groupPublicToBytes
								// Store oracle group public key across network
								request2 := api.Request{
									Action: "bls_oracle_group_public",
									Data:   groupPublicToBytes,
									Param:  strconv.Itoa(index),
								}

								go BootstrapSync(selfContact, contactsFromOwnDb.Contact[k], request2, nil, &wgInnerSecond, node)

							}

							wgInnerSecond.Wait()

							// Free Oracle  Keys
							groupKey.Free()
							groupSecret.Free()

							for i := 0; i < lengthOfGroup; i++ {
								memberKeys[i].Free()
								memberSecrets[i].Free()
							}

						}

					}()

					// BLS FOR ORACLE END........

					ticker.Stop()

					startOrganizingTribe(*listOfOracles, contactsFromOwnDb, selfContact, node, numberOfOracleGroups, system, nodeBlock)
					// pairing.Free()
					// params.Free()
					// quit <- true
				}
				// do stuff
			case <-quit:
				// if Debug == true {
				// }

				ticker.Stop()
				return
			}
		}
	}()
}

// This function organize nodes into tribes
func startOrganizingTribe(listOfOracles, contactsFromOwnDb api.Contacts, selfContact api.Contact, node *UnityNode, numberOfOracleGroups int, system System, nodeBlock *NodeBlock) {

	// err := godotenv.Load()
	// if err != nil {
	// 	fmt.Println("Error loading .env file")
	// }

	Tribes := os.Getenv("NUMBER_OF_TRIBES")
	numberOfTribes, _ := strconv.Atoi(Tribes)

	ticker := time.NewTicker(50 * time.Millisecond)
	quit := make(chan bool)

	// // Generate DKG for each Group of Oracle
	// for i := 0; i <= numberOfOracleGroups; i++ {
	// 	// if i == 0 {
	// 	secretKey := ""
	// 	listOfNodes := GetValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_Epoch_1_Oracle_Group_"+strconv.Itoa(i)))
	// 	if Debug == true {
	// 		fmt.Println("-------------"+strconv.Itoa(i)+"Group -------------\n", len(listOfNodes.Contact))
	// 	}

	// 	for j := 0; j <= len(listOfNodes.Contact)-1; j++ {
	// 		if j != 0 {
	// 			key := hex.EncodeToString(node.NodeID[:]) + "_dkg_Secret_Epoch_1_Oracle_Group_" + strconv.Itoa(i)
	// 			getSecretKey, _ := node.ValuesDB.Get([]byte(key), nil)

	// 			request := api.Request{
	// 				Action: "_dkg_Secret_Epoch_1_Oracle_Group_" + strconv.Itoa(i),
	// 				Data:   getSecretKey,
	// 				IsDKG:  true,
	// 			}

	// 			secretKeyData := BootstrapSync(*listOfNodes.Contact[j], listOfNodes.Contact[0], request, nil, nil, node)
	// 			json.Unmarshal(secretKeyData.Data, &secretKey)
	// 		}
	// 	}

	// 	var wg sync.WaitGroup
	// 	request := api.Request{
	// 		Action: "_dkg_Secret_Epoch_1_Oracle_Group_" + strconv.Itoa(i),
	// 		Data:   []byte(secretKey),
	// 	}
	// 	// store group's secret key among group
	// 	for k := 0; k <= len(listOfNodes.Contact)-1; k++ {

	// 		wg.Add(1)

	// 		go BootstrapSync(selfContact, listOfNodes.Contact[k], request, nil, &wg, node)

	// 	}
	// 	wg.Wait()

	// 	groupPublicKey := GenerateSharedPublicKey(secretKey)
	// 	if Debug == true {
	// 		fmt.Println(groupPublicKey, "groupPublicKey", i)
	// 	}

	// 	request = api.Request{
	// 		Action: "_dkg_Public_Epoch_1_Oracle_Group_" + strconv.Itoa(i),
	// 		Data:   []byte(groupPublicKey),
	// 	}
	// 	// store group's public key across the network
	// 	for k := 0; k <= len(contactsFromOwnDb.Contact)-1; k++ {

	// 		wg.Add(1)

	// 		go BootstrapSync(selfContact, contactsFromOwnDb.Contact[k], request, nil, &wg, node)
	// 	}

	// 	wg.Wait()

	// }

	allNodes := &api.Contacts{}
	allNodes.Contact = append(nodeBlock.AllNodesInNetwork[:0:0], nodeBlock.AllNodesInNetwork...)
	fmt.Println(len(allNodes.Contact), "before", len(listOfOracles.Contact))

	for i := 0; i <= len(listOfOracles.Contact)-1; i++ {
		for j := 0; j <= len(allNodes.Contact)-1; j++ {
			if Equal(allNodes.Contact[j].ID, listOfOracles.Contact[i].ID) {
				if Debug == true {
					fmt.Println(allNodes.Contact[j].Address, listOfOracles.Contact[i].Address, i)
				}
				allNodes.Contact = append(allNodes.Contact[:j], allNodes.Contact[j+1:]...)
			}
		}
	}
	// if Debug == true {
	fmt.Println(len(allNodes.Contact), "After", len(listOfOracles.Contact))
	// }
	// go func() {
	for {
		select {
		case <-ticker.C:

			groupOfRandomNumber := len(allNodes.Contact) % (numberOfTribes + 1)
			if Debug == true {
				fmt.Println(groupOfRandomNumber, "groupOfRandomNumber")
			}

			if len(allNodes.Contact) > 0 {
				allNodes.Contact = allNodes.Contact[:len(allNodes.Contact)-groupOfRandomNumber]
			}

			lengthOfGroup := len(allNodes.Contact) / (numberOfTribes + 1)
			if Debug == true {
				fmt.Println(lengthOfGroup, "lengthOfGroup")
			}

			for i := 0; i <= numberOfTribes; i++ {
				listOfNodesOfTribe := &api.Contacts{}
				for k := 0; k < lengthOfGroup; k++ {
					randNumber := 0
					minNew := 0
					maxNew := len(allNodes.Contact)
					if maxNew > 0 {
						randNumber = rand.Intn(maxNew-minNew) + minNew
					}
					if len(allNodes.Contact) != 0 {
						if Debug == true {
							fmt.Println(randNumber, "randNumber")
						}
						selectedNode := allNodes.Contact[randNumber]

						allNodes.Contact = append(allNodes.Contact[:randNumber], allNodes.Contact[randNumber+1:]...)

						// listOfNodesOfTribe := GetValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_Tribe_Group_"+strconv.Itoa(i)))
						listOfNodesOfTribe.Contact = append(listOfNodesOfTribe.Contact, selectedNode)
						maxNew--

					}
					// if len(allNodes) == 1 {
					// 	electionProccessForLeaders(listOfOracles, contactsFromOwnDb, selfContact, node, numberOfTribes)

					// 	quit <- true
					// }
				}

				listOfNodesOfTribeInByte, _ := proto.Marshal(listOfNodesOfTribe)

				// errorFromPut := PutValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_Tribe_Group_"+strconv.Itoa(i)), listOfNodesOfTribeInByte)
				// if errorFromPut != nil {
				// 	fmt.Println(errorFromPut)
				// }

				var wg sync.WaitGroup
				request := api.Request{
					Action: "_Tribe_Group_" + strconv.Itoa(i),
					Data:   listOfNodesOfTribeInByte,
					Param:  strconv.Itoa(i),
				}

				for j := 0; j <= len(contactsFromOwnDb.Contact)-1; j++ {

					wg.Add(1)

					go BootstrapSync(selfContact, contactsFromOwnDb.Contact[j], request, nil, &wg, node)

				}
				wg.Wait()

				if i == numberOfTribes {
					ticker.Stop()

					electionProccessForLeaders(listOfOracles, contactsFromOwnDb, selfContact, node, numberOfTribes, system, nodeBlock)

					// contactsFromOwnDb = api.Contacts{}
					// listOfNodesOfTribeInByte = nil
					quit <- true
				}
			}

		case <-quit:
			// if Debug == true {
			// }
			ticker.Stop()
			return
		}
	}
	// }()
}

// Handles election process of leaders
func electionProccessForLeaders(listOfOracles, latestContacts api.Contacts, selfContact api.Contact, node *UnityNode, numberOfTribes int, system System, nodeBlock *NodeBlock) {
	// err := godotenv.Load()
	// if err != nil {
	// 	fmt.Println("Error loading .env file")
	// }

	thresholdOfLeaders, _ := strconv.Atoi(os.Getenv("THRESHOLD_OF_LEADERS"))

	if Debug == true {
		fmt.Println("")
		fmt.Println("-------------------Starting Election Process Of Leaders------------------")
	}

	selectedLeadersForGroup1 := &api.Contacts{}
	selectedLeadersForGroup2 := &api.Contacts{}
	//	var secretShares string
	for i := 0; i <= numberOfTribes; i++ {
		listOfNodesOfTribe := &api.Contacts{}
		listOfNodesOfTribe.Contact = nodeBlock.TribeList[strconv.Itoa(i)].Contact

		fmt.Println(len(listOfNodesOfTribe.Contact), "listOfNodesOfTribe")

		min := 0
		max := len(listOfNodesOfTribe.Contact)

		randNumber1 := rand.Intn(max-min) + min
		randNumber2 := rand.Intn(max-min) + min

		if randNumber1 == randNumber2 {
			randNumber2 = rand.Intn(max-min) + min

		}

		fmt.Println("Random numbers", randNumber1, randNumber2)

		if Debug == true {
			fmt.Println("")
			fmt.Println("---------------Selected Leader", i, "----------------")
			fmt.Println(listOfNodesOfTribe.Contact[randNumber1])
			fmt.Println("")
		}

		selectedLeader1 := listOfNodesOfTribe.Contact[randNumber1]
		selectedLeader1.IsLeader = true
		selectedLeadersForGroup1.Contact = append(selectedLeadersForGroup1.Contact, selectedLeader1)

		selectedLeader2 := listOfNodesOfTribe.Contact[randNumber2]
		selectedLeader2.IsLeader = true
		selectedLeadersForGroup2.Contact = append(selectedLeadersForGroup2.Contact, selectedLeader2)
		// selectedLeadersData, _ := proto.Marshal(*selectedLeader)
		// t := time.Now()
		// t.Second()

		for j := 0; j <= len(listOfNodesOfTribe.Contact)-1; j++ {
			if (listOfNodesOfTribe.Contact[j].Address == selectedLeader1.Address) || (listOfNodesOfTribe.Contact[j].Address == selectedLeader2.Address) {
				listOfNodesOfTribe.Contact = append(listOfNodesOfTribe.Contact[:j], listOfNodesOfTribe.Contact[j+1:]...)
			}
		}

		// nodeBlock.TribeList[strconv.Itoa(i)] = listOfNodesOfTribe

		listOfNodesOfTribeInByte, _ := proto.Marshal(listOfNodesOfTribe)

		var wg sync.WaitGroup

		request := api.Request{
			Action: "_Tribe_Group_" + strconv.Itoa(i),
			Data:   listOfNodesOfTribeInByte,
			Param:  strconv.Itoa(i),
		}
		for m := 0; m <= len(latestContacts.Contact)-1; m++ {

			wg.Add(1)

			go BootstrapSync(selfContact, latestContacts.Contact[m], request, nil, &wg, node)

		}

		wg.Wait()
		listOfNodesOfTribeInByte = nil

	}

	for i := 1; i < 3; i++ {
		selectedLeadersForGroup := &api.Contacts{}

		if i == 1 {
			selectedLeadersForGroup = selectedLeadersForGroup1
		} else {
			selectedLeadersForGroup = selectedLeadersForGroup2
		}
		listOfSelectedLeaders, _ := proto.Marshal(selectedLeadersForGroup)

		nodeBlock.LeaderList[strconv.Itoa(i)] = selectedLeadersForGroup

		// update node type on nodeBlock

		var wgForNodeType sync.WaitGroup
		requestForNodeType := api.Request{
			Action: "NodeType",
			Param:  "IsLeader",
		}
		for j := 0; j <= len(selectedLeadersForGroup.Contact)-1; j++ {
			if selectedLeadersForGroup.Contact[j].Address != selfContact.Address {
				wgForNodeType.Add(1)

				go BootstrapSync(selfContact, selectedLeadersForGroup.Contact[j], requestForNodeType, nil, &wgForNodeType, node)
			}

		}
		wgForNodeType.Wait()

		var wgForStoreLeaderList sync.WaitGroup

		request1 := api.Request{
			Action: "Leaders_Epoch_1_Group_" + strconv.Itoa(i),
			Data:   listOfSelectedLeaders,
			Param:  strconv.Itoa(i),
		}
		fmt.Println(len(latestContacts.Contact), "latestContacts")
		for j := 0; j <= len(latestContacts.Contact)-1; j++ {

			// if latestContacts.Contact[j].Address != selfContact.Address {
			wgForStoreLeaderList.Add(1)

			go BootstrapSync(selfContact, latestContacts.Contact[j], request1, nil, &wgForStoreLeaderList, node)
			// }
		}

		wgForStoreLeaderList.Wait()

		fmt.Println(len(selectedLeadersForGroup.Contact), "listOfSelectedLeaders")

	}
	// wg.Wait()

	// BLS FOR LEADER START........

	go func() {
		for index := 1; index < 3; index++ {
			leaderGroup := nodeBlock.LeaderList[strconv.Itoa(index)]
			fmt.Println(thresholdOfLeaders, len(leaderGroup.Contact), "LEADER")

			// Genetate Keys
			groupKey, memberKeys, groupSecret, memberSecrets, _ := GenKeyShares(thresholdOfLeaders, len(leaderGroup.Contact), system)
			groupPublicToByte := system.PublicKeyToBytes(groupKey)
			groupSecretToBytes := system.PrivateKeyToBytes(groupSecret)

			if Debug == true {
				fmt.Println(groupKey, memberKeys, groupSecret, memberSecrets)
			}

			for j := 0; j < len(leaderGroup.Contact); j++ {

				// Store leader group member's secret key
				memberSecretToByte := system.PrivateKeyToBytes(memberSecrets[j])
				request1 := api.Request{
					Action: "bls_leader_secret",
					Data:   memberSecretToByte,
				}
				// Store Leader group's secret keys
				go BootstrapSync(selfContact, leaderGroup.Contact[j], request1, nil, nil, node)

				request2 := api.Request{
					Action: "bls_leader_group_secret",
					Data:   groupSecretToBytes,
				}

				go BootstrapSync(selfContact, leaderGroup.Contact[j], request2, nil, nil, node)

				nodeBlock.LeadersPublicKey[strconv.Itoa(index)] = groupPublicToByte

				// Store leader and leader group's public key among network
				for k := 0; k < len(latestContacts.Contact); k++ {
					// Store oracle public key across network
					memberPublicToByte := system.PublicKeyToBytes(memberKeys[j])
					request1 := api.Request{
						Action: "bls_leader_public",
						Data:   memberPublicToByte,
					}

					go BootstrapSync(selfContact, latestContacts.Contact[k], request1, nil, nil, node)

					// Store oracle group public key across network
					request2 := api.Request{
						Action: "bls_leader_group_public",
						Data:   groupPublicToByte,
						Param:  strconv.Itoa(index),
					}

					go BootstrapSync(selfContact, latestContacts.Contact[k], request2, nil, nil, node)

				}

			}

			// Store leader and leader group's public key among network
			//groupPublicToByte := system.PublicKeyToBytes(groupKey)

			// for k := 0; k < len(latestContacts.Contact); k++ {
			// 	//	memberPublicToByte := system.PublicKeyToBytes(memberKeys[j])
			// 	// Store oracle public key across network
			// 	// request1 := api.Request{
			// 	// 	Action: "bls_leader_public",
			// 	// 	Data:   memberPublicToByte,
			// 	// }

			// 	//	go BootstrapSync(selfContact, latestContacts.Contact[k], request1, nil, nil, node)

			// 	// Store oracle group public key across network
			// 	request2 := api.Request{
			// 		Action: "bls_leader_group_public",
			// 		Data:   groupPublicToByte,
			// 		Param:  strconv.Itoa(index),
			// 	}

			// 	BootstrapSync(selfContact, latestContacts.Contact[k], request2, nil, nil, node)

			// }

			//************

			// paraBytes, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(node.NodeID[:])+"bls_params"), nil)
			// paramss, _ := ParamsFromBytes(paraBytes)
			// pairings := GenPairing(paramss)
			// sysBytes, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(node.NodeID[:])+"bls_system"), nil)
			// system2, _ := SystemFromBytes(pairings, sysBytes)

			// message := "This is a message."

			// listOfLeadersData := GetValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_Leaders_Epoch_1_Group_2"))

			// keyInByte, _ := json.Marshal("bls_leader_secret")
			// memberSecretsData := []PrivateKey{}

			// for i := 0; i < len(listOfLeadersData.Contact); i++ {

			// 	request := api.Request{
			// 		Action: "retrive_bls_secret",
			// 		Data:   keyInByte,
			// 	}

			// 	response := BootstrapSync(selfContact, listOfLeadersData.Contact[i], request, nil, nil, node)
			// 	LeaderPrivateKey := system2.PrivateFromBytes(response.Data)

			// 	memberSecretsData = append(memberSecretsData, LeaderPrivateKey)
			// }

			// keyInByte2, _ := json.Marshal("bls_leader_group_2_public")

			// request3 := api.Request{
			// 	Action: "retrive_bls_secret",
			// 	Data:   keyInByte2,
			// }

			// response3 := BootstrapSync(selfContact, listOfLeadersData.Contact[0], request3, nil, nil, node)
			// LeaderGroupPublicKeyKey, _ := system.PublicFromBytes(response3.Data)

			// n := 5
			// t := 2

			// memberIds := rand.Perm(n)[:t]

			// hash := sha256.Sum256([]byte(message))
			// shares := make([]Signature, t)

			// for l := 0; l < t; l++ {
			// 	shares[l] = Sign(hash, memberSecretsData[memberIds[l]])
			// }

			// signature, _ := Threshold(shares, memberIds, system2)
			// LeaderGroupPublicKeyKey := system2.PrivateKeyToBytes(groupSecret)
			// ori, _ := system.PrivateFromBytes2(LeaderGroupPublicKeyKey)

			// ori, _ := system2.PublicFromBytes(groupPublicToByte)
			// if !Verify(signature, hash, ori) {
			// 	fmt.Println("Failed to verify signature__________________________________________________________________________")
			// } else {
			// 	fmt.Println("ohh verified")
			// }
			//************

			// Free Leaders Keys
			groupKey.Free()
			groupSecret.Free()

			for i := 0; i < len(leaderGroup.Contact); i++ {
				memberKeys[i].Free()
				memberSecrets[i].Free()
			}

		}
	}()

	// BLS FOR LEADER END........

	groupOfRSIPFromTribes(node, selfContact, numberOfTribes, latestContacts, system, nodeBlock)

	TribalMappingForDecimal(node, selfContact, numberOfTribes, latestContacts, nodeBlock)

	system.Free()

	// latestContacts = api.Contacts{}

	t2 := time.Now()
	t2.Second()
	if Debug == true {
		fmt.Println(t2, "---------------------------------------end time")
	}
}

// Making decimal list of RSIP groups
func TribalMappingForDecimal(node *UnityNode, selfContact api.Contact, numberOfTribes int, latestContacts api.Contacts, nodeBlock *NodeBlock) {

	if Debug == true {
		fmt.Println("------------------------------------Tribal Mapping--------------------------------------")
	}

	for j := 0; j <= numberOfTribes; j++ {
		if Debug == true {
			fmt.Println("\n------------------------------------Tribe->", j, "--------------------------------------")
		}

		numberOfRSIPGroup := os.Getenv("NUMBER_OF_RSIP_GROUP")
		numberOfRSIPGroupInInt, _ := strconv.Atoi(numberOfRSIPGroup)

		secKeysOfAllRSIP := api.RSIPDecimals{}

		for k := 0; k <= numberOfRSIPGroupInInt; k++ {

			GroupNodes := &api.Contacts{}
			// contacts, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(selfContact.ID[:])+"-Tribe_"+strconv.Itoa(j)+"_RSIP_GROUP_"+strconv.Itoa(k)), nil)
			for index := 0; index < len(nodeBlock.RSIPList[strconv.Itoa(j)]); index++ {
				if nodeBlock.RSIPList[strconv.Itoa(j)][index].Group == strconv.Itoa(k) {
					for l := 0; l < len(nodeBlock.RSIPList[strconv.Itoa(j)][index].Contacts.Contact); l++ {
						GroupNodes.Contact = append(GroupNodes.Contact, nodeBlock.RSIPList[strconv.Itoa(j)][index].Contacts.Contact[l])
					}
				}
			}

			// proto.Unmarshal(contacts, GroupNodes)

			key := "bls_RSIP_group_secret"
			keyInByte, _ := json.Marshal(key)

			request := api.Request{
				Action: "retrive_bls_secret",
				Data:   keyInByte,
			}

			fmt.Println("GroupNodes.Contact", len(GroupNodes.Contact))
			res := BootstrapSync(selfContact, GroupNodes.Contact[0], request, nil, nil, node)
			data := binary.BigEndian.Uint32(res.Data)

			decKeyWithGroup := api.RSIPDecimal{}
			decKeyWithGroup.Group = int32(k)
			decKeyWithGroup.Decimal = int32(data)

			if Debug == true {
				fmt.Println("Sec-Decimal-DKG-STRUCT-WITH-GROUP[", j, "]-----------------", decKeyWithGroup)
			}

			secKeysOfAllRSIP.RSIPDecimal = append(secKeysOfAllRSIP.RSIPDecimal, &decKeyWithGroup)

		}

		// Store secKeysOfAllRSIP across network
		fmt.Println("LEN-OF-CONTACTS-----------------", len(latestContacts.Contact))

		sort.Slice(secKeysOfAllRSIP.RSIPDecimal, func(i, j int) bool {
			return secKeysOfAllRSIP.RSIPDecimal[i].Decimal > secKeysOfAllRSIP.RSIPDecimal[j].Decimal
		})

		fmt.Println(secKeysOfAllRSIP)

		var wg sync.WaitGroup

		decKeysInByte, _ := proto.Marshal(&secKeysOfAllRSIP)
		request := api.Request{
			Action: "_Tribe_" + strconv.Itoa(j) + "_Decimal_DKG",
			Data:   decKeysInByte,
		}
		for l := 0; l < len(latestContacts.Contact); l++ {

			wg.Add(1)

			go BootstrapSync(selfContact, latestContacts.Contact[l], request, nil, &wg, node)

		}

		wg.Wait()

	}
	// latestContacts = api.Contacts{}
	EndTime := time.Since(SetupTime)
	fmt.Println("END-NETWORK-SETUP", EndTime)

	// storing membership block

	// currentBlock := GetValue(node, []byte("BLOCK_NUMBER"))

	blockDataInByte, _ := json.Marshal(nodeBlock)

	err := PutValuesById(node, []byte(string(node.NodeID[:])+"_Block_"+strconv.Itoa(node.CurrentElectionBlock)), blockDataInByte)

	var wg sync.WaitGroup

	// decKeysInByte, _ := proto.Marshal(&secKeysOfAllRSIP)
	request := api.Request{
		Action: "STORE_BLOCK_IN_DB",
		Index:  int32(node.CurrentElectionBlock),
	}
	for l := 0; l < len(latestContacts.Contact); l++ {

		if latestContacts.Contact[l].Address != selfContact.Address {
			wg.Add(1)

			go BootstrapSync(selfContact, latestContacts.Contact[l], request, nil, &wg, node)
		}

	}

	wg.Wait()

	fmt.Println("Block is stored in levelDB", err)

	// time.Sleep(50 * time.Second)

	// newTicker := time.NewTicker(20 * time.Second)
	// for {
	// 	select {
	// 	case <-newTicker.C:
	// 		syncBatchDatailsToGlobalDHT(selfContact, node)
	// 	}
	// }

	// fmt.Println(" ")
	// fmt.Println(" ")
	// fmt.Println(" ")
	// fmt.Println("Making TCP Connections Between Nodes.....")

	// request := api.Request{
	// 	Action: "_TCP_CONNECTIONS",
	// }
	// for l := 0; l < len(latestContacts.Contact); l++ {

	// 	wg.Add(1)

	// 	go BootstrapSync(selfContact, latestContacts.Contact[l], request, nil, &wg, node)

	// }

	// wg.Wait()

	// fmt.Println("FINISHED CREATING OF TCP CONNECTIONS", time.Since(SetupTime))

}

// Organize RSIP groups from tribe
func groupOfRSIPFromTribes(node *UnityNode, selfContact api.Contact, numberOfTribes int, latestContacts api.Contacts, system System, nodeBlock *NodeBlock) {
	numberOfRSIPGroup := os.Getenv("NUMBER_OF_RSIP_GROUP")
	numberOfRSIPGroupInInt, _ := strconv.Atoi(numberOfRSIPGroup)
	numberOfAlphaOracles := os.Getenv("NUMBER_OF_ALPHA")
	numberOfAlphaNodes, _ := strconv.Atoi(numberOfAlphaOracles)

	numberOfRSIP := 0

	var wg sync.WaitGroup

	for j := 0; j <= numberOfTribes; j++ {

		// get list of oracles tribe
		// listOfOracles := GetValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_Tribe_Group_"+strconv.Itoa(j)))
		listOfOracles := nodeBlock.TribeList[strconv.Itoa(j)]

		// min := 0
		// max := len(listOfOracles) - 1

		// // group nodes into RSIP
		// for i := 0; i <= numberOfRSIPGroupInInt; i++ {

		// 	randNumber := rand.Intn(max-min) + min
		// 	selectedNode := listOfOracles[randNumber]

		// 	listOfOracles = append(listOfOracles[:randNumber], listOfOracles[randNumber+1:]...)

		// 	listOfRSIPNodes := GetValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_Tribe_"+strconv.Itoa(j)+"_RSIP_GROUP_"+strconv.Itoa(i)))
		// 	listOfRSIPNodes = append(listOfRSIPNodes, selectedNode)
		// 	listOfOraclesInByte, _ := proto.Marshal(listOfRSIPNodes)

		// 	errorFromPut := PutValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_Tribe_"+strconv.Itoa(j)+"_RSIP_GROUP_"+strconv.Itoa(i)), listOfOraclesInByte)
		// 	if errorFromPut != nil {
		// 		fmt.Println(errorFromPut)
		// 	}
		// 	max--
		// }

		if Debug == true {
			fmt.Println("listOfOracles", len(listOfOracles.Contact), numberOfRSIPGroupInInt+1)
		}

		groupOfRandomNumber := len(listOfOracles.Contact) % (numberOfRSIPGroupInInt + 1)
		if Debug == true {
			fmt.Println(groupOfRandomNumber, "groupOfRandomNumber")
		}

		if len(listOfOracles.Contact) > 0 {
			listOfOracles.Contact = listOfOracles.Contact[:len(listOfOracles.Contact)-groupOfRandomNumber]
		}

		lengthOfGroup := len(listOfOracles.Contact) / (numberOfRSIPGroupInInt + 1)
		if Debug == true {
			fmt.Println(lengthOfGroup, "lengthOfGroup")
		}

		for len(listOfOracles.Contact) > 0 {
			for i := 0; i <= numberOfRSIPGroupInInt; i++ {
				listOfRSIPNodes := &api.Contacts{}
				numberOfAlpha := numberOfAlphaNodes

				for k := 0; k < lengthOfGroup; k++ {
					randNumber := 0
					min := 0
					max := len(listOfOracles.Contact)
					numberOfAlpha--
					if max > 0 {
						randNumber = rand.Intn(max-min) + min
					}
					if len(listOfOracles.Contact) != 0 {
						selectedNode := listOfOracles.Contact[randNumber]

						if numberOfAlpha > 0 {
							selectedNode.Category = "alpha"
						} else {
							selectedNode.Category = "beta"
						}

						listOfOracles.Contact = append(listOfOracles.Contact[:randNumber], listOfOracles.Contact[randNumber+1:]...)

						// listOfRSIPNodes := GetValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"-Tribe_"+strconv.Itoa(j)+"_RSIP_GROUP_"+strconv.Itoa(i)))

						listOfRSIPNodes.Contact = append(listOfRSIPNodes.Contact, selectedNode)
						max--

					}
				}
				if Debug == true {
					fmt.Println(len(listOfRSIPNodes.Contact), "Group RSIPS")
				}
				listOfOraclesInByte, _ := proto.Marshal(listOfRSIPNodes)

				var wgForNodeType sync.WaitGroup
				requestForNodeType := api.Request{
					Action:      "NodeType",
					Param:       "IsRSIP",
					RSIPGroup:   int32(i),
					TribeNumber: int32(j),
				}
				for j := 0; j <= len(listOfRSIPNodes.Contact)-1; j++ {

					if listOfRSIPNodes.Contact[j].Address != selfContact.Address {
						wgForNodeType.Add(1)

						go BootstrapSync(selfContact, listOfRSIPNodes.Contact[j], requestForNodeType, nil, &wgForNodeType, node)
					}

				}
				wgForNodeType.Wait()

				errorFromPut := PutValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"-Tribe_"+strconv.Itoa(j)+"_RSIP_GROUP_"+strconv.Itoa(i)), listOfOraclesInByte)
				if errorFromPut != nil {
					fmt.Println(errorFromPut)
				}

				nodeBlock.RSIPList[strconv.Itoa(j)] = append(nodeBlock.RSIPList[strconv.Itoa(j)], GroupList{
					Group:    strconv.Itoa(int(i)),
					Contacts: *listOfRSIPNodes,
				})

				request := api.Request{
					Action:      "-Tribe_" + strconv.Itoa(j) + "_RSIP_GROUP_" + strconv.Itoa(i),
					Data:        listOfOraclesInByte,
					RSIPGroup:   int32(i),
					TribeNumber: int32(j),
				}
				for k := 0; k <= len(latestContacts.Contact)-1; k++ {

					if latestContacts.Contact[k].Address != selfContact.Address {
						wg.Add(1)

						go BootstrapSync(selfContact, latestContacts.Contact[k], request, nil, &wg, node)
					}

				}

				wg.Wait()
				listOfRSIPNodes = nil
				listOfOraclesInByte = nil

			}
		}
		listOfOracles = &api.Contacts{}
	}
	// for j := 0; j <= numberOfTribes; j++ {

	// 	// Generate DKG for each Group
	// 	for i := 0; i <= numberOfRSIPGroupInInt; i++ {
	// 		// if i == 0 {
	// 		secretKey := ""
	// 		listOfRSIPNodes := GetValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"-Tribe_"+strconv.Itoa(j)+"_RSIP_GROUP_"+strconv.Itoa(i)))
	// 		for m := 0; m <= len(listOfRSIPNodes.Contact)-1; m++ {
	// 			if m != 0 {

	// 				keyInByte, _ := json.Marshal("_Tribe_" + strconv.Itoa(j) + "_dkg_Secret_RSIP_Group_" + strconv.Itoa(i))

	// 				request := api.Request{
	// 					Action: "Retrieve-RSIP-Group-Seckey",
	// 					Data:   keyInByte,
	// 					Param:  "secret-key",
	// 				}

	// 				secretKeyData2 := BootstrapSync(selfContact, listOfRSIPNodes.Contact[0], request, nil, nil, node)

	// 				request2 := api.Request{
	// 					Action: "_Tribe_" + strconv.Itoa(j) + "_dkg_Secret_RSIP_Group_" + strconv.Itoa(i),
	// 					Data:   secretKeyData2.Data,
	// 					IsDKG:  true,
	// 				}

	// 				secretKeyData := BootstrapSync(*listOfRSIPNodes.Contact[m], listOfRSIPNodes.Contact[0], request2, nil, nil, node)
	// 				json.Unmarshal(secretKeyData.Data, &secretKey)

	// 			}
	// 		}

	// 		request := api.Request{
	// 			Action: "_Tribe_" + strconv.Itoa(j) + "_dkg_Secret_RSIP_Group_" + strconv.Itoa(i),
	// 			Data:   []byte(secretKey),
	// 		}
	// 		// store group's secret key among group
	// 		for k := 0; k <= len(listOfRSIPNodes.Contact)-1; k++ {

	// 			wg.Add(1)

	// 			go BootstrapSync(selfContact, listOfRSIPNodes.Contact[k], request, nil, &wg, node)
	// 		}

	// 		wg.Wait()

	// 		groupPublicKey := GenerateSharedPublicKey(secretKey)
	// 		if Debug == true {
	// 			fmt.Println(groupPublicKey, "groupPublicKey", i)
	// 		}

	// 		request = api.Request{
	// 			Action: "_Tribe_" + strconv.Itoa(j) + "_dkg_Public_RSIP_Group_" + strconv.Itoa(i),
	// 			Data:   []byte(groupPublicKey),
	// 		}
	// 		// store group's public key across the network
	// 		for k := 0; k <= len(latestContacts.Contact)-1; k++ {

	// 			wg.Add(1)

	// 			go BootstrapSync(selfContact, latestContacts.Contact[k], request, nil, &wg, node)
	// 		}

	// 		wg.Wait()

	// 	}

	// }

	// BLS FOR RSIP START........

	thresholdofRSIP, _ := strconv.Atoi(os.Getenv("THRESHOLD_OF_RSIPS"))
	fmt.Println(thresholdofRSIP, numberOfRSIP, "RSIP", "number of Tribes", numberOfTribes, "number of RSIP group", numberOfRSIPGroupInInt)
	// go func() {
	for index := 0; index <= numberOfTribes; index++ {

		for j := 0; j <= numberOfRSIPGroupInInt; j++ {

			listOfRSIPNodes := &api.Contacts{}
			// listOfRSIPNodes := GetValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"-Tribe_"+strconv.Itoa(index)+"_RSIP_GROUP_"+strconv.Itoa(j)))

			for l := 0; l < len(nodeBlock.RSIPList[strconv.Itoa(index)]); l++ {
				if nodeBlock.RSIPList[strconv.Itoa(index)][l].Group == strconv.Itoa(j) {
					for m := 0; m < len(nodeBlock.RSIPList[strconv.Itoa(index)][l].Contacts.Contact); m++ {
						listOfRSIPNodes.Contact = append(listOfRSIPNodes.Contact, nodeBlock.RSIPList[strconv.Itoa(index)][j].Contacts.Contact[m])
					}
				}
			}

			// Genetate Keys
			groupKey, memberKeys, groupSecret, memberSecrets, _ := GenKeyShares(thresholdofRSIP, len(listOfRSIPNodes.Contact), system)

			// groupKey, memberKeys, groupSecret, memberSecrets, _ := GenKeyShares(len(listOfRSIPNodes.Contact), len(listOfRSIPNodes.Contact), system)

			groupSecretToByte := system.PrivateKeyToBytes(groupSecret)
			groupPublicToByte := system.PublicKeyToBytes(groupKey)

			if Debug == true {
				fmt.Println(groupKey, memberKeys, groupSecret, memberSecrets)
			}
			fmt.Println(len(listOfRSIPNodes.Contact), "listOfRSIPNodes")

			RSIPGroupKey := GroupsPublicKeys{
				Group:     strconv.Itoa(j),
				PublicKey: groupPublicToByte,
			}

			nodeBlock.RSIPPublicKey[strconv.Itoa(index)] = append(nodeBlock.RSIPPublicKey[strconv.Itoa(index)], RSIPGroupKey)

			start2 := time.Now()
			for k := 0; k < len(listOfRSIPNodes.Contact); k++ {

				wg.Add(2)
				memberSecretsToByte := system.PrivateKeyToBytes(memberSecrets[k])

				// Store oracle group member's secret key
				request1 := api.Request{
					Action: "bls_RSIP_secret",
					Data:   memberSecretsToByte,
				}

				go BootstrapSync(selfContact, listOfRSIPNodes.Contact[k], request1, nil, &wg, node)

				// Store oracle group's secret keys
				request2 := api.Request{
					Action: "bls_RSIP_group_secret",
					Data:   groupSecretToByte,
				}

				go BootstrapSync(selfContact, listOfRSIPNodes.Contact[k], request2, nil, &wg, node)

			}

			wg.Wait()
			fmt.Println("Store Private key", time.Since(start2))

			// for k := 0; k < len(listOfRSIPNodes.Contact); k++ {
			start3 := time.Now()

			fmt.Println("latestContacts.Contact", len(latestContacts.Contact))

			// Store oracle and oracle group's public key among network
			for l := 0; l < len(latestContacts.Contact); l++ {

				if latestContacts.Contact[l].Address != selfContact.Address {
					wg.Add(1)
					// memberPublicToByte := system.PublicKeyToBytes(memberKeys[k])

					// // Store oracle public key across network
					// request1 := api.Request{
					// 	Action: "bls_RSIP_public",
					// 	Data:   memberPublicToByte,
					// }

					// go BootstrapSync(selfContact, latestContacts.Contact[l], request1, nil, &wg, node)

					// Store oracle group public key across network
					request2 := api.Request{
						Action:      "bls_RSIP_group_public",
						Data:        groupPublicToByte,
						Param:       "bls_tribe_" + strconv.Itoa(index) + "_RSIP_group_" + strconv.Itoa(j),
						RSIPGroup:   int32(j),
						TribeNumber: int32(index),
					}

					go BootstrapSync(selfContact, latestContacts.Contact[l], request2, nil, &wg, node)
				}

			}
			wg.Wait()
			fmt.Println("Store public key", time.Since(start3))
			// }

			listOfRSIPNodes = &api.Contacts{}
			// Free Oracle  Keys
			groupKey.Free()
			groupSecret.Free()

			for i := 0; i < len(listOfRSIPNodes.Contact); i++ {
				memberKeys[i].Free()
				memberSecrets[i].Free()
			}

		}
	}
	// }()
	fmt.Println("RSIP COMPLETED")
}

// Sync data to DHT from RSIP nodes
func syncBatchDatailsToGlobalDHT(selfContact api.Contact, node *UnityNode) {
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
}
