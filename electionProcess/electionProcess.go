package electionProcess

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/joho/godotenv"

	"github.com/unity-go/api"
	. "github.com/unity-go/dkg"
	. "github.com/unity-go/findValues"

	// . "github.com/unity-go/transport"
	"math/rand"
	"regexp"
	"time"

	. "github.com/unity-go/unityCoreRPC"
	. "github.com/unity-go/unityNode"
	. "github.com/unity-go/util"
)

func Regex(s string) string {
	re := regexp.MustCompile(`\n`)
	return re.ReplaceAllString(s, "$1W")
}

var SetupTime time.Time

func Scheduler(node *UnityNode, selfContact api.Contact) {
	fmt.Println("START-NETWORK-SETUP")
	SetupTime = time.Now()
	fmt.Println("Debug-Value", Debug)

	if Debug == true {
		fmt.Println("", "First Oracle------------------------------")
	}

	Oracles := os.Getenv("NUMBER_OF_ORACLES")
	OraclesGroups := os.Getenv("NUMBER_OF_ORACLE_GROUPS")
	numberOfAlphaOracles := os.Getenv("NUMBER_OF_ALPHA")

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
	listOfOracles := &api.Contacts{}
	contactsFromOwnDb := GetValuesById(node, []byte(string(node.NodeID[:])))
	contactsFromOwnDbNew := GetValuesById(node, []byte(string(node.NodeID[:])))

	fmt.Println("contactsFromOwnDbNew", len(contactsFromOwnDbNew.Contact))
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

				// selectedOracleData, _ := proto.Marshal(*selectedContactToChange)

				// t := time.Now()

				// if a != 1 {
				// 	secretShares, _ = FindNodeForDkg(selfContact, listOfOracles[a-1], "Oracles", selectedOracleData)

				// }

				if Debug == true {
					fmt.Println("")
					fmt.Println(a, "selected node is", *selectedContactToChange, len(contactsFromOwnDb.Contact))
				}
				dataToStoreInSelf, _ := proto.Marshal(&contactsFromOwnDb)
				err := PutValuesById(node, []byte(string(node.NodeID[:])), dataToStoreInSelf)
				if err != nil {
					return
				}

				// data, _ := proto.Marshal(contactsFromOwnDb)
				// for i := 0; i <= len(contactsFromOwnDb)-1; i++ {
				// 	if contactsFromOwnDb[i].Address != selfContact.Address {
				// 		_, _, err := Bootstrap(contactsFromOwnDb[i], selfContact, "Oracle-Update", data)
				// 		if err != nil {
				// 			fmt.Println("Bootstrap error:", err)
				// 		}
				// 	}
				// }
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

						go BootstrapSync(selfContact, contactsFromOwnDb.Contact[i], request, nil, nil)

					}

					secretKey := ""

					for j := 0; j <= len(listOfOracles.Contact)-1; j++ {

						if j != 0 {
							keyInByte, _ := json.Marshal("dkg_Secret_Oracles")

							request := api.Request{
								Action: "Retrieve-RSIP-Group-Seckey",
								Data:   keyInByte,
								Param:  "secret-key",
							}

							getSecretKey := BootstrapSync(selfContact, listOfOracles.Contact[0], request, nil, nil)

							request2 := api.Request{
								Action: "dkg_Secret_Oracles",
								Data:   getSecretKey.Data,
								IsDKG:  true,
							}
							secretKeyData := BootstrapSync(*listOfOracles.Contact[j], listOfOracles.Contact[0], request2, nil, nil)
							json.Unmarshal(secretKeyData.Data, &secretKey)

						}
					}

					request = api.Request{
						Action: "dkg_Secret_Oracles",
						Data:   []byte(secretKey),
					}
					// store group's secret key among group
					for k := 0; k <= len(listOfOracles.Contact)-1; k++ {

						go BootstrapSync(selfContact, listOfOracles.Contact[k], request, nil, nil)

						if err != nil {
							fmt.Println("Share Secret Key Of Group Error:", err)
						}
					}

					groupPublicKey := GenerateSharedPublicKey(secretKey)
					if Debug == true {
						fmt.Println(groupPublicKey, "groupPublicKey")
					}

					request = api.Request{
						Action: "dkg_Public_Oracles",
						Data:   []byte(groupPublicKey),
					}
					// store group's public key across the network
					for k := 0; k <= len(contactsFromOwnDb.Contact)-1; k++ {

						go BootstrapSync(selfContact, contactsFromOwnDb.Contact[k], request, nil, nil)

					}
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

							listOfNodesOfTribeInByte, _ := proto.Marshal(listOfNodesOfTribe)

							errorFromPut := PutValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_Epoch_1_Oracle_Group_"+strconv.Itoa(i)), listOfNodesOfTribeInByte)
							if errorFromPut != nil {
								fmt.Println(errorFromPut)
							}
							var wg sync.WaitGroup
							request := api.Request{
								Action: "_Epoch_1_Oracle_Group_" + strconv.Itoa(i),
								Data:   listOfNodesOfTribeInByte,
							}
							for j := 0; j <= len(contactsFromOwnDb.Contact)-1; j++ {

								wg.Add(1)

								go BootstrapSync(selfContact, contactsFromOwnDb.Contact[j], request, nil, &wg)

							}

							wg.Wait()

							if len(listOfOraclesToFormGroups.Contact) == 1 {

							}
						}
					}

					startOrganizingTribe(*listOfOracles, contactsFromOwnDb, selfContact, node, numberOfOracleGroups)
					quit <- true
				}
				// do stuff
			case <-quit:
				if Debug == true {
					fmt.Println("Stopped")
				}

				ticker.Stop()
				return
			}
		}
	}()
}

