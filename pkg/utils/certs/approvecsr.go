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
 * Author: zhangxiaoyu
 * Create: 2021-08-26
 * Description: util for token
 ******************************************************************************/
package certs

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/constants"
	"isula.org/eggo/pkg/utils/kubectl"
	certificatesv1 "k8s.io/api/certificates/v1"
	certificatesv1beta1 "k8s.io/api/certificates/v1beta1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/cert"
)

type ServingCSR interface {
	Do(client *kubernetes.Clientset, worker *api.HostConfig) (bool, error)
}

type CertificateV1 struct {
}

func (cv1 *CertificateV1) check(csr certificatesv1.CertificateSigningRequest, worker *api.HostConfig) bool {
	// 1. check csr has been approved or denied
	if len(csr.Status.Certificate) != 0 || len(csr.Status.Conditions) != 0 {
		return false
	}

	// 2. check csr is requested by nodes
	username := "system:node:" + worker.Name
	if csr.Spec.Username != username {
		return false
	}

	valid := false
	group := "system:nodes"
	for _, g := range csr.Spec.Groups {
		if g == group {
			valid = true
			break
		}
	}
	if !valid {
		logrus.Warnf("csr %s is not requested by nodes", csr.Name)
		return false
	}

	// 3. check csr is requested for serving certificates
	// usageRequired: "server auth"
	// usagesOptional: "digital signature", "key encipherment"
	required := false
	for _, u := range csr.Spec.Usages {
		if u == certificatesv1.UsageServerAuth {
			required = true
			continue
		}

		if u != certificatesv1.UsageDigitalSignature && u != certificatesv1.UsageKeyEncipherment {
			logrus.Warnf("csr %s is not requested for serving certificates", csr.Name)
			return false
		}
	}
	if !required {
		logrus.Warnf("csr %s is not requested for serving certificates", csr.Name)
		return false
	}

	// 4. check only have IP and DNS subjectAltNames that belong to the requesting node,
	//    and have no URI and Email subjectAltNames
	if !checkCSRSubjectAltNames(csr.Name, csr.Spec.Request, worker) {
		return false
	}

	return true
}

func (cv1 *CertificateV1) approve(client *kubernetes.Clientset, csr certificatesv1.CertificateSigningRequest) error {
	csr.Status.Conditions = append(csr.Status.Conditions,
		certificatesv1.CertificateSigningRequestCondition{
			Type:           certificatesv1.CertificateApproved,
			Status:         corev1.ConditionTrue,
			Reason:         "ApproveServing",
			Message:        "This Kubelet Serving CSR was approved",
			LastUpdateTime: v1.Now(),
		})

	_, err := client.CertificatesV1().CertificateSigningRequests().UpdateApproval(context.TODO(), csr.Name, &csr, v1.UpdateOptions{})
	if err != nil {
		logrus.Errorf("approve v1 certificates %s signing request failed", csr.Name)
	}

	return err
}

func (cv1 *CertificateV1) Do(client *kubernetes.Clientset, worker *api.HostConfig) (bool, error) {
	csrList, err := client.CertificatesV1().CertificateSigningRequests().List(context.TODO(), v1.ListOptions{})
	if err != nil {
		return false, err
	}

	for _, csr := range csrList.Items {
		if !cv1.check(csr, worker) {
			continue
		}

		if err := cv1.approve(client, csr); err != nil {
			return false, err
		}

		logrus.Infof("approve serving csr %s of %s", csr.Name, worker.Address)
		return true, nil
	}

	return false, nil
}

type CertificateV1beta1 struct {
}

func (cv1beta1 *CertificateV1beta1) check(csr certificatesv1beta1.CertificateSigningRequest, worker *api.HostConfig) bool {
	// 1. check csr has been approved or denied
	if len(csr.Status.Certificate) != 0 || len(csr.Status.Conditions) != 0 {
		return false
	}

	// 2. check csr is requested by nodes
	username := "system:node:" + worker.Name
	if csr.Spec.Username != username {
		return false
	}

	valid := false
	group := "system:nodes"
	for _, g := range csr.Spec.Groups {
		if g == group {
			valid = true
			break
		}
	}
	if !valid {
		logrus.Warnf("csr %s is not requested by nodes", csr.Name)
		return false
	}

	// 3. check csr is requested for serving certificates
	// usageRequired: "server auth"
	// usagesOptional: "digital signature", "key encipherment"
	required := false
	for _, u := range csr.Spec.Usages {
		if u == certificatesv1beta1.UsageServerAuth {
			required = true
			continue
		}

		if u != certificatesv1beta1.UsageDigitalSignature && u != certificatesv1beta1.UsageKeyEncipherment {
			logrus.Warnf("csr %s is not requested for serving certificates", csr.Name)
			return false
		}
	}
	if !required {
		logrus.Warnf("csr %s is not requested for serving certificates", csr.Name)
		return false
	}

	// 4. check only have IP and DNS subjectAltNames that belong to the requesting node,
	//    and have no URI and Email subjectAltNames
	if !checkCSRSubjectAltNames(csr.Name, csr.Spec.Request, worker) {
		return false
	}

	return true
}

