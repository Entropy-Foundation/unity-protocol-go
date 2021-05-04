package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	// "runtime"
	"strconv"
	// "time"

	// "sync"

	. "github.com/dfinity/go-dfinity-crypto/groupsig"
	"github.com/golang/protobuf/proto"
	"github.com/joho/godotenv"
	"github.com/unity-go/api"
	. "github.com/unity-go/dkg"
	. "github.com/unity-go/electionProcess"
	. "github.com/unity-go/findValues"

	. "github.com/unity-go/transport"
	. "github.com/unity-go/unityCoreRPC"
	. "github.com/unity-go/unityNode"
	. "github.com/unity-go/util"
)

func main() {
	// runtime.GOMAXPROCS(9000)
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	Debug, _ = strconv.ParseBool(os.Getenv("DEBUG"))
	ShowTime, _ = strconv.ParseBool(os.Getenv("TIME"))

	// initialize dkg
	InitBLS()

	Mdb.Init()

	// start with create new node and boostrapping
	run()
	done := make(chan bool)
	_ = <-done
	// fmt.Scanln()
}

func run() error {
	// parse env variables
	port, firstContact := parseFlags()
	// get systems ip
	out, _ := exec.Command("hostname", "-i").Output()

	selfAddress := ""
	selfAddress = fmt.Sprintf(Regex(string(out))+":%d", *port)

	fmt.Println(selfAddress)

	selfAddressObj := Address{}

	// nullContact := api.Contact{}
	if firstContact.Address == "" {
		selfAddressObj = Address{
			NodeID:   NodeID{190, 130, 221, 157, 195, 212, 75, 66, 9, 219, 166, 67, 159, 38, 189, 130, 253, 179, 57, 225},
			Address:  selfAddress,
			QuorumID: 1,
		}
	} else {
		selfAddressObj = Address{
			NodeID:   NewRandomNodeID(),
			Address:  selfAddress,
			QuorumID: 1,
		}
	}

	selfContact := &api.Contact{
		ID:       selfAddressObj.NodeID,
		Address:  selfAddress,
		QuorumID: int32(selfAddressObj.QuorumID),
	}

	// create new unity node
	newNode := NewUnityNode(&selfAddressObj)

	// generate public and private key for node and save in Db
	pubKey := GenerateKeysAndStoreInDb(newNode)

	// store static clients from seed file in levelDb
	readClientsAndStoreInDb(newNode)

	selfContact.PubKey = pubKey

	newSelfContact := &api.Contacts{}
	newSelfContact.Contact = append(newSelfContact.Contact, selfContact)
	myContact, _ := proto.Marshal(newSelfContact)
	// store self contact in Db
	errorFromGet := newNode.ValuesDB.Put(selfContact.ID[:], myContact, nil)

	if errorFromGet != nil {
		fmt.Println(errorFromGet)
	}

	InitializeTransport(selfContact.Address, newNode, *selfContact)
	// send self contact to target contact for bootstrapping
	if firstContact.Address != "" {

		response := BootstrapSync(*selfContact, &firstContact, api.Request{}, nil, nil)

		contacts := response.Contacts

		dataToSendInDB, _ := proto.Marshal(contacts)

		errorFromGet := newNode.ValuesDB.Put(selfContact.ID[:], dataToSendInDB, nil)
		if errorFromGet != nil {
			fmt.Println(errorFromGet)
		}

		// iterate over the latest contacts got from boot node
		for i := 0; i <= len(contacts.Contact)-1; i++ {
			if contacts.Contact[i].Address != selfContact.Address && contacts.Contact[i].Address != firstContact.Address {

				BootstrapSync(*selfContact, contacts.Contact[i], api.Request{}, nil, nil)

			}
		}
	}
	return nil
}

func parseFlags() (port *int, firstContact api.Contact) {

	firstNodeID := os.Getenv("FIRST_NODE_ID")
	isCompose := os.Getenv("IS_COMPOSE")
	isDocker := os.Getenv("IS_DOCKER")
	firstIp := os.Getenv("FIRST_IP")

	if os.Getenv("port") == "" {
		port = flag.Int("port", 6000, "a int")

	} else {
		data := os.Getenv("port")
		i, err := strconv.Atoi(data)
		fmt.Println(i, err)
		port = (flag.Int("port", i, "a int"))
	}
	firstID := flag.String("first-id", firstNodeID, "a hexideicimal node ID")
	firstIP := flag.String("first-ip", "", "the TCP address of an existing node")
	// quorumID = flag.Int("quorum-id", 1, "a int")

	flag.Parse()

	// fmt.Println(isCompose, "isCompose")

	if *firstIP == "" && isCompose != "true" && isDocker != "true" {
		firstIP = nil
	} else if isCompose != "true" && isDocker != "true" {
		firstContact = api.Contact{
			ID:      NewNodeID(*firstID),
			Address: *firstIP,
		}
	} else if isDocker != "true" && isCompose == "true" {
		firstContact = api.Contact{
			ID:      NewNodeID(*firstID),
			Address: "172.31.4.130:6000",
		}
	} else if isDocker == "true" {
		firstContact = api.Contact{
			ID:      NewNodeID(*firstID),
			Address: firstIp,
		}
	}
	return
}
func readClientsAndStoreInDb(node *UnityNode) {
	// node.Mutex.Lock()
	byteValue, _ := ioutil.ReadFile("seed.json")
	clientsForAPI := api.Clients{}

	if err := json.Unmarshal(byteValue, &clientsForAPI); err != nil {
		fmt.Println("proto.Unmarshal", err)
	}

	clientsToStore, _ := proto.Marshal(&clientsForAPI)

	err := PutValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_Clients"), clientsToStore)
	if err != nil {
		return
	}
	// node.Mutex.Unlock()

}

func chkErr(err error) {
	if err != nil {
		log.Fatalf("Error when calling function: %s", err)
	}
}
