package certs

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"

	"github.com/sirupsen/logrus"
	certutil "k8s.io/client-go/util/cert"
	keyutil "k8s.io/client-go/util/keyutil"
)

func GetCertName(name string) string {
	return fmt.Sprintf("%s.crt", name)
}

func GetKeyName(name string) string {
	return fmt.Sprintf("%s.key", name)
}

func GetKeySigner(alg x509.PublicKeyAlgorithm) (crypto.Signer, error) {
	if alg == x509.ECDSA {
		return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	}

	return rsa.GenerateKey(rand.Reader, 4096)
}

func ParseIPsFromString(ipStrs []string) ([]net.IP, error) {
	var ips []net.IP
	for _, sip := range ipStrs {
		tip := net.ParseIP(sip)
		if tip == nil {
			return nil, fmt.Errorf("invalid ip: %s", sip)
		}
		ips = append(ips, tip)
	}

	return ips, nil
}

func ReadCertFromFile(filename string) (*x509.Certificate, error) {
	certs, err := certutil.CertsFromFile(filename)
	if err != nil {
		return nil, err
	}
	// just support one certs in file
	return certs[0], nil
}

func ReadKeyFromFile(filename string) (crypto.Signer, error) {
	privateKey, err := keyutil.PrivateKeyFromFile(filename)
	if err != nil {
		return nil, err
	}
	switch key := privateKey.(type) {
	case *rsa.PrivateKey:
		return key, nil
	case *ecdsa.PrivateKey:
		return key, nil
	default:
		return nil, fmt.Errorf("file: %s with unsupport private key type", filename)
	}
}

func WriteKey(key crypto.Signer, filename string) error {
	encodedKey, err := keyutil.MarshalPrivateKeyToPEM(key)
	if err != nil {
		logrus.Errorf("marshal private key failed: %v", err)
		return err
	}
	if err := keyutil.WriteKey(filename, encodedKey); err != nil {
		logrus.Errorf("write key: %s failed: %v", filename, err)
		return err
	}
	return nil
}

func WriteCert(cert *x509.Certificate, filename string) error {
	certData, err := certutil.EncodeCertificates(cert)
	if err != nil {
		logrus.Errorf("encode certificate failed: %v", err)
		return err
	}
	if err := certutil.WriteCert(filename, certData); err != nil {
		logrus.Errorf("write certificate: %s failed: %v", filename, err)
		return err
	}
	return nil
}

func WritePublicKey(key crypto.PublicKey, filename string) error {
	mdata, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		logrus.Errorf("marshal public key failed: %v", err)
		return err
	}
	block := pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: mdata,
	}

	if err := keyutil.WriteKey(filename, pem.EncodeToMemory(&block)); err != nil {
		logrus.Errorf("write key: %s failed: %v", filename, err)
		return err
	}
	return nil
}
