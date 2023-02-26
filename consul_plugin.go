package grpc_discover

import (
	"fmt"
	"log"
	"net"
	"strconv"

	consulapi "github.com/hashicorp/consul/api"
	"google.golang.org/grpc/resolver"
)

type ConsulPlugin struct {
	client *consulapi.Client
}

func NewConsulPlugin(config *consulapi.Config) (*ConsulPlugin, error) {
	client, err := consulapi.NewClient(config)

	return &ConsulPlugin{client: client}, err
}

func (c *ConsulPlugin) Register(serverName string, address string, checkAddress string) (serverID string, err error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return "", err
	}
	iport, err := strconv.Atoi(port)
	if err != nil {
		return "", err
	}

	// 创建注册到consul的服务到
	registration := new(consulapi.AgentServiceRegistration)
	registration.ID = getServerID(serverName)
	registration.Name = serverName
	registration.Port = iport
	//registration.Tags = tags
	registration.Address = host

	// 增加consul健康检查回调函数
	check := new(consulapi.AgentServiceCheck)
	check.HTTP = checkAddress
	check.Timeout = "5s"
	check.Interval = "5s"
	check.DeregisterCriticalServiceAfter = "10s" // 故障检查失败10s后 consul自动将注册服务删除
	registration.Check = check

	// 注册服务到consul
	err = c.client.Agent().ServiceRegister(registration)
	return registration.ID, err
}

func (c *ConsulPlugin) UnRegister(serverID string) error {
	return c.client.Agent().ServiceDeregister(serverID)
}

func (c *ConsulPlugin) AutoUnRegister(serverID string) {
	Signal(func() {
		c.UnRegister(serverID)
	})
}

func (c *ConsulPlugin) DiscoverByServerName(serverName string) ([]string, error) {
	//只获取健康的service
	serviceHealthy, _, err := c.client.Health().Service(serverName, "", true, nil)
	if err != nil {
		return nil, err
	}

	if len(serviceHealthy) == 0 {
		return nil, ErrServiceNotFound
	}

	var srvAddress []string

	for _, v := range serviceHealthy {
		srvAddress = append(srvAddress, fmt.Sprintf("%s:%d", v.Service.Address, v.Service.Port))
	}

	return srvAddress, nil
}

func (c *ConsulPlugin) DiscoverByServerID(serverID string) (string, error) {
	serviceHealthy, _, err := c.client.Health().Service(getServerNameByIDConsulVersion(serverID), "", true, nil)
	if err != nil {
		return "", err
	}

	if len(serviceHealthy) == 0 {
		return "", ErrServiceNotFound
	}

	for _, v := range serviceHealthy {
		if v.Service.ID == serverID {
			return fmt.Sprintf("%s:%d", v.Service.Address, v.Service.Port), nil
		}
	}

	return "", ErrServiceNotFound
}

func (c *ConsulPlugin) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	rc := &consulResolver{
		target: target,
		cc:     cc,
		opts:   opts,
		c:      c,
	}

	rc.ResolveNow(resolver.ResolveNowOptions{})
	return rc, nil
}

func (c *ConsulPlugin) Scheme() string {
	return "consul"
}

type consulResolver struct {
	target resolver.Target
	cc     resolver.ClientConn
	opts   resolver.BuildOptions
	c      *ConsulPlugin
}

func (e *consulResolver) ResolveNow(options resolver.ResolveNowOptions) {
	//只获取健康的service
	serviceHealthy, _, err := e.c.client.Health().Service(e.target.Endpoint(), "", true, nil)
	if err != nil {
		log.Printf("[GRPC Discover][Consul Pugin] ResolveNow %s:///%s Error: %s \n", e.target.Scheme, e.target.Endpoint, err)
		return
	}

	if len(serviceHealthy) == 0 {
		log.Printf("[GRPC Discover][Consul Pugin] ResolveNow %s:///%s Error: %s \n", e.target.Scheme, e.target.Endpoint, "could not find service")
		return
	}

	var srvAddress []resolver.Address

	for _, v := range serviceHealthy {
		srvAddress = append(srvAddress, resolver.Address{
			Addr: fmt.Sprintf("%s:%d", v.Service.Address, v.Service.Port),
		})
	}

	err = e.cc.UpdateState(resolver.State{Addresses: srvAddress})
	if err != nil {
		log.Printf("[GRPC Discover][Consul Pugin] ResolveNow %s:///%s Error: %s \n", e.target.Scheme, e.target.Endpoint, err)
	}
}

func (e *consulResolver) Close() {}
