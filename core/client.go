package core

import (
	"context"
	"sync"
	"time"

	"github.com/projecteru2/core/client"
	pb "github.com/projecteru2/core/rpc/gen"
	coretypes "github.com/projecteru2/core/types"

	"github.com/projecteru2/core/log"
)

// Client use core to store meta
type Client struct {
	clientPool *client.Pool
}

var coreClient *Client
var once sync.Once

// New new a Store
func New(ctx context.Context) (*Client, error) {
	auth := &coretypes.AuthConfig{
		Username: defaultEruUsrName,
		Password: defaultEruPassword,
	}
	clientPoolConfig := &client.PoolConfig{
		EruAddrs:          []string{defaultMainEruCoreEndPoint},
		Auth:              *auth,
		ConnectionTimeout: 2 * time.Minute,
	}
	clientPool, err := client.NewCoreRPCClientPool(ctx, clientPoolConfig)
	if err != nil {
		return nil, err
	}
	return &Client{clientPool}, nil
}

// GetClient returns a gRPC client
func (c *Client) GetClient() pb.CoreRPCClient {
	return c.clientPool.GetClient()
}

// Init inits the core store only once
func Init(ctx context.Context) (err error) {
	once.Do(func() {
		coreClient, err = New(ctx)
		if err != nil {
			log.WithFunc("core.client").Error(ctx, err, "failed to create core store")
			return
		}
	})
	return
}

// Get returns the core store instance
func Get() *Client {
	return coreClient
}
