// Copyright 2023 SAP SE or an SAP affiliate company
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

package util

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

var (
	testdataPath       = "../testdata"
	etcdCACertFilePath = filepath.Join(testdataPath, "ca.pem")
)

func TestCreateCACertPool(t *testing.T) {
	table := []struct {
		description       string
		trustedCAFilePath string
		expectError       bool
	}{
		{"test: should return error when empty ca cert file path is passed", "", true},
		{"test: should return error when wrong ca cert file path is passed", testdataPath + "/wrong-path", true},
		{"test: should not return error when valid ca cert file path is passed", etcdCACertFilePath, false},
	}

	g := NewWithT(t)
	for _, entry := range table {
		t.Log(entry.description)
		_, err := CreateCACertPool(entry.trustedCAFilePath)
		g.Expect(err != nil).To(Equal(entry.expectError))
	}
}
