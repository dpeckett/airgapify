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

	expected := sets.NewString("image1:v1", "image2:v2")
	assert.True(t, expected.Equal(result))
}
