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

// CoinbasePingerSpec defines the desired state of CoinbasePinger
type CoinbasePingerSpec struct {
	Endpoint string `json:"endpoint"`
	Interval string `json:"interval"`
}

// CoinbasePingerStatus defines the observed state of CoinbasePinger
type CoinbasePingerStatus struct {
	Conditions []Condition `json:"conditions"`
}

// Condition contains webping result fetched from a pod metadata
type Condition struct {
	Type     string      `json:"type"`
	Status   bool        `json:"status"`
	Reason   string      `json:"reason"`
	Message  string      `json:"message"`
	PingTime metav1.Time `json:"pingTime,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// CoinbasePinger is the Schema for the coinbasepingers API
type CoinbasePinger struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CoinbasePingerSpec   `json:"spec"`
	Status CoinbasePingerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CoinbasePingerList contains a list of CoinbasePinger
type CoinbasePingerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CoinbasePinger `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CoinbasePinger{}, &CoinbasePingerList{})
}
