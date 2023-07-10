// Copyright 2023 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

// SetupReadinessProbe sets up the readiness probe for this application. It is a blocking function and therefore
// the consumer should always call this within a go-routine unless the caller itself wants to block on this which is unlikely.
func (a *Application) SetupReadinessProbe() {
	// Start go routine to regularly query etcd to update its readiness status.
	go a.queryAndUpdateEtcdReadiness()
	// If the http server errors out then you will have to panic and that should cause the container to exit and then be restarted by kubelet.
	if a.isTLSEnabled() {
		http.Handle("/readyz", http.HandlerFunc(a.readinessHandler))
		err := http.ListenAndServeTLS(fmt.Sprintf(":%d", ReadyServerPort), a.cfg.ClientTLSInfo.CertFile, a.cfg.ClientTLSInfo.KeyFile, nil)
		if err != nil {
			a.logger.Fatal("failed to start TLS readiness endpoint", zap.Error(err))
		}
	} else {
		http.Handle("/readyz", http.HandlerFunc(a.readinessHandler))
		err := http.ListenAndServe(fmt.Sprintf(":%d", ReadyServerPort), nil)
		if err != nil {
			a.logger.Fatal("failed to start readiness endpoint", zap.Error(err))
		}
	}
}

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
	_, err := a.etcdClient.Get(etcdConnCtx, "foo" /*, clientv3.WithSerializable()*/)
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
