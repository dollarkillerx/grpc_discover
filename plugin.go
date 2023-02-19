package grpc_discover

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/rs/xid"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func Signal(call func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		select {
		case sig := <-c:
			switch sig {
			case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT: // 监听程序退出信号
				call()
				os.Exit(0)
				return
			case syscall.SIGHUP:
				os.Exit(0)
				return
			default:
				os.Exit(0)
				return
			}
		}
	}()
}

func getServerID(serverName string) string {
	return fmt.Sprintf("grpc-discover-%s-%s", serverName, xid.New().String())
}

func getServerIDPrefix(serverName string) string {
	return fmt.Sprintf("grpc-discover-%s", serverName)
}

func getServerNameByIDConsulVersion(serverID string) string {
	split := strings.Split(serverID, "-")
	return split[2]
}

// errors
var (
	ErrServiceNotFound = errors.New("service not found")
)
