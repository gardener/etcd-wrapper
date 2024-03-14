// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

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
