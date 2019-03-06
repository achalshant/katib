package main

import (
	"context"
	"log"
	"net"

	_ "github.com/go-sql-driver/mysql"
	dbif "github.com/kubeflow/katib/pkg/db"
	"google.golang.org/grpc"
)

const (
	port = "0.0.0.0:6789"
)

type dbserver struct{}

// SayHello implements helloworld.GreeterServer
func (s *dbserver) SayHello(ctx context.Context, in *dbif.HelloRequest) (*dbif.HelloReply, error) {
	log.Printf("Received: %v", in.Name)
	return &dbif.HelloReply{Message: "Hello " + in.Name}, nil
}

func main() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	dbif.RegisterDBIFServer(s, &dbserver{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
