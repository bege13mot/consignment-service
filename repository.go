package main

import (
	"fmt"

	pb "github.com/bege13mot/consignment-service/proto/consignment"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	dbName                = "shippy"
	consignmentCollection = "consignments"
)

// Repository help
type Repository interface {
	Create(*pb.Consignment) error
	GetAll() ([]*pb.Consignment, error)
	Close()
}

//ConsignmentRepository help
type ConsignmentRepository struct {
	session *mgo.Session
}

type test struct {
	ID          bson.ObjectId `bson:"_id,omitempty"`
	Description string        `bson:"description"`
}

// Create a new consignment
func (repo *ConsignmentRepository) Create(consignment *pb.Consignment) error {

	repo.collection().Insert(consignment)
	cons := []test{}
	repo.collection().Find(bson.M{"id": ""}).All(&cons)

	for i, v := range cons {
		fmt.Println(i, v.ID.Hex())
		consignment.Id = v.ID.Hex()
		repo.collection().Update(bson.M{"id": ""}, bson.M{"$set": bson.M{"id": v.ID.Hex()}})
	}

	return nil
}

// GetAll consignments
func (repo *ConsignmentRepository) GetAll() ([]*pb.Consignment, error) {
	var consignments []*pb.Consignment
	// Find normally takes a query, but as we want everything, we can nil this.
	// We then bind our consignments variable by passing it as an argument to .All().
	// That sets consignments to the result of the find query.
	// There's also a `One()` function for single results.
	err := repo.collection().Find(nil).All(&consignments)
	return consignments, err
}

// Close closes the database session after each query has ran.
// Mgo creates a 'master' session on start-up, it's then good practice
// to clone a new session for each request that's made. This means that
// each request has its own database session. This is safer and more efficient,
// as under the hood each session has its own database socket and error handling.
// Using one main database socket means requests having to wait for that session.
// I.e this approach avoids locking and allows for requests to be processed concurrently. Nice!
// But... it does mean we need to ensure each session is closed on completion. Otherwise
// you'll likely build up loads of dud connections and hit a connection limit. Not nice!
func (repo *ConsignmentRepository) Close() {
	repo.session.Close()
}

func (repo *ConsignmentRepository) collection() *mgo.Collection {
	return repo.session.DB(dbName).C(consignmentCollection)
}