func startOrganizingTribe(listOfOracles, contactsFromOwnDb api.Contacts, selfContact api.Contact, node *UnityNode, numberOfOracleGroups int) {

	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}

	Tribes := os.Getenv("NUMBER_OF_TRIBES")
	numberOfTribes, _ := strconv.Atoi(Tribes)

	ticker := time.NewTicker(50 * time.Millisecond)
	quit := make(chan bool)

	// Generate DKG for each Group of Oracle
	for i := 0; i <= numberOfOracleGroups; i++ {
		// if i == 0 {
		secretKey := ""
		listOfNodes := GetValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_Epoch_1_Oracle_Group_"+strconv.Itoa(i)))
		if Debug == true {
			fmt.Println("-------------"+strconv.Itoa(i)+"Group -------------\n", len(listOfNodes.Contact))
		}

		for j := 0; j <= len(listOfNodes.Contact)-1; j++ {
			if j != 0 {
				key := hex.EncodeToString(node.NodeID[:]) + "_dkg_Secret_Epoch_1_Oracle_Group_" + strconv.Itoa(i)
				getSecretKey, _ := node.ValuesDB.Get([]byte(key), nil)

				request := api.Request{
					Action: "_dkg_Secret_Epoch_1_Oracle_Group_" + strconv.Itoa(i),
					Data:   getSecretKey,
					IsDKG:  true,
				}

				secretKeyData := BootstrapSync(*listOfNodes.Contact[j], listOfNodes.Contact[0], request, nil, nil)
				json.Unmarshal(secretKeyData.Data, &secretKey)
			}
		}

		var wg sync.WaitGroup
		request := api.Request{
			Action: "_dkg_Secret_Epoch_1_Oracle_Group_" + strconv.Itoa(i),
			Data:   []byte(secretKey),
		}
		// store group's secret key among group
		for k := 0; k <= len(listOfNodes.Contact)-1; k++ {

			wg.Add(1)

			go BootstrapSync(selfContact, listOfNodes.Contact[k], request, nil, &wg)

		}
		wg.Wait()

		groupPublicKey := GenerateSharedPublicKey(secretKey)
		if Debug == true {
			fmt.Println(groupPublicKey, "groupPublicKey", i)
		}

		request = api.Request{
			Action: "_dkg_Public_Epoch_1_Oracle_Group_" + strconv.Itoa(i),
			Data:   []byte(groupPublicKey),
		}
		// store group's public key across the network
		for k := 0; k <= len(contactsFromOwnDb.Contact)-1; k++ {

			wg.Add(1)

			go BootstrapSync(selfContact, contactsFromOwnDb.Contact[k], request, nil, &wg)
		}

		wg.Wait()

	}

	allNodes := &api.Contacts{}
	for i := 0; i <= len(contactsFromOwnDb.Contact)-1; i++ {
		allNodes.Contact = append(allNodes.Contact, contactsFromOwnDb.Contact[i])
	}
	if Debug == true {
		fmt.Println(len(allNodes.Contact), "before", len(listOfOracles.Contact))
	}

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
	if Debug == true {
		fmt.Println(len(allNodes.Contact), "After", len(listOfOracles.Contact))
	}
	go func() {
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

					errorFromPut := PutValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_Tribe_Group_"+strconv.Itoa(i)), listOfNodesOfTribeInByte)
					if errorFromPut != nil {
						fmt.Println(errorFromPut)
					}

					var wg sync.WaitGroup
					request := api.Request{
						Action: "_Tribe_Group_" + strconv.Itoa(i),
						Data:   listOfNodesOfTribeInByte,
					}
					for j := 0; j <= len(contactsFromOwnDb.Contact)-1; j++ {

						wg.Add(1)

						go BootstrapSync(selfContact, contactsFromOwnDb.Contact[j], request, nil, &wg)

					}
					wg.Wait()

					if i == numberOfTribes {
						electionProccessForLeaders(listOfOracles, contactsFromOwnDb, selfContact, node, numberOfTribes)

						quit <- true
					}
				}

			case <-quit:
				if Debug == true {
					fmt.Println("Stopped")
				}
				ticker.Stop()
				return
			}
		}
	}()
}

