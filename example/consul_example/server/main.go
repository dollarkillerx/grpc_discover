package main

import (
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/dollarkillerx/grpc_discover"
	"github.com/dollarkillerx/grpc_discover/example/proto"
	consulapi "github.com/hashicorp/consul/api"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func main() {
	lis, err := net.Listen("tcp", "127.0.0.1:8372")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	config := consulapi.DefaultConfig()
	config.Address = "127.0.0.1:8500"
	plugin, err := grpc_discover.NewConsulPlugin(config)
	if err != nil {
		panic(err)
	}

	// heartbeat
	go func() {
		http.HandleFunc("/heartbeat", func(writer http.ResponseWriter, request *http.Request) {
			writer.Write([]byte("ack"))
		})

		err = http.ListenAndServe("0.0.0.0:5030", nil)
		if err != nil {
			fmt.Println("error: ", err.Error())
		}
	}()

	// 注册服务 registration service
	serverID, err := plugin.Register("GreeterServer", lis.Addr().String(), "http://192.168.31.65:5030/heartbeat")
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
