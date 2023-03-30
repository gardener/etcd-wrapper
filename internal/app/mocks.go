package app

import (
	"context"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// EtcdFakeKV mocks the KV interface of etcd
// required to mock etcd get calls
type EtcdFakeKV struct{}

func (c *EtcdFakeKV) Get(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	return nil, nil
}
func (c *EtcdFakeKV) Put(ctx context.Context, key, val string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error) {
	return nil, nil
}
func (c *EtcdFakeKV) Delete(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.DeleteResponse, error) {
	return nil, nil
}
func (c *EtcdFakeKV) Compact(ctx context.Context, rev int64, opts ...clientv3.CompactOption) (*clientv3.CompactResponse, error) {
	return nil, nil
}
func (c *EtcdFakeKV) Txn(ctx context.Context) clientv3.Txn {
	return nil
}
func (c *EtcdFakeKV) Do(ctx context.Context, op clientv3.Op) (clientv3.OpResponse, error) {
	return clientv3.OpResponse{}, nil
}
