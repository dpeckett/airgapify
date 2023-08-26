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

package archive

import (
	"context"
	"fmt"
	"os"

	"github.com/cheggaaa/pb/v3"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/klauspost/compress/zstd"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/sets"
)

type Options struct {
	DisableProgress bool
	Platform        *v1.Platform
}

// Create creates a Docker image archive from a set of image references.
func Create(ctx context.Context, logger *zap.Logger, outputPath string, images sets.Set[string], opts *Options) error {
	refToImage := make(map[name.Reference]v1.Image)
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

		logger.Info("Fetching image", zap.String("image", image))

		refToImage[ref], err = remote.Image(ref, options...)
		if err != nil {
			return fmt.Errorf("failed to fetch image %q: %w", image, err)
		}
	}

	logger.Info("Writing image archive", zap.String("path", outputPath))

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create image archive: %w", err)
	}
	defer f.Close()

	w, err := zstd.NewWriter(f)
	if err != nil {
		return fmt.Errorf("failed to create zstd writer: %w", err)
	}
	defer w.Close()

	var progressCh chan v1.Update
	if !opts.DisableProgress {
		progressCh = progressBar()
	}

	options := []tarball.WriteOption{
		tarball.WithProgress(progressCh),
	}

	err = tarball.MultiRefWrite(refToImage, w, options...)
	if progressCh != nil {
		close(progressCh)
	}
	if err != nil {
		return fmt.Errorf("failed to write image archive: %w", err)
	}

	return nil
}

func progressBar() chan v1.Update {
	progressCh := make(chan v1.Update, 100)
	go func() {
		var bar *pb.ProgressBar
		for update := range progressCh {
			if bar == nil {
				bar = pb.Start64(update.Total)
			}
			bar = bar.SetCurrent(update.Complete)
		}
		if bar != nil {
			bar.Finish()
		}
	}()

	return progressCh
}
