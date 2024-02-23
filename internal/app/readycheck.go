// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gardener/etcd-wrapper/internal/bootstrap"
	"github.com/gardener/etcd-wrapper/internal/types"
	"github.com/gardener/etcd-wrapper/internal/util"

	"go.etcd.io/etcd/clientv3"
	"go.uber.org/zap"
)

const (
	// ReadyServerPort is the port number for the server that serves the readiness probe
	ReadyServerPort       = int64(9095)
	etcdConnectionTimeout = 5 * time.Second
	etcdGetTimeout        = 5 * time.Second
	etcdQueryInterval     = 2 * time.Second
	etcdEndpointPort      = "2379"
)

// queryAndUpdateEtcdReadiness periodically queries the etcd DB to check its readiness and updates the status
// of the query into the etcdStatus struct. It stops querying when the application context is cancelled.
func (a *Application) queryAndUpdateEtcdReadiness() {
	// Create a ticker to periodically query etcd readiness
	ticker := time.NewTicker(etcdQueryInterval)
	defer ticker.Stop()

	for {
		// Query etcd readiness and update the status
		a.etcdReady = a.isEtcdReady()
		select {
		// Stop querying and return when the context is cancelled
		case <-a.ctx.Done():
			a.logger.Error("stopped periodic DB query: context cancelled", zap.Error(a.ctx.Err()))
			return
		// Wait for the next tick before querying again
		case <-ticker.C:
		}
	}
}

// isEtcdReady checks if ETCD is ready by making a `GET` call (with a timeout).
// if there is an error then it returns false else it returns true.
func (a *Application) isEtcdReady() bool {
	etcdConnCtx, cancelFunc := context.WithTimeout(a.ctx, etcdGetTimeout)
	defer cancelFunc()
	_, err := a.etcdClient.Get(etcdConnCtx, "foo")
	if err != nil {
		a.logger.Error("failed to retrieve from etcd db", zap.Error(err))
	}
	return err == nil
}

// readinessHandler reads the etcd status from the etcdStatus struct and writes that onto the http responsewriter
func (a *Application) readinessHandler(w http.ResponseWriter, _ *http.Request) {
	if a.etcdReady {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusServiceUnavailable)
}

// createEtcdClient creates an ETCD client
func (a *Application) createEtcdClient() (*clientv3.Client, error) {
	// fetch tls configuration
	tlsConfig, err := util.CreateTLSConfig(a.isTLSEnabled, a.Config.EtcdClientTLS.ServerName, a.cfg.ClientTLSInfo.TrustedCAFile, &util.KeyPair{
		CertPath: a.Config.EtcdClientTLS.CertPath,
		KeyPath:  a.Config.EtcdClientTLS.KeyPath,
	})
	if err != nil {
		return nil, err
	}

	// Create etcd client
	cli, err := clientv3.New(clientv3.Config{
		Context:     a.ctx,
		Endpoints:   []string{util.ConstructBaseAddress(a.isTLSEnabled(), fmt.Sprintf("%s:%s", a.Config.EtcdClientTLS.ServerName, etcdEndpointPort))},
		DialTimeout: etcdConnectionTimeout,
		LogConfig:   bootstrap.SetupLoggerConfig(types.DefaultLogLevel),
		TLS:         tlsConfig,
	})
	if err != nil {
		return nil, err
	}
	return cli, nil
}

// isTLSEnabled checks if TLS has been enabled in the etcd configuration.
func (a *Application) isTLSEnabled() bool {
	return len(strings.TrimSpace(a.cfg.ClientTLSInfo.CertFile)) != 0 &&
		len(strings.TrimSpace(a.cfg.ClientTLSInfo.KeyFile)) != 0 &&
		len(strings.TrimSpace(a.cfg.ClientTLSInfo.TrustedCAFile)) != 0
}
