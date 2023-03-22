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
	"github.com/gardener/etcd-wrapper/internal/util"
	"net/http"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

const (
	// ReadyServerPort is the port number for the server that serves the readiness probe
	ReadyServerPort     = int64(9095)
	etcdClientTimeout   = 5 * time.Second
	etcdEndpointAddress = "://:2379"
	protocolHTTP        = "http"
	protocolHTTPS       = "https"
)

// SetupReadinessProbe sets up the readiness probe for this application. It is a blocking function and therefore
// the consumer should always call this within a go-routine unless the caller itself wants to block on this which is unlikely.
func (a *Application) SetupReadinessProbe() {
	// If the http server errors out then you will have to panic and that should cause the container to exit and then be restarted by kubelet.
	if a.isTLSEnabled() {
		http.Handle("/readyz", http.HandlerFunc(a.readinessHandler))
		err := http.ListenAndServeTLS(fmt.Sprintf(":%d", ReadyServerPort), a.cfg.ClientTLSInfo.CertFile, a.cfg.ClientTLSInfo.KeyFile, nil)
		if err != nil {
			a.logger.Error("error creating https listener", zap.Error(err))
		}
	} else {
		http.Handle("/readyz", http.HandlerFunc(a.readinessHandler))
		err := http.ListenAndServe(fmt.Sprintf(":%d", ReadyServerPort), nil)
		if err != nil {
			a.logger.Error("error creating http listener", zap.Error(err))
		}
	}
}

func (a *Application) readinessHandler(w http.ResponseWriter, r *http.Request) {
	// etcd `get` call
	etcdConnCtx, cancelFunc := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancelFunc()
	_, err := a.etcdClient.Get(etcdConnCtx, "foo", clientv3.WithSerializable())
	if err != nil {
		a.logger.Error("failed to retrieve from etcd db", zap.Error(err))
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (a *Application) createEtcdClient() (*clientv3.Client, error) {
	protocol := protocolHTTP

	// fetch tls configuration
	tlsConf, err := a.getTLSConfig()
	if err != nil {
		return nil, err
	}

	// use https protocol if tls is enabled
	if a.isTLSEnabled() {
		protocol = protocolHTTPS
	}

	// Create etcd client
	cli, err := clientv3.New(clientv3.Config{
		//TODO: need TLS here?
		Context:     a.ctx,
		Endpoints:   []string{protocol + etcdEndpointAddress},
		DialTimeout: etcdClientTimeout,
		Logger:      a.logger,
		TLS:         tlsConf,
	})
	if err != nil {
		return nil, err
	}
	return cli, nil
}

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

func (a *Application) isTLSEnabled() bool {
	return len(a.cfg.ClientTLSInfo.CertFile) != 0 && len(a.cfg.ClientTLSInfo.KeyFile) != 0 && len(a.cfg.ClientTLSInfo.TrustedCAFile) != 0
}
