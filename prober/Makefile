#
# Copyright 2022 The Sigstore Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

TOOLS_DIR := hack/toolz
TOOLS_BIN_DIR := $(abspath $(TOOLS_DIR)/bin)

# Binaries
REKOR-CLI := $(TOOLS_BIN_DIR)/rekor-cli
PROBER := $(TOOLS_BIN_DIR)/prober
COSIGN := $(TOOLS_BIN_DIR)/cosign
CRANE := $(TOOLS_BIN_DIR)/crane
VERIFY := $(TOOLS_BIN_DIR)/verify

.PHONY: all clean rekor-cli prober cosign crane verify

all: rekor-cli prober cosign crane verify

clean:
	rm -rf hack/toolz/bin

## --------------------------------------
## Tooling Binaries
## --------------------------------------

rekor-cli: $(REKOR-CLI)

$(REKOR-CLI): $(TOOLS_DIR)/go.mod
	cd $(TOOLS_DIR); go build -trimpath -tags=toolz -o $(REKOR-CLI) github.com/sigstore/rekor/cmd/rekor-cli

prober: $(PROBER)

$(PROBER): $(TOOLS_DIR)/go.mod
	cd $(TOOLS_DIR); go build -trimpath -tags=toolz -o $(PROBER) github.com/sigstore/scaffolding/cmd/prober

cosign: $(COSIGN)

$(COSIGN): $(TOOLS_DIR)/go.mod
	cd $(TOOLS_DIR); go build -trimpath -tags=toolz -o $(COSIGN) github.com/sigstore/cosign/cmd/cosign

crane: $(CRANE)

$(CRANE): $(TOOLS_DIR)/go.mod
	cd $(TOOLS_DIR); go build -trimpath -tags=toolz -o $(CRANE) github.com/google/go-containerregistry/cmd/crane

verify: $(VERIFY)

$(VERIFY): $(TOOLS_DIR)/go.mod
	cd $(TOOLS_DIR); go build -trimpath -tags=toolz -o $(VERIFY) github.com/sigstore/root-signing/cmd/verify
