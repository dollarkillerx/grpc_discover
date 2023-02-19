package main

import (
	"fmt"
	"log"
	"net"

	"github.com/dollarkillerx/grpc_discover"
	"github.com/dollarkillerx/grpc_discover/example/proto"
	"github.com/redis/go-redis/v9"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func main() {
	lis, err := net.Listen("tcp", "127.0.0.1:8372")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	plugin, err := grpc_discover.NewRedisPlugin(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "root",
	})
	if err != nil {
		panic(err)
	}

	// 注册服务 registration service
	serverID, err := plugin.Register("GreeterServer", lis.Addr().String())
	if err != nil {
		panic(err)
	}
	plugin.AutoUnRegister(serverID) // 自动反注册 Automatic anti-registration

	s := grpc.NewServer()
	proto.RegisterGreeterServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}

type server struct {
	proto.UnimplementedGreeterServer
}

func (s *server) SayHello(ctx context.Context, reply *proto.HelloReply) (*proto.HelloRequest, error) {
	fmt.Println(reply.Message)

	return &proto.HelloRequest{Name: "jxc"}, nil
}
