/* SPDX-License-Identifier: Apache-2.0
 *
 * Copyright 2024 Damian Peckett <damian@pecke.tt>.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConfigExtractionRuleSpec struct {
	// TypeMeta is the type of object to apply the rule to.
	metav1.TypeMeta `json:",inline"`
	// Paths is a list of JSON paths to extract image references from.
	Paths []string `json:"paths"`
}

type ConfigSpec struct {
	// Rules is a list of custom image extraction rules to apply to the manifests.
	Rules []ConfigExtractionRuleSpec `json:"rules,omitempty"`
	// Images is a list of additional images to include in the archive.
	// This is useful for images that are not directly referenced in the manifests.
	// Eg. those that are created by operators.
	Images []string `json:"images,omitempty"`
}

// +kubebuilder:object:root=true
type Config struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ConfigSpec `json:"spec"`
}

// +kubebuilder:object:root=true
type ConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Config `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Config{}, &ConfigList{})
}
