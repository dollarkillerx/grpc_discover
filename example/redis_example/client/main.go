package main

import (
	"context"
	"log"
	"time"

	"github.com/dollarkillerx/grpc_discover"
	"github.com/dollarkillerx/grpc_discover/example/proto"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	plugin, err := grpc_discover.NewRedisPlugin(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "root",
	})
	if err != nil {
		panic(err)
	}
	resolver.Register(plugin)

	// Set up a connection to the server.
	conn, err := grpc.Dial("redis:///GreeterServer", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := proto.NewGreeterClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.SayHello(ctx, &proto.HelloReply{Message: "cpxsd"})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", r.GetName())
}
