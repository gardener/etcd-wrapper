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
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gardener/etcd-wrapper/internal/types"
	"github.com/gardener/etcd-wrapper/internal/util"
)

// InitStatus is the status of initialisation as returned from backup-restore.
type InitStatus int

const (
	// Unknown indicates that the initialisation by backup-restore is unknown.
	Unknown InitStatus = iota
	// New indicates that the initialisation by backup-restore is new and has not started yet.
	New
	// InProgress indicates that the initialisation by backup-restore is in-progress.
	InProgress
	// Successful indicates that the initialisation by backup-restore is successful.
	Successful
)

// DefaultEtcdConfigFilePath defines the default file path for the etcd configuration file
const DefaultEtcdConfigFilePath = "/etc/etcd.conf.yaml"

//go:generate stringer -type=InitStatus

// ValidationType represents the type of validation that should be done of etcd DB during initialisation.
type ValidationType string

const (
	// SanityValidation only does sanity validation of the etcd DB.
	SanityValidation ValidationType = "sanity" // validation_sanity
	// FullValidation does a complete validation of the etcd DB.
	FullValidation ValidationType = "full" // validation_full
	// httpClientRequestTimeout is the timeout for all requests made by the http client
	httpClientRequestTimeout = 10 * time.Second
)

// BackupRestoreClient is a client to connect to the backup-restore HTTPs server.
type BackupRestoreClient interface {
	// GetInitializationStatus gets the latest state of initialization from the backup-restore.
	GetInitializationStatus(ctx context.Context) (InitStatus, error)
	// TriggerInitialization triggers the initialization on the backup-restore passing in the ValidationType.
	TriggerInitialization(ctx context.Context, validationType ValidationType) error
	// GetEtcdConfig gets the etcd configuration from the backup-restore, stores it into a file and returns the path to the file.
	GetEtcdConfig(ctx context.Context) (string, error)
}

// brClient implements BackupRestoreClient interface.
type brClient struct {
	client                   *http.Client
	backupRestoreBaseAddress string
	etcdConfigFilePath       string
}

// NewDefaultClient creates a BackupRestoreClient using the BackupRestoreConfig and etcd configuration at etcdConfigPath.
// It delegates the responsibility to NewClient by passing in a default implementation of HttpClientCreator.
func NewDefaultClient(brConfig types.BackupRestoreConfig, etcdConfigPath string) (BackupRestoreClient, error) {
	client, err := createClient(brConfig)
	if err != nil {
		return nil, err
	}
	return NewClient(client, brConfig.GetBaseAddress(), etcdConfigPath), nil
}

// NewClient creates and returns a new BackupRestoreClient object
func NewClient(httpClient *http.Client, backupRestoreBaseAddress string, etcdConfigFilePath string) BackupRestoreClient {
	return &brClient{
		client:                   httpClient,
		backupRestoreBaseAddress: backupRestoreBaseAddress,
		etcdConfigFilePath:       etcdConfigFilePath}
}

func (c *brClient) GetInitializationStatus(ctx context.Context) (InitStatus, error) {
	response, err := c.createAndExecuteHTTPRequest(ctx, http.MethodGet, c.backupRestoreBaseAddress+"/initialization/status")
	if err != nil {
		return Unknown, err
	}
	defer util.CloseResponseBody(response)

	if !util.ResponseHasOKCode(response) {
		return Unknown, fmt.Errorf("server returned error response code when attempting to get initialization status: %v", response)
	}

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return Unknown, err
	}
	initializationStatus := string(bodyBytes)

	switch initializationStatus {
	case New.String():
		return New, nil
	case Successful.String():
		return Successful, nil
	default:
		return InProgress, nil
	}
}

func (c *brClient) TriggerInitialization(ctx context.Context, validationType ValidationType) error {
	// TODO (@aaronfern): triggering initialization should not be using `GET` verb. `POST` should be used instead. This will require changes to backup-restore (to be done later).
	url := c.backupRestoreBaseAddress + fmt.Sprintf("/initialization/start?mode=%s", validationType)
	response, err := c.createAndExecuteHTTPRequest(ctx, http.MethodGet, url)
	if err != nil {
		return err
	}
	defer util.CloseResponseBody(response)

	if !util.ResponseHasOKCode(response) {
		return fmt.Errorf("server returned error response code when attempting to trigger initialization: %v", response)
	}

	return nil
}

func (c *brClient) GetEtcdConfig(ctx context.Context) (string, error) {
	response, err := c.createAndExecuteHTTPRequest(ctx, http.MethodGet, c.backupRestoreBaseAddress+"/config")
	if err != nil {
		return "", err
	}
	defer util.CloseResponseBody(response)

	if !util.ResponseHasOKCode(response) {
		return "", fmt.Errorf("server returned error response code when attempting to fetch etcd config: %v", response)
	}

	etcdConfigBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	if err = os.WriteFile(c.etcdConfigFilePath, etcdConfigBytes, 0644); err != nil {
		return "", err
	}
	return c.etcdConfigFilePath, nil
}

func (c *brClient) createAndExecuteHTTPRequest(ctx context.Context, method, url string) (*http.Response, error) {
	// create cancellable child context for http request
	httpCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// create new request
	req, err := http.NewRequestWithContext(httpCtx, method, url, nil)
	if err != nil {
		return nil, err
	}

	// send http request
	response, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func createClient(brConfig types.BackupRestoreConfig) (*http.Client, error) {
	tlsConfig, err := createTLSConfig(brConfig)
	if err != nil {
		return nil, err
	}
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   httpClientRequestTimeout,
	}
	return client, nil
}

func createTLSConfig(brConfig types.BackupRestoreConfig) (*tls.Config, error) {
	if brConfig.TLSEnabled {
		caCertPool, err := util.CreateCACertPool(*brConfig.CaCertBundlePath)
		if err != nil {
			return nil, err
		}
		return &tls.Config{
			RootCAs: caCertPool,
		}, nil
	}

	// If TLS is not enabled then create an insecure TLS config
	return &tls.Config{
		InsecureSkipVerify: true,
	}, nil
}
