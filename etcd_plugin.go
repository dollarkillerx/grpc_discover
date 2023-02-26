package grpc_discover

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/pkg/errors"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/resolver"
)

type ETCDPlugin struct {
	client  *clientv3.Client
	kv      clientv3.KV
	watcher clientv3.Watcher
	lease   clientv3.Lease

	mu      sync.Mutex
	mapping map[string]clientv3.LeaseID
}

// NewETCDPlugin 初始化 etcd 插件，Initialize etcd plugin
func NewETCDPlugin(config clientv3.Config) (*ETCDPlugin, error) {
	client, err := clientv3.New(config)
	if err != nil {
		return nil, err
	}

	kv := clientv3.NewKV(client)
	watcher := clientv3.NewWatcher(client)
	lease := clientv3.NewLease(client)
	return &ETCDPlugin{
		client:  client,
		kv:      kv,
		watcher: watcher,
		lease:   lease,
		mapping: map[string]clientv3.LeaseID{},
	}, nil
}

// Register 服务注册
func (e *ETCDPlugin) Register(serverName string, address string) (serverID string, err error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	serverID = getServerID(serverName)

	leaseID, err := e.lease.Grant(context.TODO(), 10)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err = e.kv.Put(ctx, serverID, address, clientv3.WithLease(leaseID.ID))
	if err != nil {
		return "", err
	}

	ch, err := e.lease.KeepAlive(context.TODO(), leaseID.ID)
	if err != nil {
		return "", err
	}
	go func() {
		for {
			select {
			case <-ch:
			}
		}
	}()

	e.mapping[serverID] = leaseID.ID

	log.Printf("[GRPC Discover][ETCD Pugin] Register ServerName: %s ServerAddress: %s ServerID: %s \n", serverName, address, serverID)

	return serverID, nil
}

// UnRegister 服务反注册
func (e *ETCDPlugin) UnRegister(serverID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	id, ex := e.mapping[serverID]
	if !ex {
		return errors.New("service does not exist")
	}

	_, err := e.kv.Delete(context.TODO(), serverID)
	if err != nil {
		return err
	}

	_, err = e.lease.Revoke(context.TODO(), id)
	return err
}

// AutoUnRegister 自动退出
func (e *ETCDPlugin) AutoUnRegister(serverID string) {
	Signal(func() {
		e.UnRegister(serverID)
	})
}

func (e *ETCDPlugin) DiscoverByServerName(serverName string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	get, err := e.kv.Get(ctx, getServerIDPrefix(serverName), clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	var srvAddress []string

	for _, v := range get.Kvs {
		srvAddress = append(srvAddress, string(v.Value))
	}

	return srvAddress, nil
}

func (e *ETCDPlugin) DiscoverByServerID(serverID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	get, err := e.kv.Get(ctx, serverID)
	if err != nil {
		return "", err
	}

	if len(get.Kvs) != 1 {
		return "", ErrServiceNotFound
	}
	return string(get.Kvs[0].Value), nil
}

func (e *ETCDPlugin) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	r := &etcdResolver{
		target: target,
		cc:     cc,
		opts:   opts,
		e:      e,
	}

	r.ResolveNow(resolver.ResolveNowOptions{})
	return r, nil
}

func (e *ETCDPlugin) Scheme() string {
	return "etcd"
}

type etcdResolver struct {
	target resolver.Target
	cc     resolver.ClientConn
	opts   resolver.BuildOptions
	e      *ETCDPlugin
}

func (e *etcdResolver) ResolveNow(options resolver.ResolveNowOptions) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	get, err := e.e.kv.Get(ctx, getServerIDPrefix(e.target.Endpoint()), clientv3.WithPrefix())
	if err != nil {
		log.Printf("[GRPC Discover][ETCD Pugin] ResolveNow %s:///%s Error: %s \n", e.target.Scheme, e.target.Endpoint, err)
		return
	}

	if get.Count == 0 {
		log.Printf("[GRPC Discover][ETCD Pugin] ResolveNow %s:///%s Error: %s \n", e.target.Scheme, e.target.Endpoint, "could not find service")
		return
	}

	var srvAddress []resolver.Address

	for _, v := range get.Kvs {
		srvAddress = append(srvAddress, resolver.Address{
			Addr: string(v.Value),
		})
	}

	err = e.cc.UpdateState(resolver.State{Addresses: srvAddress})
	if err != nil {
		log.Printf("[GRPC Discover][ETCD Pugin] ResolveNow %s:///%s Error: %s \n", e.target.Scheme, e.target.Endpoint, err)
	}
}

func (e *etcdResolver) Close() {}