func (cv1beta1 *CertificateV1beta1) approve(client *kubernetes.Clientset, csr certificatesv1beta1.CertificateSigningRequest) error {
	csr.Status.Conditions = append(csr.Status.Conditions,
		certificatesv1beta1.CertificateSigningRequestCondition{
			Type:           certificatesv1beta1.CertificateApproved,
			Reason:         "ApproveServing",
			Message:        "This Kubelet Serving CSR was approved",
			Status:         corev1.ConditionTrue,
			LastUpdateTime: v1.Now(),
		})

	_, err := client.CertificatesV1beta1().CertificateSigningRequests().UpdateApproval(context.TODO(), &csr, v1.UpdateOptions{})
	if err != nil {
		logrus.Errorf("approve beta1 certificates %s signing request failed: %v", csr.Name, err)
	}

	return err
}

func (cv1beta1 *CertificateV1beta1) Do(client *kubernetes.Clientset, worker *api.HostConfig) (bool, error) {
	csrList, err := client.CertificatesV1beta1().CertificateSigningRequests().List(context.TODO(), v1.ListOptions{})
	if err != nil {
		return false, err
	}

	for _, csr := range csrList.Items {
		if !cv1beta1.check(csr, worker) {
			continue
		}

		if err := cv1beta1.approve(client, csr); err != nil {
			return false, err
		}

		logrus.Infof("approve serving csr %s of %s", csr.Name, worker.Address)
		return true, nil
	}

	return false, nil
}

func parseCSR(name string, csr []byte) (*x509.CertificateRequest, error) {
	block, _ := pem.Decode(csr)
	if block == nil {
		return nil, fmt.Errorf("invalid certificate signing request")
	}

	if block.Type != cert.CertificateRequestBlockType {
		return nil, fmt.Errorf("%s is not a certificate signing request", name)
	}

	return x509.ParseCertificateRequest(block.Bytes)
}

func checkCSRSubjectAltNames(name string, csr []byte, worker *api.HostConfig) bool {
	x509, err := parseCSR(name, csr)
	if err != nil {
		logrus.Errorf("parse csr %s failed: %v", name, err)
		return false
	}

	if len(x509.URIs) != 0 || len(x509.EmailAddresses) != 0 {
		logrus.Errorf("invalid csr %s has URI or Email subjectAltNames", name)
		return false
	}
	if len(x509.IPAddresses) != 1 && x509.IPAddresses[0].String() != worker.Address {
		logrus.Errorf("invalid csr %s IP subjectAltNames", name)
		return false
	}
	if len(x509.DNSNames) != 1 && x509.DNSNames[0] != worker.Name {
		logrus.Errorf("invalid csr %s DNS subjectAltNames", name)
		return false
	}

	return true
}

func ApproveCsr(cluster string, workers []*api.HostConfig) error {
	if len(workers) == 0 {
		return nil
	}

	path := filepath.Join(api.GetClusterHomePath(cluster), constants.KubeConfigFileNameAdmin)
	client, err := kubectl.GetKubeClient(path)
	if err != nil {
		return err
	}

	var csr ServingCSR
	if _, err := client.CertificatesV1().CertificateSigningRequests().List(context.TODO(), v1.ListOptions{}); err == nil {
		csr = &CertificateV1{}
	} else if _, err := client.CertificatesV1().CertificateSigningRequests().List(context.TODO(), v1.ListOptions{}); err == nil {
		csr = &CertificateV1beta1{}
	} else {
		return fmt.Errorf("list certificates signing request failed")
	}

	var wg sync.WaitGroup
	wg.Add(len(workers))
	for _, w := range workers {
		go func(worker *api.HostConfig) {
			defer wg.Done()
			var err error
			approved := false
			for times := 0; times < 10; times++ {
				approved, err = csr.Do(client, worker)
				if err != nil {
					logrus.Errorf("do approve certificate failed for %s: %v", worker.Address, err)
					return
				}
				if approved {
					break
				}

				// maybe the serving csr hasn't received
				time.Sleep(time.Duration(10) * time.Second)
			}

			if !approved {
				logrus.Errorf("cannot get certificate signing requests of %s", worker.Address)
			}
		}(w)
	}
	wg.Wait()

	return nil
}