func electionProccessForLeaders(listOfOracles, latestContacts api.Contacts, selfContact api.Contact, node *UnityNode, numberOfTribes int) {
	if Debug == true {
		fmt.Println("")
		fmt.Println("-------------------Starting Election Process Of Leaders------------------")
	}

	var wg sync.WaitGroup

	selectedLeadersForGroup1 := &api.Contacts{}
	selectedLeadersForGroup2 := &api.Contacts{}
	var secretShares string
	for i := 0; i <= numberOfTribes; i++ {
		// value, _ := FindNode(selfContact, listOfOracles[i], "PING", []byte{})
		listOfNodesOfTribe := GetValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_Tribe_Group_"+strconv.Itoa(i)))
		min := 0
		max := len(listOfNodesOfTribe.Contact)
		randNumber1 := rand.Intn(max-min) + min
		randNumber2 := rand.Intn(max-min) + min

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

		listOfNodesOfTribeInByte, _ := proto.Marshal(&listOfNodesOfTribe)

		errorFromSave := PutValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_Tribe_Group_"+strconv.Itoa(i)), listOfNodesOfTribeInByte)
		if errorFromSave != nil {
			fmt.Println(errorFromSave)
		}

		request := api.Request{
			Action: "_Tribe_Group_" + strconv.Itoa(i),
			Data:   listOfNodesOfTribeInByte,
		}
		for m := 0; m <= len(latestContacts.Contact)-1; m++ {

			wg.Add(1)

			go BootstrapSync(selfContact, latestContacts.Contact[m], request, nil, &wg)
		}

		wg.Wait()

		// if i != 0 {
		// 	secretShares, _ = FindNodeForDkg(selfContact, selectedLeaders[i], "Leaders", selectedLeadersData)
		// }

	}

	for i := 1; i < 3; i++ {
		selectedLeadersForGroup := &api.Contacts{}

		if i == 1 {
			selectedLeadersForGroup = selectedLeadersForGroup1
		} else {
			selectedLeadersForGroup = selectedLeadersForGroup2
		}
		listOfSelectedLeaders, _ := proto.Marshal(selectedLeadersForGroup)
		request1 := api.Request{
			Action: "Leaders-List-Save",
			Data:   listOfSelectedLeaders,
			Param:  "Leaders_Epoch_1_Group_" + strconv.Itoa(i),
		}
		for j := 0; j <= len(latestContacts.Contact)-1; j++ {

			wg.Add(1)

			go BootstrapSync(selfContact, latestContacts.Contact[j], request1, nil, &wg)
		}

	}

	wg.Wait()

	// listOfSelectedLeaders2, _ := proto.Marshal(selectedLeadersForGroup2)
	// request2 := api.Request{
	// 	Action: "Leaders-List-Save",
	// 	Data:   listOfSelectedLeaders2,
	// 	Param:  "Leaders_Epoch_1_Group_2",
	// }
	// for j := 0; j <= len(latestContacts.Contact)-1; j++ {

	// 	wg.Add(1)

	// 	go BootstrapSync(selfContact, latestContacts.Contact[j], request2, nil, &wg)
	// }

	// wg.Wait()

	if Debug == true {
		fmt.Println("secretShares", secretShares, "Main.GO")
	}

	for i := 1; i < 3; i++ {
		selectedLeadersForGroup := &api.Contacts{}
		if i == 1 {
			selectedLeadersForGroup = selectedLeadersForGroup1
		} else {
			selectedLeadersForGroup = selectedLeadersForGroup2
		}
		for j := 0; j <= len(selectedLeadersForGroup.Contact)-1; j++ {
			if j != 0 {

				keyInByte, _ := json.Marshal("dkg_Secret_Leaders_Group_" + strconv.Itoa(i))

				request := api.Request{
					Action: "Retrieve-RSIP-Group-Seckey",
					Data:   keyInByte,
					Param:  "secret-key",
				}

				getSecretKey := BootstrapSync(selfContact, selectedLeadersForGroup.Contact[0], request, nil, nil)

				request2 := api.Request{
					Action: "dkg_Secret_Leaders_Group_" + strconv.Itoa(i),
					Data:   getSecretKey.Data,
					IsDKG:  true,
				}

				secretSharesData := BootstrapSync(*selectedLeadersForGroup.Contact[j], selectedLeadersForGroup.Contact[0], request2, nil, nil)
				json.Unmarshal(secretSharesData.Data, &secretShares)

				fmt.Println(secretShares, "secretShares")

			}
		}

		request := api.Request{
			Action: "Shared-Secret-Key-Leader",
			Data:   []byte(secretShares),
			Param:  "dkg_Secret_Leaders_Group_" + strconv.Itoa(i),
		}
		for k := 0; k <= len(selectedLeadersForGroup.Contact)-1; k++ {

			wg.Add(1)

			go BootstrapSync(selfContact, selectedLeadersForGroup.Contact[k], request, nil, &wg)
		}

		wg.Wait()

		storeGroupsPublicKey(secretShares, latestContacts, selfContact, "Shared-Public-Key-Leader", strconv.Itoa(i))

		// err := PutValuesById(node, []byte(string(node.NodeID[:])), listOfSelectedLeaders)
		// if err != nil {
		// 	return
		// }

		getPublicKey, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(node.NodeID[:])+"dkg_Public_Leaders_Group_"+strconv.Itoa(i)), nil)
		// if Debug == true {
		fmt.Println("")
		fmt.Println("-------------------Generated Public Key Of Leaders Group By DKG------------------------------")
		fmt.Println(string(getPublicKey), "public")
		// }

	}

	groupOfRSIPFromTribes(node, selfContact, numberOfTribes, latestContacts)

	TribalMappingForDecimal(node, selfContact, numberOfTribes, latestContacts)

	t2 := time.Now()
	t2.Second()
	if Debug == true {
		fmt.Println(t2, "---------------------------------------end time")
	}
}

