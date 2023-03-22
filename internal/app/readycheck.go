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
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

const (
	etcdClientTimeout = 5 * time.Second
	etcdEndpoint      = "://127.0.0.1:2379"
	ReadyServerPort   = int64(9095)
	protocolHTTP      = "http"
	protocolHTTPS     = "https"
)

func (a *Application) createEtcdClient() (*clientv3.Client, error) {
	tlsConf := &tls.Config{}
	protocol := protocolHTTP
	if len(a.cfg.ClientTLSInfo.CertFile) != 0 && len(a.cfg.ClientTLSInfo.KeyFile) != 0 && len(a.cfg.ClientTLSInfo.TrustedCAFile) != 0 {
		// Create certificate key pair
		certificate, err := tls.LoadX509KeyPair(a.cfg.ClientTLSInfo.CertFile, a.cfg.ClientTLSInfo.KeyFile)
		if err != nil {
			a.logger.Error("failed to load key pair", zap.Error(err))
		}

		// Create CA cert pool
		caCertBundle, err := os.ReadFile(a.cfg.ClientTLSInfo.TrustedCAFile)
		if err != nil {
			a.logger.Error("error reading trusted CA file", zap.Error(err))
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCertBundle)

		// Create TLS configuration
		tlsConf.RootCAs = caCertPool
		tlsConf.Certificates = []tls.Certificate{certificate}

		// Use https protocol
		protocol = protocolHTTPS
	}

	// Create etcd client
	cli, err := clientv3.New(clientv3.Config{
		//TODO: need TLS here?
		Context:     a.ctx,
		Endpoints:   []string{protocol + etcdEndpoint},
		DialTimeout: etcdClientTimeout,
		Logger:      a.logger,
		TLS:         tlsConf,
	})
	if err != nil {
		a.logger.Error("failed to create etcd client", zap.Error(err))
	}
	return cli, nil
}

func (a *Application) readinessHandler(w http.ResponseWriter, r *http.Request) {
	var healthStatus bool
	healthStatus = true

	// etcd `get` call
	etcdConnCtx, cancelFunc := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancelFunc()
	_, err := a.etcdClient.Get(etcdConnCtx, "foo", clientv3.WithSerializable())
	if err != nil {
		a.logger.Error("failed to retrieve from etcd db", zap.Error(err))
		healthStatus = false
	}

	// Return value
	jsonValue, err := json.Marshal(healthStatus)
	if err != nil {
		a.logger.Error("Unable to marshal health status to json", zap.Error(err))
		return
	}
	_, _ = w.Write(jsonValue)
}

// SetupReadinessProbe sets up the readiness probe for this application. It is a blocking function and therefore
// the consumer should always call this within a go-routine unless the caller itself wants to block on this which is unlikely.
func (a *Application) SetupReadinessProbe() {
	// If the http server errors out then you will have to panic and that should cause the container to exit and then be restarted by kubelet.
	if len(a.cfg.ClientTLSInfo.CertFile) != 0 && len(a.cfg.ClientTLSInfo.KeyFile) != 0 {
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
