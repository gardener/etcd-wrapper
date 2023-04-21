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
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/gardener/etcd-wrapper/internal/util"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

const (
	// ReadyServerPort is the port number for the server that serves the readiness probe
	ReadyServerPort       = int64(9095)
	etcdConnectionTimeout = 5 * time.Second
	etcdGetTimeout        = 5 * time.Second
	etcdQueryInterval     = 2 * time.Second
	etcdEndpointAddress   = ":2379"
)

//var (
//	etcdReady bool
//)

// Create a struct which will hold the last status for etcd.
// In SetupReadinessProbe first call go a.queryAndUpdateEtcdReadiness()/queryAndUpdateEtcdReadiness this function will
// periodically query etcd and updates the readiness struct. The handler just reads from the struct.
// If the struct has no status it assumes NotAvailable else it returns the current status.

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
	_, err := a.etcdClient.Get(etcdConnCtx, "foo", clientv3.WithSerializable())
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
	tlsConf, err := a.getTLSConfig()
	if err != nil {
		return nil, err
	}

	// Create etcd client
	cli, err := clientv3.New(clientv3.Config{
		Context:     a.ctx,
		Endpoints:   []string{util.ConstructBaseAddress(a.isTLSEnabled(), etcdEndpointAddress)},
		DialTimeout: etcdConnectionTimeout,
		Logger:      a.logger,
		TLS:         tlsConf,
	})
	if err != nil {
		return nil, err
	}
	return cli, nil
}

// getTLSConfig creates a TLS config if TLS has been enabled.
func (a *Application) getTLSConfig() (*tls.Config, error) {
	tlsConf := &tls.Config{}
	if a.isTLSEnabled() {
		// Create certificate key pair
		certificate, err := tls.LoadX509KeyPair(a.cfg.ClientTLSInfo.CertFile, a.cfg.ClientTLSInfo.KeyFile)
		if err != nil {
			return nil, err
		}

		// Create CA cert pool
		caCertPool, err := util.CreateCACertPool(a.cfg.ClientTLSInfo.TrustedCAFile)
		if err != nil {
			return nil, err
		}

		// Create TLS configuration
		tlsConf.RootCAs = caCertPool
		tlsConf.Certificates = []tls.Certificate{certificate}
	}
	return tlsConf, nil
}

// isTLSEnabled checks if TLS has been enabled in the etcd configuration.
func (a *Application) isTLSEnabled() bool {
	// TODO: make sure we don't have nil pointer dereference
	return len(a.cfg.ClientTLSInfo.CertFile) != 0 && len(a.cfg.ClientTLSInfo.KeyFile) != 0 && len(a.cfg.ClientTLSInfo.TrustedCAFile) != 0
}
