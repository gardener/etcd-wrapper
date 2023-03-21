package bootstrap

import (
	"context"

	"github.com/gardener/etcd-wrapper/internal/brclient"
)

// InitializeEtcd triggers the initialization of etcd DB.
func InitializeEtcd(ctx context.Context) error {
	//create new BackupRestoreClient
	brClient := brclient.NewClient("")
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		brClient.GetInitializationStatus()
		//TODO: Aaron to complete.
	}
}
