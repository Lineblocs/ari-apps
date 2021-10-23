package grpc;

import (
	"fmt"
	"log"
	"net"
	grpc_engine "google.golang.org/grpc"
)

func StartListener() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 9000))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	fmt.Println("GRPC is running!!");
	s := Server{}

	grpcServer := grpc_engine.NewServer()

	RegisterLineblocsServer(grpcServer, &s)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %s", err)
	}
}