GIT_TAG ?= $(shell git describe --tags --always --dirty)
GIT_HASH ?= $(shell git rev-parse HEAD)

LDFLAGS=-buildid= -X sigs.k8s.io/release-utils/version.gitVersion=$(GIT_TAG) -X sigs.k8s.io/release-utils/version.gitCommit=$(GIT_HASH)

KO_DOCKER_REPO ?= ghcr.io/sigstore/sigstore-probers

.PHONY: release-images
release-images:
	LDFLAGS="$(LDFLAGS)" KO_DOCKER_REPO=$(KO_DOCKER_REPO) \
	ko build --base-import-paths --platform=all --tags $(GIT_TAG),latest --image-refs imagerefs-prober ./prober/prober/

.PHONY: sign-release-images
sign-release-images:
	echo "Signing prober"; export GIT_HASH=$(GIT_HASH) GIT_VERSION=$(GIT_TAG) ARTIFACT=imagerefs-prober; ./scripts/sign-release-images.sh
