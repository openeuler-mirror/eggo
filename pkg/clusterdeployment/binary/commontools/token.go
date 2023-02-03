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
 * Create: 2021-06-07
 * Description: util for token
 ******************************************************************************/
package commontools

import (
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/constants"
	"isula.org/eggo/pkg/utils/runner"

	kkutil "github.com/kubesphere/kubekey/pkg/util"
	"github.com/lithammer/dedent"
	"github.com/sirupsen/logrus"
	bootstraputil "k8s.io/cluster-bootstrap/token/util"
)

const (
	TokenTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: bootstrap-token-{{ .ID }}
  namespace: kube-system
type: bootstrap.kubernetes.io/token
stringData:
  description: "{{ .Description }}"
  token-id: {{ .ID }}
  token-secret: {{ .Secret }}
  expiration: {{ .Expiration }}
  {{- range $i, $v := .Usages }}
  {{ $v }}
  {{- end }}
  {{- if .AuthExtraGroups }}
  auth-extra-groups: {{ .AuthExtraGroups }}
  {{- end }}
`
)

func CreateBootstrapToken(r runner.Runner, bconf *api.BootstrapTokenConfig, kubeconfig, manifestDir string) error {
	var sb strings.Builder
	var usages []string
	now := time.Now()
	tmpl := template.Must(template.New("bootstrap token").Parse(dedent.Dedent(TokenTemplate)))
	datastore := map[string]interface{}{}
	datastore["Description"] = bconf.Description
	datastore["ID"] = bconf.ID
	datastore["Secret"] = bconf.Secret
	// default set ttl 24 hours
	ttl := 24 * time.Hour
	if bconf.TTL != nil {
		ttl = *bconf.TTL
	}
	datastore["Expiration"] = now.Add(ttl).Format(time.RFC3339)
	for _, usage := range bconf.Usages {
		usages = append(usages, fmt.Sprintf("usage-bootstrap-%s: \"true\"", usage))
	}
	datastore["Usages"] = usages
	if len(bconf.AuthExtraGroups) > 0 {
		datastore["AuthExtraGroups"] = strings.Join(bconf.AuthExtraGroups, ",")
	}
	coreConfig, err := kkutil.Render(tmpl, datastore)
	if err != nil {
		logrus.Errorf("rend core config failed: %v", err)
		return err
	}
	sb.WriteString("sudo -E /bin/sh -c \"")
	sb.WriteString(fmt.Sprintf("mkdir -p %s", manifestDir))
	tokenYamlBase64 := base64.StdEncoding.EncodeToString([]byte(coreConfig))
	sb.WriteString(fmt.Sprintf(" && echo %s | base64 -d > %s/bootstrap_token.yaml", tokenYamlBase64, manifestDir))
	sb.WriteString(fmt.Sprintf(" && KUBECONFIG=%s kubectl apply -f %s/bootstrap_token.yaml", kubeconfig, manifestDir))
	sb.WriteString("\"")

	_, err = r.RunCommand(sb.String())
	if err != nil {
		logrus.Errorf("create core config failed: %v", err)
		return err
	}
	return nil
}

func CreateBootstrapTokensForCluster(r runner.Runner, ccfg *api.ClusterConfig) error {
	for _, token := range ccfg.BootStrapTokens {
		if err := CreateBootstrapToken(r, token, filepath.Join(ccfg.GetConfigDir(), constants.KubeConfigFileNameAdmin), ccfg.GetManifestDir()); err != nil {
			logrus.Errorf("create bootstrap token failed: %v", err)
			return err
		}
	}
	return nil
}

func GetBootstrapToken(r runner.Runner, tokenStr string, kubeconfig, manifestDir string) (string, error) {
	// TODO: check exist token first
	token, id, secret, err := ParseBootstrapTokenStr(tokenStr)
	if err != nil {
		return "", err
	}
	bconf := &api.BootstrapTokenConfig{
		Description:     "bootstrap token for eggo",
		ID:              id,
		Secret:          secret,
		Usages:          []string{"authentication", "signing"},
		AuthExtraGroups: []string{"system:bootstrappers:worker,system:bootstrappers:ingress"},
	}
	err = CreateBootstrapToken(r, bconf, kubeconfig, manifestDir)

	return token, err
}

func ParseBootstrapTokenStr(useToken string) (token, id, secret string, err error) {
	if useToken == "" {
		tokenStr, err := bootstraputil.GenerateBootstrapToken()
		if err != nil {
			logrus.Errorf("generate bootstrap token string error: %v", err)
			return "", "", "", err
		}
		useToken = tokenStr
	}

	splitStrs := bootstraputil.BootstrapTokenRegexp.FindStringSubmatch(useToken)
	if len(splitStrs) != 3 {
		logrus.Errorf("generate bootstrap token string invalid: %s", useToken)
		return "", "", "", fmt.Errorf("generate bootstrap token string invalid: %s", useToken)
	}

	return splitStrs[0], splitStrs[1], splitStrs[2], nil
}
