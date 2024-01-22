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

package main

import (
	"fmt"
	"log/slog"
	"os"

	airgapifyv1alpha1 "github.com/dpeckett/airgapify/api/v1alpha1"
	"github.com/dpeckett/airgapify/internal/archive"
	"github.com/dpeckett/airgapify/internal/extractor"
	"github.com/dpeckett/airgapify/internal/loader"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/urfave/cli/v2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

func init() {
	_ = airgapifyv1alpha1.AddToScheme(scheme.Scheme)
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	app := &cli.App{
		Name:  "airgapify",
		Usage: "A little tool that will construct an OCI image archive from a set of Kubernetes manifests.",
		Flags: []cli.Flag{
			&cli.GenericFlag{
				Name:    "log-level",
				Aliases: []string{"l"},
				Usage:   "Set the log level",
				Value:   fromLogLevel(slog.LevelInfo),
			},
			&cli.StringSliceFlag{
				Name:     "file",
				Aliases:  []string{"f"},
				Usage:    "Path to one or more Kubernetes manifests.",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Where to write the oci image archive (will be a tar.zst archive).",
				Value:   "images.tar.zst",
			},
			&cli.StringFlag{
				Name:    "platform",
				Aliases: []string{"p"},
				Usage:   "The target platform for the image archive.",
			},
			&cli.BoolFlag{
				Name:  "no-progress",
				Usage: "Disable progress output.",
			},
		},
		Before: func(c *cli.Context) error {
			logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
				Level: (*slog.Level)(c.Generic("log-level").(*logLevelFlag)),
			}))

			return nil
		},
		Action: func(c *cli.Context) error {
			objects, err := loader.LoadObjectsFromFiles(c.StringSlice("file"))
			if err != nil {
				return fmt.Errorf("failed to load objects: %w", err)
			}

			logger.Info("Loaded objects", "count", len(objects))

			rules := extractor.DefaultRules

			for _, obj := range objects {
				if obj.GetAPIVersion() == "airgapify.pecke.tt/v1alpha1" && obj.GetKind() == "Config" {
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
				logger.Info("Found image references", "count", images.Len())
			}

			options := &archive.Options{}

			if c.IsSet("no-progress") && c.Bool("no-progress") {
				options.DisableProgress = true
			}

			if c.IsSet("platform") {
				options.Platform, err = v1.ParsePlatform(c.String("platform"))
				if err != nil {
					return fmt.Errorf("failed to parse platform: %w", err)
				}
			}

			outputPath := c.String("output")

			if err := archive.Create(c.Context, logger, outputPath, images, options); err != nil {
				return fmt.Errorf("failed to create image archive: %w", err)
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		logger.Error("Failed to run application", "error", err)
		os.Exit(1)
	}
}

type logLevelFlag slog.Level

func fromLogLevel(l slog.Level) *logLevelFlag {
	f := logLevelFlag(l)
	return &f
}

func (f *logLevelFlag) Set(value string) error {
	return (*slog.Level)(f).UnmarshalText([]byte(value))
}

func (f *logLevelFlag) String() string {
	return (*slog.Level)(f).String()
}
