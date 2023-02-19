# Grpc Discover

### Plugin
- [X] ETCD
- [X] Consul
- [X] Redis

### example

- [etcd_example](example%2Fetcd_example)
- [consul_example](example%2Fconsul_example)
- [redis_example](example%2Fredis_example)

server use etcd plugin
``` 
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
```

client use etcd plugin
``` 
    plugin, err := grpc_discover.NewETCDPlugin(clientv3.Config{
		Endpoints:   []string{"127.0.0.1:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		panic(err)
	}
	resolver.Register(plugin)

	// Set up a connection to the server.
	conn, err := grpc.Dial("etcd:///GreeterServer", grpc.WithTransportCredentials(insecure.NewCredentials()))
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
```