/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type PackageSrcConfig struct {
	// +optional
	// tar.gz
	Type string `json:"type,omitempty"`

	// +optional
	// untar path on dst node
	DstPath string `json:"dstpath,omitempty"`

	// source packages name
	SrcPackages map[string]string `json:"srcPackages,omitempty"`
}

type PackageConfig struct {
	Name string `json:"name"`
	// repo bin file dir image json shell
	Type     string `json:"type"`
	Dst      string `json:"dst,omitempty"`
	Schedule string `json:"schedule,omitempty"`
	TimeOut  string `json:"timeout,omitempty"`
}

type AdditionConfig struct {
	// +optional
	Master []*PackageConfig `json:"master,omitempty"`

	// +optional
	Worker []*PackageConfig `json:"worker,omitempty"`

	// +optional
	ETCD []*PackageConfig `json:"etcd,omitempty"`

	// +optional
	LoadBalance []*PackageConfig `json:"loadbalance,omitempty"`
}

type InstallConfig struct {
	PackageSrc *PackageSrcConfig `json:"package-source,omitempty"`

	// +optional
	KubernetesMaster []*PackageConfig `json:"kubernetes-master,omitempty"`

	// +optional
	KubernetesWorker []*PackageConfig `json:"kubernetes-worker,omitempty"`

	// +optional
	Network []*PackageConfig `json:"network,omitempty"`

	// +optional
	ETCD []*PackageConfig `json:"etcd,omitempty"`

	// +optional
	LoadBalance []*PackageConfig `json:"loadbalance,omitempty"`

	// +optional
	Container []*PackageConfig `json:"container,omitempty"`

	// +optional
	Image []*PackageConfig `json:"image,omitempty"`

	// +optional
	Dns []*PackageConfig `json:"dns,omitempty"`

	// +optional
	Addition AdditionConfig `json:"addition,omitempty"`
}

type OpenPorts struct {
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65535
	Port *int32 `json:"port"`

	// tcp/udp
	Protocol string `json:"protocol"`
}
type OpenPortsConfig struct {
	// +optional
	Master []*OpenPorts `json:"master,omitempty"`

	// +optional
	Worker []*OpenPorts `json:"worker,omitempty"`

	// +optional
	ETCD []*OpenPorts `json:"etcd,omitempty"`

	// +optional
	LoadBalance []*OpenPorts `json:"loadbalance,omitempty"`
}

// InfrastructureSpec defines the desired state of Infrastructure
type InfrastructureSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// PackagePersistentVolumeClaim volume stored install packages
	//+kubebuilder:validation:Required
	PackagePersistentVolumeClaim *v1.ObjectReference `json:"packagePersistentVolumeClaim,omitempty"`

	InstallConfig InstallConfig `json:"install,omitempty"`

	OpenPorts OpenPortsConfig `json:"open-ports,omitempty"`
}

// InfrastructureStatus defines the observed state of Infrastructure
type InfrastructureStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Infrastructure is the Schema for the infrastructures API
type Infrastructure struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InfrastructureSpec   `json:"spec,omitempty"`
	Status InfrastructureStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// InfrastructureList contains a list of Infrastructure
type InfrastructureList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Infrastructure `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Infrastructure{}, &InfrastructureList{})
}
