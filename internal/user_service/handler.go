package user_service

import (
	"context"
	"flag"
	"fmt"
	pb "github.com/Gowtham1729/go_social_media/gen/user_service/v1"
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
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 8080))
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
		"0.0.0.0:8080", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Printf("failed to dial server: %v", err)
	}

	gwmux := runtime.NewServeMux()
	err = pb.RegisterUserServiceHandler(context.Background(), gwmux, conn)
	if err != nil {
		fmt.Printf("failed to register gateway: %v", err)
	}

	gwServer := &http.Server{
		Addr:    ":8090",
		Handler: gwmux,
	}
	log.Fatalln(gwServer.ListenAndServe())

}
