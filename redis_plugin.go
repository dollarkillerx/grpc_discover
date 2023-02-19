package grpc_discover

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/resolver"
)

type RedisPlugin struct {
	client *redis.Client

	mu    sync.Mutex
	close map[string]chan struct{}
}

func NewRedisPlugin(config *redis.Options) (*RedisPlugin, error) {
	client := redis.NewClient(config)
	err := client.Ping(context.TODO()).Err()
	if err != nil {
		return nil, err
	}

	return &RedisPlugin{client: client, close: map[string]chan struct{}{}}, err
}

func (r *RedisPlugin) Register(serverName string, address string) (serverID string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	serverID = getServerID(serverName)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err = r.client.Set(ctx, serverID, address, time.Second*10).Err()
	if err != nil {
		return "", err
	}

	closeCh := make(chan struct{})
	r.close[serverID] = closeCh

	go r.keepAlive(closeCh, serverID, address)

	log.Printf("[GRPC Discover][Redis Pugin] Register ServerName: %s ServerAddress: %s ServerID: %s \n", serverName, address, serverID)

	return serverID, nil
}

func (r *RedisPlugin) keepAlive(closeCh chan struct{}, serverID string, address string) {
	ticker := time.NewTicker(time.Second * 3)
loop:
	for {
		select {
		case <-closeCh:
			break loop
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			err := r.client.Set(ctx, serverID, address, time.Second*10).Err()
			if err != nil {
				log.Printf("[GRPC Discover][Redis Pugin] keepAlive Error  %s \n", err)
			}
		}
	}
}

func (r *RedisPlugin) UnRegister(serverID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	c, ex := r.close[serverID]
	if !ex {
		return errors.New("service does not exist")
	}

	close(c)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := r.client.Del(ctx, serverID).Err()
	return err
}

func (r *RedisPlugin) AutoUnRegister(serverID string) {
	Signal(func() {
		r.UnRegister(serverID)
	})
}

func (r *RedisPlugin) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	rc := &redisResolver{
		target: target,
		cc:     cc,
		opts:   opts,
		r:      r,
	}

	rc.ResolveNow(resolver.ResolveNowOptions{})
	return rc, nil
}

func (r *RedisPlugin) Scheme() string {
	return "redis"
}

type redisResolver struct {
	target resolver.Target
	cc     resolver.ClientConn
	opts   resolver.BuildOptions
	r      *RedisPlugin
}

func (e *redisResolver) ResolveNow(options resolver.ResolveNowOptions) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := e.r.client.Keys(ctx, getServerIDPrefix(e.target.Endpoint)+"*").Result()
	if err != nil {
		log.Printf("[GRPC Discover][Redis Pugin] ResolveNow %s:///%s Error: %s \n", e.target.Scheme, e.target.Endpoint, err)
		return
	}

	if len(result) == 0 {
		log.Printf("[GRPC Discover][Redis Pugin] ResolveNow %s:///%s Error: %s \n", e.target.Scheme, e.target.Endpoint, "could not find service")
		return
	}

	var srvAddress []resolver.Address

	for _, v := range result {
		val, err := e.r.client.Get(context.TODO(), v).Result()
		if err != nil {
			log.Printf("[GRPC Discover][Redis Pugin] ResolveNow %s:///%s Error: %s \n", e.target.Scheme, e.target.Endpoint, err)
			continue
		}
		srvAddress = append(srvAddress, resolver.Address{
			Addr: val,
		})
	}

	err = e.cc.UpdateState(resolver.State{Addresses: srvAddress})
	if err != nil {
		log.Printf("[GRPC Discover][Redis Pugin] ResolveNow %s:///%s Error: %s \n", e.target.Scheme, e.target.Endpoint, err)
	}
}

func (e *redisResolver) Close() {}
