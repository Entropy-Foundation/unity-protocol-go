package mongodb

import (
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

func (m *Mongo) Init() {
	if _, err := toml.DecodeFile("config.toml", &m); err != nil {
		log.Fatal(err)
	}
	m.Connect()
}

func (m *Mongo) Connect() {

	session, err := mgo.Dial(m.Server)
	if err != nil {
		log.Fatal(err)
	}

	m.db = session.DB(m.Database)
}

func (m *Mongo) Add(batch api.TransactionWithSignature) {

	err := m.db.C("batches").Insert(&batch)
	if err != nil {
		panic(err)
	}
}

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

func (m *Mongo) GetLogs(batchIds []string) (logs []LogStruct) {

	err := m.db.C("logs").Find(bson.M{"batchid": bson.M{"$in": batchIds}}).All(&logs)
	if err != nil {
		panic(err)
	}

	return
}

func (m *Mongo) RetrieveTransactionHistory(from, to, batchId, TxID string) (txs []api.TransactionWithSignature) {

	var err error

	if TxID != "" {

		err = m.db.C("batches").Find(bson.M{"id": TxID}).All(&txs)

	} else {

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
			} else {
				err = m.db.C("batches").Find(bson.M{"batchid": batchId}).All(&txs)
			}

		}

	}

	if err != nil {
		panic(err)
	}

	return
}

func (m *Mongo) RetrieveTransactionByBatchIds(batchIds []string) (txs []api.TransactionWithSignature) {

	var err error

	err = m.db.C("batches").Find(bson.M{"batchid": bson.M{"$in": batchIds}}).All(&txs)

	if err != nil {
		panic(err)
	}

	return
}
