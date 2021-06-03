/******************************************************************************
 * Copyright (c) Huawei Technologies Co., Ltd. 2021. All rights reserved.
 * eggo licensed under the Mulan PSL v2.
 * You can use this software according to the terms and conditions of the Mulan PSL v2.
 * You may obtain a copy of Mulan PSL v2 at:
 *     http://license.coscl.org.cn/MulanPSL2
 * THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
 * PURPOSE.
 * See the Mulan PSL v2 for more details.
 * Author: haozi007
 * Create: 2021-05-19
 * Description: template utils
 ******************************************************************************/
package template

import (
	"fmt"
	"text/template"

	kkutil "github.com/kubesphere/kubekey/pkg/util"
	"github.com/lithammer/dedent"
)

func Add(a, b int) int {
	return a + b
}

func NotLast(idx, size int) bool {
	return idx != size-1
}

var (
	funcMap = template.FuncMap{
		"Add":     Add,
		"NotLast": NotLast,
	}

	BaseCsrTemplate = `[ req ]
default_bits = 4096
prompt = no
default_md = sha256
{{- if .HaveAltNames }}
req_extensions = req_ext
{{- end }}
distinguished_name = dn

[ dn ]
{{- if .Organization }}
O = {{ .Organization }}
{{- end }}
CN = {{ .CommonName }}

{{- if .HaveAltNames }}

[ req_ext ]
subjectAltName = @alt_names
{{- end }}

{{- if .HaveAltNames }}

[ alt_names ]
{{- range $i, $v := .DNSNames }}
DNS.{{ Add $i 1 }} = {{ $v }}
{{- end }}
{{- range $i, $v := .IPs }}
IP.{{ Add $i 1 }} = {{ $v }}
{{- end }}
{{- end }}

[ v3_ext ]
authorityKeyIdentifier = keyid,issuer:always
basicConstraints = CA:FALSE
keyUsage = keyEncipherment,dataEncipherment
extendedKeyUsage = {{ .ExtendedKeyUsage }}
{{- if .HaveAltNames }}
subjectAltName = @alt_names
{{- end }}
`

	BaseSystemdServiceTemplate = `[Unit]
Description={{ .Description }}
Documentation={{ .Documentation }}
{{- range $i, $v := .Afters }}
After={{ $v }}
{{- end }}

[Service]
{{- range $i, $v := .EnvironmentFiles }}
EnvironmentFile=-{{ $v }}
{{- end }}
{{- range $i, $v := .ExecStartPre }}
ExecStartPre={{ $v }}
{{- end }}
{{- $alen := len .Arguments }}
ExecStart={{ .Command }}{{if ne $alen 0 }} \{{end}}
{{- range $i, $v := .Arguments }}
		{{ $v }}{{if NotLast $i $alen }} \{{end}}
{{- end }}

Restart={{ .RestartPolicy }}
LimitNOFILE={{ .LimitNoFile }}

[Install]
WantedBy={{ .WantedBy }}
`
)

type CsrConfig struct {
	Organization     string
	CommonName       string
	IPs              []string
	DNSNames         []string
	ExtendedKeyUsage string
}

func CreateCsrTemplate(name string, conf *CsrConfig) (string, error) {
	if conf == nil {
		return "", fmt.Errorf("invalid csr config")
	}
	tmpl := template.Must(template.New(name).Funcs(funcMap).Parse(dedent.Dedent(BaseCsrTemplate)))
	datastore := map[string]interface{}{}
	if len(conf.IPs) > 0 {
		datastore["HaveAltNames"] = true
		datastore["IPs"] = conf.IPs
	}
	if len(conf.DNSNames) > 0 {
		datastore["HaveAltNames"] = true
		datastore["DNSNames"] = conf.DNSNames
	}
	datastore["Organization"] = conf.Organization
	datastore["CommonName"] = conf.CommonName
	datastore["ExtendedKeyUsage"] = conf.ExtendedKeyUsage

	return kkutil.Render(tmpl, datastore)
}

type SystemdServiceConfig struct {
	Description      string
	Documentation    string
	Afters           []string
	EnvironmentFiles []string
	Command          string
	Arguments        []string
	RestartPolicy    string
	LimitNoFile      string
	WantedBy         string
	ExecStartPre     []string
}

func CreateSystemdServiceTemplate(name string, conf *SystemdServiceConfig) (string, error) {
	if conf == nil {
		return "", fmt.Errorf("invalid csr config")
	}
	tmpl := template.Must(template.New(name).Funcs(funcMap).Parse(dedent.Dedent(BaseSystemdServiceTemplate)))
	datastore := map[string]interface{}{}

	if conf.Description == "" {
		return "", fmt.Errorf("must provide a description")
	}
	datastore["Description"] = conf.Description

	if conf.Documentation == "" {
		return "", fmt.Errorf("must provide a documentation")
	}
	datastore["Documentation"] = conf.Documentation

	if len(conf.Afters) > 0 {
		datastore["Afters"] = conf.Afters
	}

	if len(conf.ExecStartPre) > 0 {
		datastore["ExecStartPre"] = conf.ExecStartPre
	}

	if len(conf.EnvironmentFiles) > 0 {
		datastore["EnvironmentFiles"] = conf.EnvironmentFiles
	}

	if conf.Command == "" {
		return "", fmt.Errorf("must provide a command")
	}
	datastore["Command"] = conf.Command

	if len(conf.Arguments) > 0 {
		datastore["Arguments"] = conf.Arguments
	}

	restartPolicy := "on-failure"
	if conf.RestartPolicy != "" {
		restartPolicy = conf.RestartPolicy
	}
	datastore["RestartPolicy"] = restartPolicy

	limitNoFile := "65536"
	if conf.LimitNoFile != "" {
		limitNoFile = conf.LimitNoFile
	}
	datastore["LimitNoFile"] = limitNoFile

	wantedBy := "multi-user.target"
	if conf.WantedBy != "" {
		wantedBy = conf.WantedBy
	}
	datastore["WantedBy"] = wantedBy

	return kkutil.Render(tmpl, datastore)
}

func TemplateRender(temp string, datastore map[string]interface{}) (string, error) {
	tmpl := template.Must(template.New("test").Funcs(funcMap).Parse(dedent.Dedent(temp)))
	return kkutil.Render(tmpl, datastore)
}
