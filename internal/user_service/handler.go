package user_service

import (
	"context"
	"flag"
	"fmt"
	pb "github.com/Gowtham1729/go_social_media/gen/user_service/v1"
	"github.com/Gowtham1729/go_social_media/internal/config"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
	"net/http"
)

type UserServer struct {
	pb.UnimplementedUserServiceServer
}

func (s *UserServer) SignUpUser(ctx context.Context, req *pb.SignUpUserRequest) (*pb.SignUpUserResponse, error) {
	fmt.Println(ctx)
	fmt.Println(req)
	return &pb.SignUpUserResponse{User: &pb.User{
		Id:        "123456",
		Username:  "abcdef",
		Email:     "abc@gmail.com",
		Name:      "abc def",
		CreatedAt: nil,
		UpdatedAt: nil,
	}}, nil
}

func RunUserServer() {
	flag.Parse()
	err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	cfg := config.Get()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.GrpcPort))
	if err != nil {
		fmt.Printf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterUserServiceServer(s, &UserServer{})
	reflection.Register(s)

	fmt.Printf("server listening at %v", lis.Addr())
	go func() {
		log.Fatalln(s.Serve(lis))
	}()

	conn, err := grpc.NewClient(
		fmt.Sprintf("0.0.0.0:%d", cfg.Server.GrpcPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Printf("failed to dial server: %v", err)
	}

	gwmux := runtime.NewServeMux()
	err = pb.RegisterUserServiceHandler(context.Background(), gwmux, conn)
	if err != nil {
		fmt.Printf("failed to register gateway: %v", err)
	}

	gwServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.HttpPort),
		Handler: gwmux,
	}
	log.Fatalln(gwServer.ListenAndServe())

}
