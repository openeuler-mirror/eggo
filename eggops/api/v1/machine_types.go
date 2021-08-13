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
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MachineSpec defines the desired state of Machine
type MachineSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// hostname for machine
	//+kubebuilder:validation:Required
	HostName string `json:"hostname,omitempty"`

	// architecture of machine
	Arch string `json:"arch,omitempty"`

	// ip for ssh login
	//+kubebuilder:validation:Required
	IP string `json:"ip,omitempty"`

	// port for ssh login, default is 22
	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:validation:Maximum=65535
	Port *int32 `json:"port,omitempty"`
}

// MachineStatus defines the observed state of Machine
type MachineStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// current role of machine, support: master, etcd, worker, loadbalance
	RoleBindings int32 `json:"role-bindings,omitempty"`

	// which cluster use this machine
	Cluster string `json:"cluster,omitempty"`

	// status of machine, 0 represents success, other represents failed
	Status int32 `json:"status,omitempty"`

	// record error information
	ErrorMessage string `json:"error-message,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Machine is the Schema for the machines API
type Machine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MachineSpec   `json:"spec,omitempty"`
	Status MachineStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MachineList contains a list of Machine
type MachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Machine `json:"items"`
}

func PrintMachineSlice(machines []Machine) string {
	var sb strings.Builder
	sb.WriteString("[")
	for i, m := range machines {
		sb.WriteString(m.Spec.HostName)
		sb.WriteString(": ")
		sb.WriteString(m.Spec.IP)
		if i < len(machines)-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString("]")
	return sb.String()
}

func init() {
	SchemeBuilder.Register(&Machine{}, &MachineList{})
}
