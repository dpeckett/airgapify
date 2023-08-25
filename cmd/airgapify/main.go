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

package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	airgapifyv1alpha1 "github.com/gpu-ninja/airgapify/api/v1alpha1"
	zaplogfmt "github.com/jsternberg/zap-logfmt"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
)

func init() {
	_ = airgapifyv1alpha1.AddToScheme(scheme.Scheme)
	_ = corev1.AddToScheme(scheme.Scheme)
	_ = appsv1.AddToScheme(scheme.Scheme)
	_ = batchv1.AddToScheme(scheme.Scheme)
}

func main() {
	config := zap.NewProductionEncoderConfig()
	logger := zap.New(zapcore.NewCore(
		zaplogfmt.NewEncoder(config),
		os.Stdout,
		zap.NewAtomicLevelAt(zapcore.InfoLevel),
	))

	app := &cli.App{
		Name:  "airgapify",
		Usage: "A tool that will construct airgapped image archives for a set of Kubernetes manifests.",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:     "file",
				Aliases:  []string{"f"},
				Usage:    "Path to one or more Kubernetes manifests.",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Where to write the image list.",
				Value:   "images.txt",
			},
		},
		Action: func(cCtx *cli.Context) error {
			objects, err := loadObjectsFromFiles(cCtx.StringSlice("file"))
			if err != nil {
				logger.Fatal("Failed to load objects", zap.Error(err))

				return err
			}

			logger.Info("Loaded objects", zap.Int("count", len(objects)))

			images := sets.New[string]()
			for _, obj := range objects {
				objImages, err := extractImages(logger, &obj)
				if err != nil {
					return fmt.Errorf("failed to find image references: %w", err)
				}

				if objImages.Len() > 0 {
					images = images.Union(objImages)
				}
			}

			if images.Len() > 0 {
				logger.Info("Found image references", zap.Int("count", images.Len()))
			}

			outputPath := cCtx.String("output")

			logger.Info("Writing image references to file", zap.String("path", outputPath))

			f, err := os.Create(outputPath)
			if err != nil {
				return fmt.Errorf("failed to create images file: %w", err)
			}
			defer f.Close()

			for image := range images {
				if _, err := f.WriteString(image + "\n"); err != nil {
					return fmt.Errorf("failed to write image to file: %w", err)
				}
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		logger.Fatal("Failed to run application", zap.Error(err))
	}
}

func loadObjectsFromFiles(files []string) ([]unstructured.Unstructured, error) {
	var objects []unstructured.Unstructured

	for _, filePath := range files {
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
	objectsYAML, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, err
	}

	var objects []unstructured.Unstructured

	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(objectsYAML), 1000000)
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

func extractImages(logger *zap.Logger, obj *unstructured.Unstructured) (sets.Set[string], error) {
	switch obj.GetAPIVersion() {
	case "v1":
		if obj.GetKind() == "Pod" {
			logger.Info("Found pod", zap.String("name", obj.GetName()))

			var pod corev1.Pod
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &pod)
			if err != nil {
				return nil, err
			}

			return extractImagesFromPodSpec(&pod.Spec), nil
		}
	case "apps/v1":
		switch obj.GetKind() {
		case "Deployment":
			logger.Info("Found deployment", zap.String("name", obj.GetName()))

			var deploy v1.Deployment
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &deploy)
			if err != nil {
				return nil, err
			}

			return extractImagesFromPodSpec(&deploy.Spec.Template.Spec), nil
		case "ReplicaSet":
			logger.Info("Found replica set", zap.String("name", obj.GetName()))

			var rs v1.ReplicaSet
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &rs)
			if err != nil {
				return nil, err
			}

			return extractImagesFromPodSpec(&rs.Spec.Template.Spec), nil
		case "StatefulSet":
			logger.Info("Found stateful set", zap.String("name", obj.GetName()))

			var sts v1.StatefulSet
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &sts)
			if err != nil {
				return nil, err
			}

			return extractImagesFromPodSpec(&sts.Spec.Template.Spec), nil
		case "DaemonSet":
			logger.Info("Found daemon set", zap.String("name", obj.GetName()))

			var ds v1.DaemonSet
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &ds)
			if err != nil {
				return nil, err
			}

			return extractImagesFromPodSpec(&ds.Spec.Template.Spec), nil
		}
	case "batch/v1":
		switch obj.GetKind() {
		case "Job":
			logger.Info("Found job", zap.String("name", obj.GetName()))

			var job batchv1.Job
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &job)
			if err != nil {
				return nil, err
			}

			return extractImagesFromPodSpec(&job.Spec.Template.Spec), nil
		case "CronJob":
			logger.Info("Found cron job", zap.String("name", obj.GetName()))

			var cron batchv1.CronJob
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &cron)
			if err != nil {
				return nil, err
			}

			return extractImagesFromPodSpec(&cron.Spec.JobTemplate.Spec.Template.Spec), nil
		}
	case "airgapify.gpu-ninja.com/v1alpha1":
		if obj.GetKind() == "Config" {
			logger.Info("Found airgapify config")

			var config airgapifyv1alpha1.Config
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &config)
			if err != nil {
				return nil, err
			}

			return sets.New[string](config.Spec.Images...), nil
		}
	case "ceph.rook.io/v1":
		if obj.GetKind() == "CephCluster" {
			logger.Info("Found ceph cluster", zap.String("name", obj.GetName()))

			image, _, _ := unstructured.NestedString(obj.Object, "spec", "cephVersion", "image")
			if image != "" {
				return sets.New[string](image), nil
			}
		}
	case "dex.gpu-ninja.com/v1alpha1":
		if obj.GetKind() == "DexIdentityProvider" {
			logger.Info("Found dex identity provider", zap.String("name", obj.GetName()))

			image, _, _ := unstructured.NestedString(obj.Object, "spec", "image")
			if image != "" {
				return sets.New[string](image), nil
			}
		}
	case "ldap.gpu-ninja.com/v1alpha1":
		if obj.GetKind() == "LDAPDirectory" {
			logger.Info("Found ldap directory", zap.String("name", obj.GetName()))

			image, _, _ := unstructured.NestedString(obj.Object, "spec", "image")
			if image != "" {
				return sets.New[string](image), nil
			}
		}
	}

	return sets.New[string](), nil
}

func extractImagesFromPodSpec(spec *corev1.PodSpec) sets.Set[string] {
	images := sets.New[string]()

	for _, initContainer := range spec.InitContainers {
		images.Insert(initContainer.Image)
	}

	for _, container := range spec.Containers {
		images.Insert(container.Image)
	}

	return images
}
