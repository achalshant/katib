package main

import (
	"context"
	"log"
	"net"

	_ "github.com/go-sql-driver/mysql"
	dbif "github.com/kubeflow/katib/pkg/db"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	port = "0.0.0.0:6789"
)

type DBIFServer struct {
}

func (d *DBIFServer) SayHello(ctx context.Context, in *dbif.HelloRequest) (*dbif.HelloReply, error) {
	log.Printf("Received: %v", in.Name)
	return &dbif.HelloReply{Message: "Hello " + in.Name}, nil
}

func main() {
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	size := 1<<31 - 1
	s := grpc.NewServer(grpc.MaxRecvMsgSize(size), grpc.MaxSendMsgSize(size))
	dbif.RegisterDBIFServer(s, &DBIFServer{})
	reflection.Register(s)
	log.Printf("DBIF Service\n")
	if err = s.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
