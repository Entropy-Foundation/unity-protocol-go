package dkg

import (
// "encoding/hex"
// "fmt"
// "os"
// "strconv"

// "github.com/joho/godotenv"

// . "github.com/dfinity/go-dfinity-crypto/groupsig"
// . "github.com/dfinity/go-dfinity-crypto/rand"
// . "github.com/unity-go/findValues"
// . "github.com/unity-go/unityNode"
// . "github.com/unity-go/util"
)

// func InitBLS() {
// 	Init(bls.CurveFp254BNb)
// }

// func GenerateKeysAndStoreInDb(node *UnityNode) string {
// 	randomNumber := NewRand()
// 	sec := NewSeckeyFromRand(randomNumber.Deri(1))

// 	err := PutValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_Secret"), []byte(sec.GetHexString()))

// 	for i := 1; i < 3; i++ {
// 		errorFromDKG := PutValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"dkg_Secret_Leaders_Group_"+strconv.Itoa(i)), []byte(sec.GetHexString()))
// 		if Debug == true {
// 			fmt.Println(errorFromDKG)
// 		}

// 	}

// 	errorFromDKGOracles := PutValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"dkg_Secret_Oracles"), []byte(sec.GetHexString()))

// 	errorFromEnv := //  godotenv.Load()
// 	if errorFromEnv != nil {
// 		fmt.Println("Error loading .env file")
// 	}

// 	numberOfRSIPGroup := os.Getenv("NUMBER_OF_RSIP_GROUP")
// 	numberOfRSIPGroupInInt, _ := strconv.Atoi(numberOfRSIPGroup)
// 	Tribes := os.Getenv("NUMBER_OF_TRIBES")
// 	OraclesGroups := os.Getenv("NUMBER_OF_ORACLE_GROUPS")
// 	numberOfOracleGroups, _ := strconv.Atoi(OraclesGroups)
// 	numberOfTribes, _ := strconv.Atoi(Tribes)
// 	for i := 0; i <= numberOfTribes; i++ {
// 		for j := 0; j <= numberOfRSIPGroupInInt; j++ {
// 			errorFromDKGOracles := PutValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_Tribe_"+strconv.Itoa(i)+"_dkg_Secret_RSIP_Group_"+strconv.Itoa(j)), []byte(sec.GetHexString()))
// 			if Debug == true {
// 				fmt.Println(errorFromDKGOracles)
// 			}

// 		}
// 	}

// 	for i := 0; i <= numberOfOracleGroups; i++ {
// 		errorFromDKGOracles := PutValuesById(node, []byte(hex.EncodeToString(node.NodeID[:])+"_dkg_Secret_Epoch_1_Oracle_Group_"+strconv.Itoa(i)), []byte(sec.GetHexString()))
// 		if Debug == true {
// 			fmt.Println(errorFromDKGOracles)
// 		}
// 	}

// 	if err != nil || errorFromDKGOracles != nil {
// 		return ""
// 	}

// 	value, _ := node.ValuesDB.Get([]byte(hex.EncodeToString(node.NodeID[:])+"_Secret"), nil)

// 	if Debug == true {
// 		fmt.Println(string(value), "sec")
// 	}

// 	// // Add Seckeys
// 	// sum := AggregateSeckeys([]Seckey{*sec})
// 	// if sum == nil {
// 	// 	// t.Log("AggregateSeckeys")
// 	// }

// 	// // Pubkey
// 	pub := NewPubkeyFromSeckey(*sec)

// 	// pub.GetHexString()
// 	if Debug == true {
// 		fmt.Println(pub.GetHexString(), "pub")
// 	}

// 	if pub == nil {
// 		// t.Log("NewPubkeyFromSeckey")
// 	}
// 	return pub.GetHexString()

// 	// Sig
// 	// sig := Sign(*sec, []byte("hi"))
// 	// asig := AggregateSigs([]Signature{sig})
// 	// if !VerifyAggregateSig([]Pubkey{*pub}, []byte("hi"), asig) {
// 	// 	fmt.Println("Not Verified")
// 	// }
// }

// func AggregateSeckeysForDKG(sec1, sec2 []byte) string {
// 	secret1 := Seckey{}
// 	secret2 := Seckey{}
// 	secret1.SetHexString(string(sec1))
// 	secret2.SetHexString(string(sec2))
// 	sum := AggregateSeckeys([]Seckey{secret1, secret2})
// 	if sum == nil {
// 		if Debug == true {
// 			fmt.Println(sum)
// 		}
// 	}

// 	return sum.GetHexString()
// }

// func GenerateSharedPublicKey(key string) string {
// 	secret := &Seckey{}

// 	secret.SetHexString(key)

// 	pub := NewPubkeyFromSeckey(*secret)
// 	if pub == nil {
// 		fmt.Println("Error")
// 	}

// 	return pub.GetHexString()
// }