func TribalMappingForDecimal(node *UnityNode, selfContact api.Contact, numberOfTribes int, latestContacts api.Contacts) {

	if Debug == true {
		fmt.Println("------------------------------------Tribal Mapping--------------------------------------")
	}

	var wg sync.WaitGroup

	for j := 0; j <= numberOfTribes; j++ {
		if Debug == true {
			fmt.Println("\n------------------------------------Tribe->", j, "--------------------------------------")
		}

		numberOfRSIPGroup := os.Getenv("NUMBER_OF_RSIP_GROUP")
		numberOfRSIPGroupInInt, _ := strconv.Atoi(numberOfRSIPGroup)

		secKeysOfAllRSIP := api.RSIPDecimals{}

		for k := 0; k <= numberOfRSIPGroupInInt; k++ {

			GroupNodes := &api.Contacts{}
			contacts, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(selfContact.ID[:])+"-Tribe_"+strconv.Itoa(j)+"_RSIP_GROUP_"+strconv.Itoa(k)), nil)
			proto.Unmarshal(contacts, GroupNodes)

			key := "_Tribe_" + strconv.Itoa(j) + "_dkg_Secret_RSIP_Group_" + strconv.Itoa(k)
			keyInByte, _ := json.Marshal(key)

			request := api.Request{
				Action: "Retrieve-RSIP-Group-Seckey",
				Data:   keyInByte,
			}
			res := BootstrapSync(selfContact, GroupNodes.Contact[0], request, nil, nil)

			decimalKey := ""
			json.Unmarshal(res.Data, &decimalKey)

			decKeyWithGroup := api.RSIPDecimal{}
			decKeyWithGroup.Group = int32(k)
			decKeyWithGroup.Decimal = decimalKey
			if Debug == true {
				fmt.Println("Sec-Decimal-DKG-STRUCT-WITH-GROUP[", j, "]-----------------", decKeyWithGroup)
			}

			secKeysOfAllRSIP.RSIPDecimal = append(secKeysOfAllRSIP.RSIPDecimal, &decKeyWithGroup)
		}

		// Store secKeysOfAllRSIP across network
		fmt.Println("LEN-OF-CONTACTS-----------------", len(latestContacts.Contact))

		// sort.Slice(secKeysOfAllRSIP, func(i, j int) bool {
		// 	return secKeysOfAllRSIP.RSIPDecimal[i].Decimal < secKeysOfAllRSIP.RSIPDecimal[j].Decimal
		// })

		fmt.Println(secKeysOfAllRSIP)

		decKeysInByte, _ := proto.Marshal(&secKeysOfAllRSIP)
		request := api.Request{
			Action: "_Tribe_" + strconv.Itoa(j) + "_Decimal_DKG",
			Data:   decKeysInByte,
		}
		for l := 0; l < len(latestContacts.Contact); l++ {

			wg.Add(1)

			go BootstrapSync(selfContact, latestContacts.Contact[l], request, nil, &wg)

		}

		wg.Wait()

	}
	EndTime := time.Since(SetupTime)
	fmt.Println("END-NETWORK-SETUP", EndTime)
}

