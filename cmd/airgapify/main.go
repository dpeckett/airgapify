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
	"fmt"
	"os"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	airgapifyv1alpha1 "github.com/gpu-ninja/airgapify/api/v1alpha1"
	"github.com/gpu-ninja/airgapify/internal/archive"
	"github.com/gpu-ninja/airgapify/internal/extractor"
	"github.com/gpu-ninja/airgapify/internal/loader"
	zaplogfmt "github.com/jsternberg/zap-logfmt"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
)

func init() {
	_ = airgapifyv1alpha1.AddToScheme(scheme.Scheme)
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
				Usage:   "Where to write the image archive.",
				Value:   "images.tar.zst",
			},
			&cli.BoolFlag{
				Name:  "compress",
				Usage: "Compress the image archive using zstd.",
				Value: true,
			},
			&cli.StringFlag{
				Name:    "platform",
				Aliases: []string{"p"},
				Usage:   "The target platform for the image archive.",
			},
		},
		Action: func(cCtx *cli.Context) error {
			objects, err := loader.LoadObjectsFromFiles(cCtx.StringSlice("file"))
			if err != nil {
				return fmt.Errorf("failed to load objects: %w", err)
			}

			logger.Info("Loaded objects", zap.Int("count", len(objects)))

			rules := extractor.DefaultRules

			for _, obj := range objects {
				if obj.GetAPIVersion() == "airgapify.gpu-ninja.com/v1alpha1" && obj.GetKind() == "Config" {
					logger.Info("Found airgapify config")

					var config airgapifyv1alpha1.Config
					err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &config)
					if err != nil {
						return fmt.Errorf("failed to convert config: %w", err)
					}

					for _, rule := range config.Spec.Rules {
						rules = append(rules, extractor.ImageReferenceExtractionRule{
							TypeMeta: rule.TypeMeta,
							Paths:    rule.Paths,
						})
					}
				}
			}

			e := extractor.NewImageReferenceExtractor(rules)
			images, err := e.ExtractImageReferences(objects)
			if err != nil {
				return fmt.Errorf("failed to extract image references: %w", err)
			}

			if images.Len() > 0 {
				logger.Info("Found image references", zap.Int("count", images.Len()))
			}

			options := &archive.Options{
				Compressed: ptr.To(cCtx.Bool("compress")),
			}

			if cCtx.IsSet("platform") {
				options.Platform, err = v1.ParsePlatform(cCtx.String("platform"))
				if err != nil {
					return fmt.Errorf("failed to parse platform: %w", err)
				}
			}

			outputPath := cCtx.String("output")

			if err := archive.Create(cCtx.Context, logger, outputPath, images, options); err != nil {
				return fmt.Errorf("failed to create image archive: %w", err)
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		logger.Fatal("Failed to run application", zap.Error(err))
	}
}
