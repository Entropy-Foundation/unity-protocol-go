package transport

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	"github.com/soheilhy/cmux"
	. "github.com/unity-go/transaction"
	. "github.com/unity-go/unityCoreRPC"
	. "github.com/unity-go/unityNode"
	. "github.com/unity-go/util"
	"google.golang.org/grpc"

	"github.com/unity-go/api"
	. "github.com/unity-go/service"

	"net"
	"net/http"
	"net/rpc"
)

func ServeHTTP(l net.Listener, node *UnityNode, selfContact api.Contact) {
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
	r.Post("/signTransaction", func(w http.ResponseWriter, r *http.Request) {
		HandleSignTransaction(w, r, node)
	})
	r.Get("/value/{id}", func(w http.ResponseWriter, r *http.Request) {
		HandleGetValue(w, r, node)
	})
	r.Get("/startElection", func(w http.ResponseWriter, r *http.Request) {
		HandleElection(w, r, node, selfContact)
	})
	r.Post("/create-client", func(w http.ResponseWriter, r *http.Request) {
		CreateClient(w, r, node, selfContact)
	})
	r.Post("/create-data", func(w http.ResponseWriter, r *http.Request) {
		CreateData(w, r, node, selfContact)
	})

	// r.Post("/fill-leader-queue", func(w http.ResponseWriter, r *http.Request) {
	// 	FillLeaderQueue(w, r, node, selfContact)
	// })
	r.Post("/self-transaction-processor", func(w http.ResponseWriter, r *http.Request) {
		SelfCallingTxProcessor(w, r, node, selfContact)
	})
	r.Post("/testnet-log", func(w http.ResponseWriter, r *http.Request) {
		TestnetLog(w, r, node, selfContact)
	})
	r.Get("/transaction-history", func(w http.ResponseWriter, r *http.Request) {
		TransactionHistory(w, r, node, selfContact)
	})

	r.Post("/transactions", func(w http.ResponseWriter, r *http.Request) {
		TransactionsByBatchIds(w, r, node, selfContact)
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

func ServeGRPC(l net.Listener, newNode *UnityNode) {

	// create a gRPC server object
	grpcServer := grpc.NewServer(grpc.MaxRecvMsgSize(MaxMsgSize), grpc.MaxSendMsgSize(MaxMsgSize))
	// create a server instance
	s := UnityCore2{
		Node: newNode,
	}

	api.RegisterPingServer(grpcServer, &s)
	// defer l.Close()

	go grpcServer.Serve(l)
}

func InitializeTransport(address string, node *UnityNode, selfContact api.Contact) {
	lister, err := net.Listen("tcp", address)
	if err != nil {
		panic(err)
	}

	m := cmux.New(lister)
	grpcL := m.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
	httpl := m.Match(cmux.HTTP1Fast())
	// If not matched by HTTP, we assume it is an RPC connection.
	rpcl := m.Match(cmux.Any())

	go ServeGRPC(grpcL, node)
	go ServeHTTP(httpl, node, selfContact)
	go ServeRPC(rpcl, node)

	go m.Serve()
	// return nil
}
