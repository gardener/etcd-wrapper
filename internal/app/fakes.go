// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"

	"go.etcd.io/etcd/clientv3"
)

// EtcdFakeKV mocks the KV interface of etcd required to mock etcd get calls
type EtcdFakeKV struct{}

// Get gets a value for a given key.
func (c *EtcdFakeKV) Get(_ context.Context, _ string, _ ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	return nil, nil
}

// Put puts a value for a given key.
func (c *EtcdFakeKV) Put(_ context.Context, _, _ string, _ ...clientv3.OpOption) (*clientv3.PutResponse, error) {
	return nil, nil
}

// Delete deletes an entry with a given key.
func (c *EtcdFakeKV) Delete(_ context.Context, _ string, _ ...clientv3.OpOption) (*clientv3.DeleteResponse, error) {
	return nil, nil
}

// Compact compacts etcd KV history before the given rev.
func (c *EtcdFakeKV) Compact(_ context.Context, _ int64, _ ...clientv3.CompactOption) (*clientv3.CompactResponse, error) {
	return nil, nil
}

// Txn creates a transaction.
func (c *EtcdFakeKV) Txn(_ context.Context) clientv3.Txn {
	return nil
}

// Do applies a single Op on KV without a transaction.
func (c *EtcdFakeKV) Do(_ context.Context, _ clientv3.Op) (clientv3.OpResponse, error) {
	return clientv3.OpResponse{}, nil
}
