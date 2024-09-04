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

package archive

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/dpeckett/archivefs/tarfs"
	"github.com/dpeckett/uncompr"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"k8s.io/apimachinery/pkg/util/sets"
)

// Create creates an OCI image archive from a set of image references.
func Create(ctx context.Context, outputPath string, images sets.String, platform *v1.Platform) error {
	ociLayoutDir, err := os.MkdirTemp("", "airgapify-archive-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary archive directory: %w", err)
	}
	defer os.RemoveAll(ociLayoutDir)

	p, err := layout.FromPath(ociLayoutDir)
	if err != nil {
		p, err = layout.Write(ociLayoutDir, empty.Index)
		if err != nil {
			return fmt.Errorf("failed to create image archive: %w", err)
		}
	}

	for image := range images {
		options := []remote.Option{
			remote.WithContext(ctx),
			remote.WithAuthFromKeychain(authn.DefaultKeychain),
		}

		if platform != nil {
			options = append(options, remote.WithPlatform(*platform))
		}

		ref, err := name.ParseReference(image)
		if err != nil {
			return fmt.Errorf("failed to parse image reference %q: %w", image, err)
		}

		slog.Info("Fetching image", "image", image)

		img, err := remote.Image(ref, options...)
		if err != nil {
			return fmt.Errorf("failed to fetch image %q: %w", image, err)
		}

		layoutOpts := []layout.Option{
			layout.WithAnnotations(map[string]string{
				"org.opencontainers.image.ref.name": ref.String(),
			}),
		}

		if platform != nil {
			layoutOpts = append(layoutOpts, layout.WithPlatform(*platform))
		}

		if err = p.AppendImage(img, layoutOpts...); err != nil {
			return fmt.Errorf("failed to create image archive: %w", err)
		}
	}

	slog.Info("Writing image archive", "path", outputPath)

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	// Optionally compress the output file based on the file extension.
	w, err := uncompr.NewWriter(outputFile, filepath.Base(outputPath))
	if err != nil {
		return fmt.Errorf("failed to create compressor: %w", err)
	}
	defer w.Close()

	if err := tarfs.Create(w, os.DirFS(ociLayoutDir)); err != nil {
		return fmt.Errorf("failed to create oci image archive: %w", err)
	}

	return nil
}
