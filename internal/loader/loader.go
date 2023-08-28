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

package loader

import (
	"io"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func LoadObjectsFromFiles(filePaths []string) ([]unstructured.Unstructured, error) {
	var objects []unstructured.Unstructured

	for _, filePath := range filePaths {
		if filePath == "-" {
			fileObjects, err := loadObjectsFromReader(os.Stdin)
			if err != nil {
				return nil, err
			}

			objects = append(objects, fileObjects...)

			continue
		}

		fi, err := os.Stat(filePath)
		if err != nil {
			return nil, err
		}

		if !fi.IsDir() {
			fileObjects, err := loadObjectsFromYAML(filePath)
			if err != nil {
				return nil, err
			}

			objects = append(objects, fileObjects...)
		} else {
			err := filepath.WalkDir(filePath, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}

				if !d.IsDir() {
					fileObjects, err := loadObjectsFromYAML(path)
					if err != nil {
						return err
					}

					objects = append(objects, fileObjects...)
				}

				return nil
			})
			if err != nil {
				return nil, err
			}
		}
	}

	return objects, nil
}

func loadObjectsFromYAML(yamlPath string) ([]unstructured.Unstructured, error) {
	f, err := os.Open(yamlPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return loadObjectsFromReader(f)
}

func loadObjectsFromReader(reader io.Reader) ([]unstructured.Unstructured, error) {
	var objects []unstructured.Unstructured

	decoder := yaml.NewYAMLOrJSONDecoder(reader, 1000000)
	for {
		var obj map[string]any
		if err := decoder.Decode(&obj); err != nil {
			if err == io.EOF {
				break
			}

			return nil, err
		}

		unstrObj := unstructured.Unstructured{Object: obj}
		objects = append(objects, unstrObj)
	}

	return objects, nil
}
