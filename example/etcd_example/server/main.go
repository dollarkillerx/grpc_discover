package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/dollarkillerx/grpc_discover"
	"github.com/dollarkillerx/grpc_discover/example/proto"
	clientv3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func main() {
	lis, err := net.Listen("tcp", "127.0.0.1:8372")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	plugin, err := grpc_discover.NewETCDPlugin(clientv3.Config{
		Endpoints:   []string{"127.0.0.1:2379"},
		DialTimeout: 5 * time.Second,
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
