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

var (
	funcMap = template.FuncMap{
		"Add": Add,
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
