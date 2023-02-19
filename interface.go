package grpc_discover

import "google.golang.org/grpc/resolver"

type GrpcDiscoverPluginInterface interface {
	Register(serverName string, address string) (serverID string, err error)
	UnRegister(serverID string) error
	AutoUnRegister(serverID string)

	Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error)
	Scheme() string
}
