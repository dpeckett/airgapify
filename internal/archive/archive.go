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

package archive

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/mholt/archiver/v3"
	"k8s.io/apimachinery/pkg/util/sets"
)

type Options struct {
	DisableProgress bool
	Platform        *v1.Platform
}

// Create creates an OCI image archive from a set of image references.
func Create(ctx context.Context, logger *slog.Logger, outputPath string, images sets.Set[string], opts *Options) error {
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

		if opts.Platform != nil {
			options = append(options, remote.WithPlatform(*opts.Platform))
		}

		ref, err := name.ParseReference(image)
		if err != nil {
			return fmt.Errorf("failed to parse image reference %q: %w", image, err)
		}

		logger.Info("Fetching image", "image", image)

		img, err := remote.Image(ref, options...)
		if err != nil {
			return fmt.Errorf("failed to fetch image %q: %w", image, err)
		}

		layoutOpts := []layout.Option{
			layout.WithAnnotations(map[string]string{
				"org.opencontainers.image.ref.name": ref.String(),
			}),
		}

		if opts.Platform != nil {
			layoutOpts = append(layoutOpts, layout.WithPlatform(*opts.Platform))
		}

		if err = p.AppendImage(img, layoutOpts...); err != nil {
			return fmt.Errorf("failed to create image archive: %w", err)
		}
	}

	format := archiver.TarZstd{
		Tar: &archiver.Tar{
			OverwriteExisting: true,
		},
	}

	logger.Info("Writing image archive", "path", outputPath)

	if err := format.Archive([]string{
		filepath.Join(ociLayoutDir, "blobs"),
		filepath.Join(ociLayoutDir, "index.json"),
		filepath.Join(ociLayoutDir, "oci-layout"),
	}, outputPath); err != nil {
		return fmt.Errorf("failed to create oci image archive: %w", err)
	}

	return nil
}
