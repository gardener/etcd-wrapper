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

package brclient

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type InitStatus int

const (
	New InitStatus = iota
	InProgress
	Successful
)

const DefaultEtcdConfigFilePath = "/etc/etcd.conf.yaml" //"/Users/I544000/go/src/github.tools.sap/I062009/etcd-bootstrapper/etcd-config-test.yaml"

//go:generate stringer -type=InitStatus

// ValidationType represents the type of validation that should be done of etcd DB during initialisation.
type ValidationType string

const (
	// SanityValidation only does sanity validation of the etcd DB.
	SanityValidation ValidationType = "sanity" // validation_sanity
	// FullValidation does a complete validation of the etcd DB.
	FullValidation ValidationType = "full" // validation_full
)

// BackupRestoreClient is a client to connect to the backup-restore HTTPs server.
type BackupRestoreClient interface {
	// GetInitializationStatus gets the latest state of initialization from the backup-restore.
	GetInitializationStatus() (InitStatus, error)
	// TriggerInitialization triggers the initialization on the backup-restore passing in the ValidationType.
	TriggerInitialization(validationType ValidationType) error
	// GetEtcdConfig gets the etcd configuration from the backup-restore, stores it into a file and returns the path to the file.
	GetEtcdConfig() (string, error)
}

type brClient struct {
	client             *http.Client
	sidecarBaseAddress string
	etcdConfigFilePath string
}

// NewTestClient returns a BackupRestoreClient object with a configurable http.Client value
// To be used for unit tests
func NewTestClient(testClient *http.Client, sidecarBaseAddress string, etcdConfigFilePath string) (BackupRestoreClient, error) {
	return &brClient{client: testClient,
		sidecarBaseAddress: sidecarBaseAddress,
		etcdConfigFilePath: etcdConfigFilePath}, nil
}

func NewClient(tlsEnabled bool, caCertBundlePath *string, sidecarBaseAddress string, etcdConfigFilePath string) (BackupRestoreClient, error) {
	var (
		tlsConfig *tls.Config
		err       error
	)

	if tlsEnabled {
		if tlsConfig, err = createTLSConfig(*caCertBundlePath); err != nil {
			return nil, err
		}
	} else {
		tlsConfig = createInsecureTLSConfig()
	}
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
		//MaxIdleConns:        0,
		//MaxIdleConnsPerHost: 0,
		//MaxConnsPerHost: 0,
		//IdleConnTimeout: 0,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second, //Change as and when needed
	}
	return &brClient{client: client,
		sidecarBaseAddress: sidecarBaseAddress,
		etcdConfigFilePath: etcdConfigFilePath}, nil
}

func (c *brClient) GetInitializationStatus() (InitStatus, error) {
	response, err := c.client.Get(c.sidecarBaseAddress + "/initialization/status")
	if err != nil {
		return New, err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	bodyBytes, _ := io.ReadAll(response.Body)
	initializationStatus := string(bodyBytes)

	if initializationStatus == New.String() {
		return New, nil
	} else if initializationStatus == Successful.String() {
		return Successful, nil
	} else {
		return InProgress, nil
	}
}

func (c *brClient) TriggerInitialization(validationType ValidationType) error {
	_, err := c.client.Get(c.sidecarBaseAddress + fmt.Sprintf("/initialization/start?mode=%s", validationType))
	return err
}

func (c *brClient) GetEtcdConfig() (string, error) {
	response, err := c.client.Get(c.sidecarBaseAddress + "/config")
	if err != nil {
		return "", err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	etcdConfig := string(bodyBytes)
	err = os.WriteFile(c.etcdConfigFilePath, []byte(etcdConfig), 0777)
	if err != nil {
		return "", err
	}
	return c.etcdConfigFilePath, nil
}

func createTLSConfig(caCertBundlePath string) (*tls.Config, error) {
	caCertBundle, err := os.ReadFile(caCertBundlePath)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCertBundle)

	return &tls.Config{
		RootCAs: caCertPool,
	}, nil
}

func createInsecureTLSConfig() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: true,
	}
}
