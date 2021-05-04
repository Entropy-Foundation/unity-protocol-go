package bootstrap

import (
	"log"

	. "github.com/unity-go/unityCoreRPC"
	. "github.com/unity-go/unityNode"
	. "github.com/unity-go/util"

	"github.com/unity-go/api"

	"google.golang.org/grpc"
)

func FindNode(self, target &api.Contact, action string, data []byte) (Contacts, error) {
	client, err := DialContact(target)
	if err != nil {
		return nil, err
	}
	req := NewFindNodeRequest(self, target, action, data)
	res := FindNodeResponse{}
	defer client.Close()
	err = client.Call("UnityCore.SendMessageToTargetNode", &req, &res)
	if err != nil {
		return nil, err
	}

	return res.Value, nil
}

// func GRPC(self, target &api.Contact) {
// 	conn, err := grpc.Dial(target.Address, grpc.WithInsecure())
// 	if err != nil {
// 		log.Fatalf("did not connect: %v", err)
// 	}

// 	client := api.NewPingClient(conn)
// 	defer conn.Close()

// 	response, err := client.SayHello(context.Background(), &api.PingMessage{Greeting: "foo"})
// 	if err != nil {
// 		log.Fatalf("Error when calling SayHello: %s", err)
// 	}
// 	log.Printf("Response from server: %s", response.Greeting)
// }

func GetClient(target &api.Contact) (api.PingClient, *grpc.ClientConn) {
	conn, err := grpc.Dial(target.Address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	client := api.NewPingClient(conn)

	return client, conn
}
