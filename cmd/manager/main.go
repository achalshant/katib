package main

import (
	"context"
	"log"
	"time"

	kdb "github.com/kubeflow/katib/pkg/db"

	"google.golang.org/grpc"
)

const (
	address     = "0.0.0.0:6789"
	defaultName = "world"
)

type server struct {
}

/*
func (s *server) CreateStudy(ctx context.Context, in *kdb.CreateStudyRequest) (*kdb.CreateStudyReply, error) {
	if in == nil || in.StudyConfig == nil {
		return &kdb.CreateStudyReply{}, errors.New("StudyConfig is missing.")
	}

	studyID, err := dbIf.CreateStudy(in.StudyConfig)
	if err != nil {
		return &kdb.CreateStudyReply{}, err
	}

	return &kdb.CreateStudyReply{StudyId: studyID}, nil
}
*/
func main() {
	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := kdb.NewDBIFClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	r, err := c.SayHello(ctx, &kdb.HelloRequest{Name: defaultName})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", r.Message)
	// csresp, err := c.CreateStudy(ctx, &kdb.CreateStudyRequest{StudyConfig: &kdb.StudyConfig{Name: "NewStudy"}})
	// if err != nil {
	// 	log.Fatalf("could not create study: %v", err)
	// }
	// log.Printf("Study created with id: %s", csresp.StudyId)
}
