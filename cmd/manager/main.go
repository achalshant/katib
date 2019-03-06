package main

import (
	"context"
	"flag"
	"log"
	"net"
	"time"

	kdb "github.com/kubeflow/katib/pkg/db"

	"google.golang.org/grpc"
)

const (
	port = "0.0.0.0:6789"
)

func main() {
	flag.Parse()
	var err error
	conn, err := grpc.Dial("0.0.0.0:6789", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to open db connection: %v", err)
	}
	defer conn.Close()
	dbIf := kdb.DBIFClient(conn)
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	size := 1<<31 - 1

	r, err := dbIf.SayHello(ctx, &dbIf.HelloRequest{Name: name})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", r.Message)
}
