package findValues

import (
	"log"

	"github.com/golang/protobuf/proto"
	"github.com/unity-go/api"
	. "github.com/unity-go/unityNode"
	. "github.com/unity-go/util"
)

func GetValuesById(node *UnityNode, id []byte) api.Contacts {
	data := api.Contacts{}

	value, err := node.ValuesDB.Get(id[:], nil)
	if err != nil {
		if Debug == true {
			log.Println(err)
		}
	}
	proto.Unmarshal(value, &data)

	return data
}

func PutValuesById(node *UnityNode, id []byte, data []byte) error {
	err := node.ValuesDB.Put(id[:], data, nil)
	if err != nil {
		return err
	}
	return nil
}

func GetValue(node *UnityNode, id []byte) []byte {

	value, err := node.ValuesDB.Get(id[:], nil)
	if err != nil {
		if Debug == true {
			log.Println(err)
		}
	}

	return value
}
