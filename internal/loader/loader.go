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
