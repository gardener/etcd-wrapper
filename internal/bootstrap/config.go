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

package bootstrap

import (
	"fmt"
	"net/url"
)

const (
	DefaultTLSEnabled       = false
	DefaultSideCarAddress   = "http://127.0.0.1:8080"
	DefaultExitCodeFilePath = "/Users/I544000/go/src/github.tools.sap/I062009/etcd-bootstrapper/exit_code" //"/var/etcd/data/exit_code" //TODO @aaronfern: use proper exit_code path here
)

type SidecarConfig struct {
	BaseAddress      string
	TLSEnabled       bool
	CaCertBundlePath *string
}

func (c *SidecarConfig) Validate() error {
	// TODO @aaronfern: improve config validation
	u, urlParseErr := url.Parse(c.BaseAddress)
	if urlParseErr != nil {
		return fmt.Errorf("error parsing base address URL: %s", urlParseErr.Error())
	}
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("invalid url passed as sidecar base address %s", urlParseErr.Error())
	}
	if c.TLSEnabled {
		if *c.CaCertBundlePath == "" {
			return fmt.Errorf("certificate bundle path cannot be empty when TLS is enabled")
		}
		if u.Scheme != "https" {
			return fmt.Errorf("incorrect scheme for sidecar address. https URL expected")
		}
	}
	return nil
}