func groupOfRSIPFromTribes(node *UnityNode, selfContact api.Contact, numberOfTribes int, latestContacts api.Contacts) {
	numberOfRSIPGroup := os.Getenv("NUMBER_OF_RSIP_GROUP")
	numberOfRSIPGroupInInt, _ := strconv.Atoi(numberOfRSIPGroup)
	numberOfAlphaOracles := os.Getenv("NUMBER_OF_ALPHA")
	numberOfAlphaNodes, _ := strconv.Atoi(numberOfAlphaOracles)

	var wg sync.WaitGroup

	for j := 0; j <= numberOfTribes; j++ {

		// get list of oracles tribe
		listOfOracles := GetValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_Tribe_Group_"+strconv.Itoa(j)))
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

				errorFromPut := PutValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"-Tribe_"+strconv.Itoa(j)+"_RSIP_GROUP_"+strconv.Itoa(i)), listOfOraclesInByte)
				if errorFromPut != nil {
					fmt.Println(errorFromPut)
				}

				request := api.Request{
					Action: "-Tribe_" + strconv.Itoa(j) + "_RSIP_GROUP_" + strconv.Itoa(i),
					Data:   listOfOraclesInByte,
				}
				for k := 0; k <= len(latestContacts.Contact)-1; k++ {

					wg.Add(1)

					go BootstrapSync(selfContact, latestContacts.Contact[k], request, nil, &wg)
				}

				wg.Wait()

			}
		}
	}
	for j := 0; j <= numberOfTribes; j++ {

		// Generate DKG for each Group
		for i := 0; i <= numberOfRSIPGroupInInt; i++ {
			// if i == 0 {
			secretKey := ""
			listOfRSIPNodes := GetValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"-Tribe_"+strconv.Itoa(j)+"_RSIP_GROUP_"+strconv.Itoa(i)))
			for m := 0; m <= len(listOfRSIPNodes.Contact)-1; m++ {
				if m != 0 {

					keyInByte, _ := json.Marshal("_Tribe_" + strconv.Itoa(j) + "_dkg_Secret_RSIP_Group_" + strconv.Itoa(i))

					request := api.Request{
						Action: "Retrieve-RSIP-Group-Seckey",
						Data:   keyInByte,
						Param:  "secret-key",
					}

					secretKeyData2 := BootstrapSync(selfContact, listOfRSIPNodes.Contact[0], request, nil, nil)

					request2 := api.Request{
						Action: "_Tribe_" + strconv.Itoa(j) + "_dkg_Secret_RSIP_Group_" + strconv.Itoa(i),
						Data:   secretKeyData2.Data,
						IsDKG:  true,
					}

					secretKeyData := BootstrapSync(*listOfRSIPNodes.Contact[m], listOfRSIPNodes.Contact[0], request2, nil, nil)
					json.Unmarshal(secretKeyData.Data, &secretKey)

				}
			}

			request := api.Request{
				Action: "_Tribe_" + strconv.Itoa(j) + "_dkg_Secret_RSIP_Group_" + strconv.Itoa(i),
				Data:   []byte(secretKey),
			}
			// store group's secret key among group
			for k := 0; k <= len(listOfRSIPNodes.Contact)-1; k++ {

				wg.Add(1)

				go BootstrapSync(selfContact, listOfRSIPNodes.Contact[k], request, nil, &wg)
			}

			wg.Wait()

			groupPublicKey := GenerateSharedPublicKey(secretKey)
			if Debug == true {
				fmt.Println(groupPublicKey, "groupPublicKey", i)
			}

			request = api.Request{
				Action: "_Tribe_" + strconv.Itoa(j) + "_dkg_Public_RSIP_Group_" + strconv.Itoa(i),
				Data:   []byte(groupPublicKey),
			}
			// store group's public key across the network
			for k := 0; k <= len(latestContacts.Contact)-1; k++ {

				wg.Add(1)

				go BootstrapSync(selfContact, latestContacts.Contact[k], request, nil, &wg)
			}

			wg.Wait()

		}

	}
}

