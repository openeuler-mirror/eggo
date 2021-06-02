package commontools

import (
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"gitee.com/openeuler/eggo/pkg/api"
	"gitee.com/openeuler/eggo/pkg/constants"
	"gitee.com/openeuler/eggo/pkg/utils/runner"
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
  $v
  {{- end }}
  {{- if .AuthExtraGroups }}
  auth-extra-groups: {{ .AuthExtraGroups }}
  {{- end }}
`
)

func CreateBootstrapToken(r runner.Runner, bconf *api.BootstrapTokenConfig, kubeconfig string) error {
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
		usages = append(usages, fmt.Sprintf("usage-bootstrap-%s: true", usage))
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
	sb.WriteString("sudo -E /bin/sh -c \"mkdir -p /tmp/.eggo")
	tokenYamlBase64 := base64.StdEncoding.EncodeToString([]byte(coreConfig))
	sb.WriteString(fmt.Sprintf(" && echo %s | base64 -d > /tmp/.eggo/bootstrap_token.yaml", tokenYamlBase64))
	sb.WriteString(fmt.Sprintf(" && KUBECONFIG=%s", kubeconfig))
	sb.WriteString(" && kubectl apply -f /tmp/.eggo/bootstrap_token.yaml")
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
		if err := CreateBootstrapToken(r, token, filepath.Join(ccfg.GetConfigDir(), constants.KubeConfigFileNameAdmin)); err != nil {
			logrus.Errorf("create bootstrap token failed: %v", err)
			return err
		}
	}
	return nil
}

func GetBootstrapToken(r runner.Runner, tokenStr string, kubeconfig string) (string, error) {
	token, id, secret, err := ParseBootstrapTokenStr(tokenStr)
	if err != nil {
		return "", err
	}
	bconf := &api.BootstrapTokenConfig{
		Description:     "bootstrap token for eggo",
		ID:              id,
		Secret:          secret,
		Usages:          []string{"authentication", "signing"},
		AuthExtraGroups: []string{""},
	}
	err = CreateBootstrapToken(r, bconf, kubeconfig)

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
