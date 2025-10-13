// Copyright 2025 The Sigstore Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"flag"
	"fmt"
	"log/slog"
	"net/url"
	"os"

	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/tuf"
	"github.com/transparency-dev/merkle/rfc6962"
	"github.com/transparency-dev/tessera/api"
	"github.com/transparency-dev/tessera/client"
	"github.com/transparency-dev/tessera/fsck"
	"golang.org/x/mod/sumdb/note"
	"golang.org/x/sync/errgroup"
)

var (
	staging = flag.Bool("staging", false, "Whether to use the staging environment instead of the production environment")
)

func main() {
	flag.Parse()
	ctx := context.Background()

	trustedRoot, signingConfig, err := downloadTUFMaterials(*staging)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	urls, err := selectRekorURLs(signingConfig)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	eg, ctx := errgroup.WithContext(ctx)

	for _, u := range urls {
		eg.Go(func() error {
			verifier, err := verifierForOrigin(trustedRoot, u)
			if err != nil {
				return err
			}
			slog.Info("Running consistency check for", "log", u.Hostname())
			err = check(ctx, verifier, u)
			if err != nil {
				return fmt.Errorf("consistency check failed for log %s: %w", u.Hostname(), err)
			}
			slog.Info("Verified consistency for", "log", u.Hostname())
			return nil
		})
	}
	err = eg.Wait()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	slog.Info("Finished!")
}

func downloadTUFMaterials(staging bool) (*root.TrustedRoot, *root.SigningConfig, error) {
	opts := tuf.DefaultOptions()
	if staging {
		opts = opts.WithRoot(tuf.StagingRoot()).WithRepositoryBaseURL(tuf.StagingMirror)
	}
	client, err := tuf.New(opts)
	if err != nil {
		return nil, nil, fmt.Errorf("creating tuf client: %w", err)
	}
	trustedRootBytes, err := client.GetTarget("trusted_root.json")
	if err != nil {
		return nil, nil, fmt.Errorf("fetching trusted root: %w", err)
	}
	trustedRoot, err := root.NewTrustedRootFromJSON(trustedRootBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing trusted root: %w", err)
	}
	signingConfig, err := root.GetSigningConfig(client)
	if err != nil {
		return nil, nil, fmt.Errorf("fetching signing config: %w", err)
	}
	return trustedRoot, signingConfig, nil
}

func selectRekorURLs(signingConfig *root.SigningConfig) ([]*url.URL, error) {
	const rekorVersion = uint32(2)
	services := signingConfig.RekorLogURLs()
	urls := make([]*url.URL, 0)
	for _, svc := range services {
		if svc.MajorAPIVersion == rekorVersion {
			u, err := url.Parse(svc.URL)
			if err != nil {
				return nil, fmt.Errorf("parsing log URL: %w", err)
			}
			urls = append(urls, u)
		}
	}
	return urls, nil
}

func verifierForOrigin(trustedRoot *root.TrustedRoot, u *url.URL) (note.Verifier, error) {
	tlogs := trustedRoot.RekorLogs()
	var pubKey crypto.PublicKey
	baseURL := u.String()
	for _, tlog := range tlogs {
		if tlog.BaseURL == baseURL {
			pubKey = tlog.PublicKey
			break
		}
	}
	if pubKey == nil {
		return nil, fmt.Errorf("could not find key for URL [%s]", baseURL)
	}
	origin := u.Hostname()
	var noteKey string
	var err error
	switch key := pubKey.(type) {
	case ed25519.PublicKey:
		noteKey, err = note.NewEd25519VerifierKey(origin, key)
		if err != nil {
			return nil, fmt.Errorf("converting key to note format: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported key type %T", key)
	}
	verifier, err := note.NewVerifier(noteKey)
	if err != nil {
		return nil, fmt.Errorf("creating verifier for key: %w", err)
	}
	return verifier, nil
}

func check(ctx context.Context, verifier note.Verifier, u *url.URL) error {
	fetcher, err := client.NewHTTPFetcher(u, nil)
	if err != nil {
		return fmt.Errorf("creating fetcher: %w", err)
	}
	f := fsck.New(u.Hostname(), verifier, fetcher, defaultMerkleLeafHasher, fsck.Opts{N: 1})
	if err := f.Check(ctx); err != nil {
		return fmt.Errorf("checking Merkle tree consistency: %w", err)
	}
	return nil
}

// copied from https://github.com/transparency-dev/tessera/blob/112eca49fafd45bb8cd4675dbff635e92e6f6dca/cmd/fsck/main.go#L110
func defaultMerkleLeafHasher(bundle []byte) ([][]byte, error) {
	eb := &api.EntryBundle{}
	if err := eb.UnmarshalText(bundle); err != nil {
		return nil, fmt.Errorf("unmarshal: %v", err)
	}
	r := make([][]byte, 0, len(eb.Entries))
	for _, e := range eb.Entries {
		h := rfc6962.DefaultHasher.HashLeaf(e)
		r = append(r, h[:])
	}
	return r, nil
}
