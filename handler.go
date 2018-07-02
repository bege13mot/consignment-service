package main

import (
	"log"

	pb "github.com/bege13mot/consignment-service/proto/consignment"
	vesselProto "github.com/bege13mot/vessel-service/proto/vessel"
	"golang.org/x/net/context"
	mgo "gopkg.in/mgo.v2"
)

// Service should implement all of the methods to satisfy the service
// we defined in our protobuf definition. You can check the interface
// in the generated code itself for the exact method signatures etc
// to give you a better idea.
type service struct {
	session *mgo.Session
	// repo         Repository
	vesselClient vesselProto.VesselServiceClient
}

func (s *service) GetRepo() Repository {
	return &ConsignmentRepository{s.session.Clone()}
}

// CreateConsignment - we created just one method on our service,
// which is a create method, which takes a context and a request as an
// argument, these are handled by the gRPC server.
func (s *service) CreateConsignment(ctx context.Context, req *pb.Consignment) (*pb.Response, error) {
	repo := s.GetRepo()
	defer repo.Close()

	// Here we call a client instance of our vessel service with our consignment weight,
	// and the amount of containers as the capacity value
	vesselResponse, err := s.vesselClient.FindAvailable(context.Background(), &vesselProto.Specification{
		MaxWeight: req.Weight,
		Capacity:  int32(len(req.Containers)),
	})
	log.Printf("Found vessel: %s, id: %s \n", vesselResponse.Vessel.Name, vesselResponse.Vessel.Id)
	if err != nil {
		return nil, err
	}

	// We set the VesselId as the vessel we got back from our
	// vessel service
	req.VesselId = vesselResponse.Vessel.Id

	// Get our consignment
	err = repo.Create(req)
	if err != nil {
		return nil, err
	}

	// Return matching the `Response` message we created in our
	// protobuf definition.
	return &pb.Response{Created: true, Consignment: req}, nil
}

func (s *service) GetConsignments(ctx context.Context, req *pb.GetRequest) (*pb.Response, error) {
	defer s.GetRepo().Close()

	consignments, err := s.GetRepo().GetAll()
	if err != nil {
		return nil, err
	}
	return &pb.Response{Consignments: consignments}, nil
}
