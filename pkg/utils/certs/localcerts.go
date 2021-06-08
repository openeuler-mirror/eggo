package certs

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"gitee.com/openeuler/eggo/pkg/utils/runner"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	certutil "k8s.io/client-go/util/cert"
	keyutil "k8s.io/client-go/util/keyutil"
)

type LocalCertGenerator struct {
	keyAlgorithm x509.PublicKeyAlgorithm
	lr           runner.Runner
}

func NewLocalCertGenerator() CertGenerator {
	// TODO: only support RSA now
	return &LocalCertGenerator{
		keyAlgorithm: x509.RSA,
		lr:           &runner.LocalRunner{},
	}
}

func (l *LocalCertGenerator) RunCommand(cmd string) (string, error) {
	return l.lr.RunCommand(cmd)
}

func (l *LocalCertGenerator) CreateServiceAccount(savePath string) error {
	_, err := keyutil.PrivateKeyFromFile(filepath.Join(savePath, ServiceAccountPrivateKeyName))
	if err == nil {
		logrus.Info("service account certs exist\n")
		return nil
	} else if !os.IsNotExist(err) {
		return errors.Wrapf(err, "service account exist, but not valid")
	}

	signer, err := GetKeySigner(l.keyAlgorithm)
	if err != nil {
		logrus.Errorf("new private key for service account failed: %v", err)
		return err
	}
	err = WriteKey(signer, filepath.Join(savePath, "sa.key"))
	if err != nil {
		logrus.Errorf("write private key for service account failed: %v", err)
		return err
	}
	err = WritePublicKey(signer.Public(), filepath.Join(savePath, "sa.pub"))
	if err != nil {
		logrus.Errorf("write public key for service account failed: %v", err)
	}
	return err
}

func (l *LocalCertGenerator) CreateCA(config *CertConfig, savePath string, name string) error {
	if _, err := certutil.CertsFromFile(filepath.Join(savePath, GetCertName(name))); err == nil {
		if _, err := keyutil.PrivateKeyFromFile(filepath.Join(savePath, GetKeyName(name))); err == nil {
			logrus.Infof("[certs] using exist %s ca", GetCertName(name))
			return nil
		}
		logrus.Infof("[certs] using exist %s keyless ca", name)
		return nil
	}
	ips, err := ParseIPsFromString(config.AltNames.IPs)
	if err != nil {
		logrus.Errorf("parse altnames failed: %v", err)
		return err
	}

	signer, err := GetKeySigner(config.PublicKeyAlgorithm)
	if err != nil {
		logrus.Errorf("invalid public key algorithm: %v", err)
		return err
	}

	cc := certutil.Config{
		CommonName:   config.CommonName,
		Organization: config.Organizations,
		Usages:       config.Usages,
		AltNames: certutil.AltNames{
			DNSNames: config.AltNames.DNSNames,
			IPs:      ips,
		},
	}
	cert, err := certutil.NewSelfSignedCACert(cc, signer)
	if err != nil {
		logrus.Errorf("create self signed ca cert failed: %v", err)
		return err
	}

	if err := WriteKey(signer, filepath.Join(savePath, GetKeyName(name))); err != nil {
		logrus.Errorf("write key: %s failed: %v", GetKeyName(name), err)
		return err
	}

	if err := WriteCert(cert, filepath.Join(savePath, GetCertName(name))); err != nil {
		logrus.Errorf("write cert: %s failed: %v", GetCertName(name), err)
		return err
	}

	return nil
}

func (l *LocalCertGenerator) CreateCertAndKey(caCertPath, caKeyPath string, config *CertConfig, savePath string, name string) error {
	serial, err := rand.Int(rand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		logrus.Errorf("generate rand serial failed: %v", err)
		return err
	}
	caCert, err := ReadCertFromFile(caCertPath)
	if err != nil {
		logrus.Errorf("read ca cert failed: %v", err)
		return err
	}
	caKey, err := ReadKeyFromFile(caKeyPath)
	if err != nil {
		logrus.Errorf("read ca key failed: %v", err)
		return err
	}
	signer, err := GetKeySigner(config.PublicKeyAlgorithm)
	if err != nil {
		logrus.Errorf("invalid public key algorithm: %v", err)
		return err
	}
	ips, err := ParseIPsFromString(config.AltNames.IPs)
	if err != nil {
		logrus.Errorf("parse altnames failed: %v", err)
		return err
	}

	certConf := x509.Certificate{
		Subject: pkix.Name{
			CommonName:   config.CommonName,
			Organization: config.Organizations,
		},
		DNSNames:     config.AltNames.DNSNames,
		IPAddresses:  ips,
		SerialNumber: serial,
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  config.Usages,
		NotBefore:    caCert.NotBefore,
		NotAfter:     time.Now().Add(time.Hour * 24 * 365).UTC(),
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &certConf, caCert, signer.Public(), caKey)
	if err != nil {
		logrus.Errorf("crete cert: %s failed: %v", name, err)
		return err
	}
	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		logrus.Errorf("parse cert: %s failed: %v", name, err)
		return err
	}

	if err := WriteKey(signer, filepath.Join(savePath, GetKeyName(name))); err != nil {
		logrus.Errorf("write key: %s failed: %v", GetKeyName(name), err)
		return err
	}

	if err := WriteCert(cert, filepath.Join(savePath, GetCertName(name))); err != nil {
		logrus.Errorf("write cert: %s failed: %v", GetCertName(name), err)
		return err
	}
	return nil
}

func (l *LocalCertGenerator) CreateKubeConfig(savePath, filename string, caCertPath, credName, certPath, keyPath string, endpoint string) error {
	writeFile := filepath.Join(savePath, filename)
	cfg := &CreateKubeConfig{
		clusterName:    "default-cluster",
		server:         endpoint,
		caPath:         caCertPath,
		credName:       credName,
		clientKeyPath:  keyPath,
		clientCertPath: certPath,
	}
	return createKubeConfig(cfg, writeFile)
}

func (l *LocalCertGenerator) CleanAll(savePath string) error {
	return os.RemoveAll(savePath)
}

type CreateKubeConfig struct {
	clusterName    string
	server         string
	caPath         string
	credName       string
	clientKeyPath  string
	clientCertPath string
}

func createKubeConfig(conf *CreateKubeConfig, filename string) error {
	if conf == nil {
		return fmt.Errorf("require config for kubeconfig")
	}
	caData, err := ioutil.ReadFile(conf.caPath)
	if err != nil {
		return err
	}
	keyData, err := ioutil.ReadFile(conf.clientKeyPath)
	if err != nil {
		return err
	}
	certData, err := ioutil.ReadFile(conf.clientCertPath)
	if err != nil {
		return err
	}
	cfg := clientcmdapi.Config{
		CurrentContext: "default-system",
		Contexts: map[string]*clientcmdapi.Context{
			"default-system": {
				Cluster:  conf.clusterName,
				AuthInfo: conf.credName,
			},
		},
		Clusters: map[string]*clientcmdapi.Cluster{
			conf.clusterName: {
				Server:                   conf.server,
				CertificateAuthorityData: caData,
			},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			conf.credName: {
				ClientKeyData:         keyData,
				ClientCertificateData: certData,
			},
		},
	}
	err = clientcmd.WriteToFile(cfg, filename)
	if err != nil {
		return err
	}
	return nil
}
