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

package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	goruntime "runtime"
	"time"

	"github.com/dpeckett/airgapify/api/v1alpha1"
	airgapifyv1alpha1 "github.com/dpeckett/airgapify/api/v1alpha1"
	"github.com/dpeckett/airgapify/internal/archive"
	"github.com/dpeckett/airgapify/internal/constants"
	"github.com/dpeckett/airgapify/internal/extractor"
	"github.com/dpeckett/airgapify/internal/loader"
	"github.com/dpeckett/airgapify/internal/util"
	"github.com/dpeckett/telemetry"
	telemetryv1alpha1 "github.com/dpeckett/telemetry/v1alpha1"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/urfave/cli/v2"
	"k8s.io/apimachinery/pkg/runtime"
)

func main() {
	persistentFlags := []cli.Flag{
		&cli.GenericFlag{
			Name:  "log-level",
			Usage: "Set the log verbosity level",
			Value: util.FromSlogLevel(slog.LevelInfo),
		},
	}

	initLogger := func(c *cli.Context) error {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: (*slog.Level)(c.Generic("log-level").(*util.LevelFlag)),
		})))

		return nil
	}

	// Collect anonymized usage statistics.
	var telemetryReporter *telemetry.Reporter

	initTelemetry := func(c *cli.Context) error {
		telemetryReporter = telemetry.NewReporter(c.Context, slog.Default(), telemetry.Configuration{
			BaseURL: constants.TelemetryURL,
			Tags:    []string{"airgapify"},
		})

		// Some basic system information.
		info := map[string]string{
			"os":      goruntime.GOOS,
			"arch":    goruntime.GOARCH,
			"num_cpu": fmt.Sprintf("%d", goruntime.NumCPU()),
			"version": constants.Version,
		}

		telemetryReporter.ReportEvent(&telemetryv1alpha1.TelemetryEvent{
			Kind:   telemetryv1alpha1.TelemetryEventKindInfo,
			Name:   "ApplicationStart",
			Values: info,
		})

		return nil
	}

	shutdownTelemetry := func(c *cli.Context) error {
		if telemetryReporter == nil {
			return nil
		}

		telemetryReporter.ReportEvent(&telemetryv1alpha1.TelemetryEvent{
			Kind: telemetryv1alpha1.TelemetryEventKindInfo,
			Name: "ApplicationStop",
		})

		// Don't want to block the shutdown of the application for too long.
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		if err := telemetryReporter.Shutdown(ctx); err != nil {
			slog.Error("Failed to close telemetry reporter", slog.Any("error", err))
		}

		return nil
	}

	app := &cli.App{
		Name:  "airgapify",
		Usage: "A little tool that will construct an OCI image archive from a set of Kubernetes manifests.",
		Flags: append([]cli.Flag{
			&cli.StringSliceFlag{
				Name:     "file",
				Aliases:  []string{"f"},
				Usage:    "Path to one or more Kubernetes manifests.",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Where to write the OCI image archive (optionally compressed).",
				Value:   "images.tar",
			},
			&cli.StringFlag{
				Name:    "platform",
				Aliases: []string{"p"},
				Usage:   "The target platform for the image archive.",
			},
		}, persistentFlags...),
		Before: util.BeforeAll(initLogger, initTelemetry),
		After:  shutdownTelemetry,
		Action: func(c *cli.Context) error {
			objects, err := loader.LoadObjectsFromFiles(c.StringSlice("file"))
			if err != nil {
				return fmt.Errorf("failed to load objects: %w", err)
			}

			slog.Info("Loaded objects", "count", len(objects))

			rules := extractor.DefaultRules

			for _, obj := range objects {
				if obj.GetAPIVersion() == v1alpha1.GroupVersion.String() && obj.GetKind() == "Config" {
					slog.Info("Found airgapify config")

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
				slog.Info("Found image references", "count", images.Len())
			}

			var platform *v1.Platform
			if c.IsSet("platform") {
				platform, err = v1.ParsePlatform(c.String("platform"))
				if err != nil {
					return fmt.Errorf("failed to parse platform: %w", err)
				}
			}

			outputPath := c.String("output")
			if err := archive.Create(c.Context, outputPath, images, platform); err != nil {
				return fmt.Errorf("failed to create image archive: %w", err)
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		slog.Error("Error", slog.Any("error", err))
		os.Exit(1)
	}
}
