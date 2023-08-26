/* SPDX-License-Identifier: Apache-2.0
 *
 * Copyright 2023 Damian Peckett <damian@pecke.tt>.
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

package extractor

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/util/jsonpath"
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
			APIVersion: "airgapify.gpu-ninja.com/v1alpha1",
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

func (e *ImageReferenceExtractor) ExtractImageReferences(objects []unstructured.Unstructured) (sets.Set[string], error) {
	images := sets.New[string]()

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

func (e *ImageReferenceExtractor) extractImagesFromObject(object unstructured.Unstructured) (sets.Set[string], error) {
	images := sets.New[string]()

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

func extractValueUsingJSONPath(object unstructured.Unstructured, jsonPath string) (sets.Set[string], error) {
	j := jsonpath.New("extractor").AllowMissingKeys(true)
	if err := j.Parse("{ " + jsonPath + " }"); err != nil {
		return nil, err
	}

	results, err := j.FindResults(object.Object)
	if err != nil {
		return nil, err
	}

	images := sets.New[string]()
	for _, r := range results {
		for _, v := range r {
			images.Insert(fmt.Sprintf("%v", v.Interface()))
		}
	}

	return images, nil
}
