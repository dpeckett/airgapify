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

package extractor_test

import (
	"testing"

	"github.com/dpeckett/airgapify/internal/extractor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/sets"
)

func TestImageReferenceExtractor(t *testing.T) {
	objects := []unstructured.Unstructured{
		{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Pod",
				"spec": map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{
							"name":  "container1",
							"image": "image1:v1",
						},
						map[string]interface{}{
							"name":  "container2",
							"image": "image2:v2",
						},
					},
				},
			},
		},
	}

	e := extractor.NewImageReferenceExtractor(extractor.DefaultRules)
	result, err := e.ExtractImageReferences(objects)
	require.NoError(t, err)

	expected := sets.New[string]("image1:v1", "image2:v2")
	assert.True(t, expected.Equal(result))
}
