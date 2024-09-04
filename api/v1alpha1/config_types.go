// SPDX-License-Identifier: AGPL-3.0-or-later
/*
 * Copyright (C) 2024 Damian Peckett <damian@pecke.tt>.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <https://www.gnu.org/licenses/>.
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
