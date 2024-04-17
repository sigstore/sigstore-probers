//go:build toolz
// +build toolz

// Copyright 2022 The Sigstore Authors.
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
//

// This package imports things required by build scripts, to force `go mod` to see them as dependencies
package toolz

import (
	_ "github.com/google/go-containerregistry/cmd/crane"
	_ "github.com/sigstore/cosign/v2/cmd/cosign"
	_ "github.com/sigstore/rekor/cmd/rekor-cli"
	_ "github.com/sigstore/scaffolding/cmd/prober"
)
