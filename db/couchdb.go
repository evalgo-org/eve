package db

import (
	"context"
	"fmt"

	kivik "github.com/go-kivik/kivik/v4"
	_ "github.com/go-kivik/kivik/v4/couchdb" // The CouchDB driver
)

func CouchDBAnimals(url string) {
	client, err := kivik.New("couch", url)
	if err != nil {
		panic(err)
	}

	exists, _ := client.DBExists(context.Background(), "animals")
	if !exists {
		err = client.CreateDB(context.Background(), "animals")
		if err != nil {
			fmt.Println(err)
		}
	}
	db := client.DB("animals")

	doc := map[string]interface{}{
		"_id":      "cow",
		"feet":     4,
		"greeting": "moo",
	}

	rev, err := db.Put(context.TODO(), "cow", doc)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Cow inserted with revision %s\n", rev)
}

func CouchDBDocNew(url, db string, doc interface{}) (string, string) {
	client, err := kivik.New("couch", url)
	if err != nil {
		panic(err)
	}
	exists, _ := client.DBExists(context.Background(), db)
	if !exists {
		err = client.CreateDB(context.Background(), db)
		if err != nil {
			fmt.Println(err)
		}
	}
	cdb := client.DB(db)
	docId, revId, err := cdb.CreateDoc(context.TODO(), doc)
	if err != nil {
		panic(err)
	}
	return docId, revId
}

func CouchDBDocGet(url, db, docId string) *kivik.Document {
	client, err := kivik.New("couch", url)
	if err != nil {
		panic(err)
	}
	exists, _ := client.DBExists(context.Background(), db)
	if !exists {
		err = client.CreateDB(context.Background(), db)
		if err != nil {
			fmt.Println(err)
		}
	}
	cdb := client.DB(db)
	return cdb.Get(context.TODO(), docId)
}
