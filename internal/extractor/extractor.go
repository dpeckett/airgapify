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

package extractor

import (
	"fmt"

	"github.com/dpeckett/airgapify/internal/util/jsonpath"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/sets"
)

type ImageReferenceExtractionRule struct {
	metav1.TypeMeta
	// Paths is a list of JSON paths to extract image references from.
	Paths []string
}

var DefaultRules = []ImageReferenceExtractionRule{
	{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		Paths: []string{"$.spec.containers[*].image"},
	},
	{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		Paths: []string{"$.spec.template.spec.containers[*].image"},
	},
	{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "apps/v1",
		},
		Paths: []string{"$.spec.template.spec.containers[*].image"},
	},
	{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "apps/v1",
		},
		Paths: []string{"$.spec.template.spec.containers[*].image"},
	},
	{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: "batch/v1",
		},
		Paths: []string{"$.spec.template.spec.containers[*].image"},
	},
	{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CronJob",
			APIVersion: "batch/v1",
		},
		Paths: []string{"$.spec.jobTemplate.spec.template.spec.containers[*].image"},
	},
	{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Config",
			APIVersion: "airgapify.pecke.tt/v1alpha1",
		},
		Paths: []string{"$.spec.images[*]"},
	},
}

type ImageReferenceExtractor struct {
	rules []ImageReferenceExtractionRule
}

func NewImageReferenceExtractor(rules []ImageReferenceExtractionRule) *ImageReferenceExtractor {
	return &ImageReferenceExtractor{
		rules: rules,
	}
}

func (e *ImageReferenceExtractor) ExtractImageReferences(objects []unstructured.Unstructured) (sets.String, error) {
	images := sets.NewString()

	for _, object := range objects {
		imagesForObject, err := e.extractImagesFromObject(object)
		if err != nil {
			return nil, err
		}

		if len(imagesForObject) > 0 {
			images = images.Union(imagesForObject)
		}
	}

	return images, nil
}

func (e *ImageReferenceExtractor) extractImagesFromObject(object unstructured.Unstructured) (sets.String, error) {
	images := sets.NewString()

	for _, rule := range e.rules {
		if object.GroupVersionKind() == rule.GroupVersionKind() {
			for _, jsonPath := range rule.Paths {
				results, err := extractValueUsingJSONPath(object, jsonPath)
				if err != nil {
					return nil, fmt.Errorf("failed to extract image references from object %s: %w", object.GetName(), err)
				}

				if len(results) > 0 {
					images = images.Union(results)
				}
			}
		}
	}

	return images, nil
}

func extractValueUsingJSONPath(object unstructured.Unstructured, jsonPath string) (sets.String, error) {
	j := jsonpath.New("extractor").AllowMissingKeys(true)
	if err := j.Parse("{ " + jsonPath + " }"); err != nil {
		return nil, err
	}

	results, err := j.FindResults(object.Object)
	if err != nil {
		return nil, err
	}

	images := sets.NewString()
	for _, r := range results {
		for _, v := range r {
			images.Insert(fmt.Sprintf("%v", v.Interface()))
		}
	}

	return images, nil
}
