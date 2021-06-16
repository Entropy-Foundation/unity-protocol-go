package mongodb

import (
	"fmt"
	"log"
	"time"

	"github.com/BurntSushi/toml"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/unity-go/api"
)

type Mongo struct {
	Server   string
	Database string
	db       *mgo.Database
}

type LogStruct struct {
	BatchID   string
	Value     string
	TimeStamp time.Time
}

type BatchIdStruct struct {
	BatchID string
}

// initialize connection with mongo
func (m *Mongo) Init() {
	if _, err := toml.DecodeFile("config.toml", &m); err != nil {
		log.Fatal(err)
	}
	m.Connect()
}

// connection to mongo
func (m *Mongo) Connect() {

	session, err := mgo.Dial(m.Server)
	if err != nil {
		log.Fatal(err)
	}

	m.db = session.DB(m.Database)
}

// add transaction in mongo
func (m *Mongo) Add(batch api.TransactionWithSignature, ch chan []byte) {

	err := m.db.C("batches").Insert(&batch)
	if err != nil {
		panic(err)
	}

	ch <- []byte("Done")
}

// Add transactions in bulk
func (m *Mongo) BulkAdd(batch []interface{}) {

	bulk := m.db.C("batches").Bulk()

	// fmt.Println(batch)

	bulk.Insert(batch...)
	_, bulkErr := bulk.Run()
	if bulkErr != nil {
		fmt.Println("bulkErr", bulkErr)
		// panic(bulkErr)
	}
	// if err != nil {
	// 	panic(err)
	// }

}

// Store batch ID in collection
func (m *Mongo) StoreBatchId(batchId string) {

	batchIdStruct := BatchIdStruct{
		BatchID: batchId,
	}

	err := m.db.C("batcheIds").Insert(&batchIdStruct)
	if err != nil {
		panic(err)
	}
}

// get batchid from collection
func (m *Mongo) GetBatchId(batchId string) (BatchIdStruct, error) {

	batchIdStr := BatchIdStruct{}
	err := m.db.C("batcheIds").Find(bson.M{"batchid": batchId}).One(&batchIdStr)
	// if err != nil {
	// 	panic(err)
	// }

	return batchIdStr, err
}

// Add logs to collection
func (m *Mongo) CreateLogs(id string, data string) {

	logs := LogStruct{
		BatchID:   id,
		Value:     data,
		TimeStamp: time.Now(),
	}

	err := m.db.C("logs").Insert(&logs)
	if err != nil {
		panic(err)
	}
}

// Get logs from collection
func (m *Mongo) GetLogs(batchIds []string) (logs []LogStruct) {

	err := m.db.C("logs").Find(bson.M{"batchid": bson.M{"$in": batchIds}}).All(&logs)
	if err != nil {
		panic(err)
	}

	return
}

// Retrive transaction history
func (m *Mongo) RetrieveTransactionHistory(from, to, batchId string) (txs []api.TransactionWithSignature) {

	var err error
	if batchId == "" {

		if from != "" && to != "" {
			err = m.db.C("batches").Find(bson.M{"from": from, "to": to}).All(&txs)
		} else if from != "" {
			err = m.db.C("batches").Find(bson.M{"from": from}).All(&txs)
		} else if to != "" {
			err = m.db.C("batches").Find(bson.M{"to": to}).All(&txs)
		}
	} else {
		if batchId != "" && from != "" && to != "" {
			err = m.db.C("batches").Find(bson.M{"from": from, "to": to, "batchid": batchId}).All(&txs)
			fmt.Println(txs)
		} else {
			err = m.db.C("batches").Find(bson.M{"batchid": batchId}).All(&txs)
			fmt.Println(txs)
		}
	}

	if err != nil {
		panic(err)
	}

	return
}

// Retrive transaction history by batch id
func (m *Mongo) RetrieveTransactionByBatchIds(batchIds []string) (txs []api.TransactionWithSignature) {

	var err error

	err = m.db.C("batches").Find(bson.M{"batchid": bson.M{"$in": batchIds}}).All(&txs)

	if err != nil {
		panic(err)
	}

	return
}
