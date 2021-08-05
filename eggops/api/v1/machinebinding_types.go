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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	UsageMaster      = 1
	UsageWorker      = 2
	UsageEtcd        = 4
	UsageLoadbalance = 8
)

var (
	StrUsages = []string{"Master", "Worker", "Etcd", "Loadbalance"}
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type MachineSetOfUsage struct {
	Machines []*Machine `json:"machines,omitempty"`
	Usage    string     `json:"usage,omitempty"`
}

func (ms MachineSetOfUsage) MatchType(expect uint32) bool {
	str := getUsageStr(int32(expect))
	return str == ms.Usage
}

// MachineBindingSpec defines the desired state of MachineBinding
type MachineBindingSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// remember which machines binded
	// key is usage string
	MachineSets []MachineSetOfUsage `json:"machineSets,omitempty"`

	// usages, support: 1 represent master, 2 represent worker, 4 represent etcd, 8 represent loadbalance
	// key is uid
	Usages map[string]int32 `json:"usages,omitempty"`
}

type MachineCondition struct {
	UsagesStatus int32  `json:"usagesStatus,omitempty"`
	Message      string `json:"message,omitempty"`
}

// MachineBindingStatus defines the observed state of MachineBinding
type MachineBindingStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// key is uid
	Conditions map[string]MachineCondition `json:"usages,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// MachineBinding is the Schema for the machinebindings API
type MachineBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MachineBindingSpec   `json:"spec,omitempty"`
	Status MachineBindingStatus `json:"status,omitempty"`
}

func getUsageStr(usage int32) string {
	i := 0
	for usage > 0 {
		usage = usage >> 1
		i++
	}

	return StrUsages[i-1]
}

func (mb *MachineBinding) UpdateCondition(mc MachineCondition, uid string) {
	if mb.Status.Conditions == nil {
		mb.Status.Conditions = make(map[string]MachineCondition)
	}
	mb.Status.Conditions[uid] = mc
}

func (mb *MachineBinding) AddMachine(machine Machine, usage int32) {
	if mb.Spec.Usages == nil {
		mb.Spec.Usages = make(map[string]int32)
	}
	old, ok := mb.Spec.Usages[string(machine.UID)]
	if ok && (old&usage) == usage {
		// machine is exist, just ignore
		return
	}
	mb.Spec.Usages[string(machine.UID)] = old | usage

	uStr := getUsageStr(usage)
	for _, set := range mb.Spec.MachineSets {
		if set.Usage == uStr {
			set.Machines = append(set.Machines, &machine)
			return
		}
	}
	mb.Spec.MachineSets = append(mb.Spec.MachineSets, MachineSetOfUsage{
		Usage:    uStr,
		Machines: []*Machine{&machine},
	})
}

//+kubebuilder:object:root=true

// MachineBindingList contains a list of MachineBinding
type MachineBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MachineBinding `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MachineBinding{}, &MachineBindingList{})
}