func addSecretShares(node *UnityNode, target, self api.Contact, action string) string {
	var getSecretKey []byte

	key := ""

	if action == "Leaders" {
		key = hex.EncodeToString(node.NodeID[:]) + "dkg_Secret_Leaders"
	} else {
		key = hex.EncodeToString(node.NodeID[:]) + "dkg_Secret_Oracles"
	}

	getSecretKey, _ = node.ValuesDB.Get([]byte(key), nil)

	response, errFromListUpdate := PingNodeForDKG(&self, &target, action, getSecretKey)

	fmt.Println("rrrrrrrrrrrrr", response)
	if errFromListUpdate != nil {
		fmt.Println("List Update Error:", errFromListUpdate)
	}
	return response
}

func storeGroupsPublicKey(key string, latestContacts api.Contacts, selfContact api.Contact, action string, i string) {

	var wg sync.WaitGroup

	groupPublicKey := GenerateSharedPublicKey(key)
	request := api.Request{
		Action: action,
		Data:   []byte(groupPublicKey),
		Param:  "dkg_Public_Leaders_Group_" + i,
	}
	for j := 0; j <= len(latestContacts.Contact)-1; j++ {
		if Debug == true {
			fmt.Println(latestContacts.Contact[j].Address, selfContact.Address, "Contacts----------------------")
		}

		wg.Add(1)

		go BootstrapSync(selfContact, latestContacts.Contact[j], request, nil, &wg)

	}

	wg.Wait()

}
