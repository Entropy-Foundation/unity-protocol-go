package transport

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	// Gorilla "github.com/gorilla/websocket"
	"github.com/soheilhy/cmux"
	"github.com/unity-go/api"
	// . "github.com/unity-go/findValues"
	. "github.com/unity-go/service"
	. "github.com/unity-go/transaction"
	. "github.com/unity-go/unityCoreRPC"
	. "github.com/unity-go/unityNode"
	. "github.com/unity-go/util"
	"github.com/unity-go/websocket"
	"google.golang.org/grpc"
	// "log"
	"net"
	"net/http"
	"net/rpc"
	"strconv"
	// "strings"
)

func ServeHTTP(l net.Listener, node *UnityNode, selfContact api.Contact, nodeBlock *NodeBlock) {
	r := chi.NewRouter()

	cors := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	})

	r.Use(cors.Handler)

	r.Get("/clients", func(w http.ResponseWriter, r *http.Request) {
		HandleGetClients(w, r, node)
	})
	// r.Post("/signTransaction", func(w http.ResponseWriter, r *http.Request) {
	// 	HandleSignTransaction(w, r, node)
	// })
	r.Get("/value/{id}", func(w http.ResponseWriter, r *http.Request) {
		HandleGetValue(w, r, node)
	})
	r.Get("/startElection", func(w http.ResponseWriter, r *http.Request) {
		HandleElection(w, r, node, selfContact, nodeBlock)
	})
	r.Post("/create-client", func(w http.ResponseWriter, r *http.Request) {
		CreateClient(w, r, node, selfContact, nodeBlock)
	})

	r.Get("/get-pki", func(w http.ResponseWriter, r *http.Request) {
		CreatePKI(w, r, node, selfContact)
	})

	r.Post("/create-data", func(w http.ResponseWriter, r *http.Request) {
		CreateData(w, r, node, selfContact, nodeBlock)
	})

	// r.Post("/fill-leader-queue", func(w http.ResponseWriter, r *http.Request) {
	// 	FillLeaderQueue(w, r, node, selfContact)
	// })
	r.Post("/self-transaction-processor", func(w http.ResponseWriter, r *http.Request) {
		SelfCallingTxProcessor(w, r, node, selfContact, nodeBlock)
	})
	r.Post("/testnet-log", func(w http.ResponseWriter, r *http.Request) {
		TestnetLog(w, r, node, selfContact)
	})
	r.Get("/transaction-history", func(w http.ResponseWriter, r *http.Request) {
		TransactionHistory(w, r, node, selfContact)
	})

	r.Get("/sync-transactions", func(w http.ResponseWriter, r *http.Request) {
		SyncTransactionToCentralDB(w, r, node, selfContact)
	})

	r.Post("/transactions", func(w http.ResponseWriter, r *http.Request) {
		TransactionsByBatchIds(w, r, node, selfContact)
	})

	r.Get("/connections", func(w http.ResponseWriter, r *http.Request) {
		// TransactionsByBatchIds(w, r, node, selfContact)
		length := len(node.Connections)

		w.Write([]byte(strconv.Itoa(length)))
	})

	r.Get("/block", func(w http.ResponseWriter, r *http.Request) {

		// block := GetValue(node, []byte(string(node.NodeID[:])+"_Block_"))

		block, _ := json.Marshal(&nodeBlock)

		w.Write(block)
	})

	s := &http.Server{
		Handler: r,
	}
	if err := s.Serve(l); err != cmux.ErrListenerClosed {
		panic(err)
	}
}

func ServeRPC(l net.Listener, newNode *UnityNode) {
	s := rpc.NewServer()
	if err := s.Register(&UnityCore{
		Node: newNode,
	}); err != nil {
		panic(err)
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			if err != cmux.ErrListenerClosed {
				panic(err)
			}
			return
		}
		go s.ServeConn(conn)
	}
}

func ServeGRPC(l net.Listener, newNode *UnityNode, nodeBlock *NodeBlock) {

	// create a gRPC server object
	grpcServer := grpc.NewServer(grpc.MaxRecvMsgSize(MaxMsgSize), grpc.MaxSendMsgSize(MaxMsgSize))
	// create a server instance
	s := UnityCore2{
		Node:      newNode,
		NodeBlock: nodeBlock,
	}

	api.RegisterPingServer(grpcServer, &s)
	// defer l.Close()

	// go grpcServer.Serve(l)
	if err := grpcServer.Serve(l); err != cmux.ErrListenerClosed {
		panic(err)
	}
}

func Echo(w http.ResponseWriter, r *http.Request, node *UnityNode, pool *websocket.Pool) {

	conn, err := websocket.Upgrade(w, r)
	if err != nil {
		fmt.Fprintf(w, "%+v\n", err)
	}

	client := &websocket.Client{
		Conn: conn,
		Pool: pool,
	}

	pool.Register <- client
	client.Read()

	// node.WsConnection = c
	// defer c.Close()
	// for {
	// 	mt, message, err := c.ReadMessage()
	// 	if err != nil {
	// 		log.Println("read:", err)
	// 		break
	// 	}
	// 	log.Printf("recv: %s", message, mt)
	// 	err = c.WriteMessage(mt, message)
	// 	if err != nil {
	// 		log.Println("write:", err)
	// 		break
	// 	}
	// }
}

func InitializeTransport(address string, node *UnityNode, selfContact, firstContact api.Contact, nodeBlock *NodeBlock) {
	fmt.Println(address, "address")
	lister, err := net.Listen("tcp", address)
	if err != nil {
		panic(err)
	}

	pool := websocket.NewPool()
	go pool.Start()

	m := cmux.New(lister)
	grpcL := m.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
	httpl := m.Match(cmux.HTTP1Fast())
	// If not matched by HTTP, we assume it is an RPC connection.
	// rpcl := m.Match(cmux.Any())

	go ServeGRPC(grpcL, node, nodeBlock)
	go ServeHTTP(httpl, node, selfContact, nodeBlock)
	// go ServeRPC(rpcl, node)

	go m.Serve()

	// if firstContact.Address == "" {
	// 	newAddress := strings.Replace(selfContact.Address, ":6000", ":9000", -1)
	// 	http.HandleFunc("/socket.io", func(w http.ResponseWriter, r *http.Request) {
	// 		Echo(w, r, node, pool)
	// 	})
	// 	log.Fatal(http.ListenAndServe(newAddress, nil))
	// } else {

	// 	s := firstContact.Address
	// 	sz := len(s)

	// 	if sz > 0 && string(s[sz-5]) == ":" {
	// 		s = s[:sz-4]
	// 	}

	// 	addressToDial := "ws://" + string(s) + "9000/socket.io"
	// 	fmt.Println(addressToDial)
	// 	c, _, err := Gorilla.DefaultDialer.Dial(addressToDial, nil)
	// 	if err != nil {
	// 		log.Fatal("dial:", err)
	// 	}

	// 	node.WsConnection = c
	// }

	// return nil
}
