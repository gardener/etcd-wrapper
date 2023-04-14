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

package cmd

import (
	"bufio"
	"html/template"
	"io"
)

var (
	cliHelpTemplate = `
NAME:
{{printf "%s - %s" .Name .ShortDesc}}

USAGE:
{{printf "\t%s" .UsageLine}}
{{if .LongDesc}}
DESCRIPTION:
{{printf "\t%s" .LongDesc}}
{{end}}
`
)

// PrintHelp prints out help test for the start-etcd command
func PrintHelp(w io.Writer) error {
	bufW := bufio.NewWriter(w)
	defer func() {
		_ = bufW.Flush()
	}()
	return executeTemplate(w, cliHelpTemplate, EtcdCmd)
}

func executeTemplate(w io.Writer, tmplText string, tmplData interface{}) error {
	tmpl := template.Must(template.New("usage").Parse(tmplText))
	if err := tmpl.Execute(w, tmplData); err != nil {
		return err
	}
	return nil
}
