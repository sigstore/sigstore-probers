package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"

	ct "github.com/google/certificate-transparency-go"
	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	f_log "github.com/transparency-dev/formats/log"
	tdnote "github.com/transparency-dev/formats/note"
)

var (
	shard      string
	env        string
	retryCount uint
)

// ctfe.sigstore.dev/test
const ctTest = `
-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEbfwR+RJudXscgRBRpKX1XFDy3Pyu
dDxz/SfnRi1fT8ekpfBd2O1uoz7jr3Z8nKzxA69EUQ+eFCFI3zeubPWU7w==
-----END PUBLIC KEY-----`

// ctfe.sigstore.dev/2022
const ct2022 = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEiPSlFi0CmFTfEjCUqF9HuCEcYXNK
AaYalIJmBZ8yyezPjTqhxrKBpMnaocVtLJBI1eM3uXnQzQGAJdJ4gs9Fyw==
-----END PUBLIC KEY-----`

// ctfe.sigstage.dev/test
const ctStagingTest = `-----BEGIN RSA PUBLIC KEY-----
MIICCgKCAgEA27A2MPQXm0I0v7/Ly5BIauDjRZF5Jor9vU+QheoE2UIIsZHcyYq3
slHzSSHy2lLj1ZD2d91CtJ492ZXqnBmsr4TwZ9jQ05tW2mGIRI8u2DqN8LpuNYZG
z/f9SZrjhQQmUttqWmtu3UoLfKz6NbNXUnoo+NhZFcFRLXJ8VporVhuiAmL7zqT5
3cXR3yQfFPCUDeGnRksnlhVIAJc3AHZZSHQJ8DEXMhh35TVv2nYhTI3rID7GwjXX
w4ocz7RGDD37ky6p39Tl5NB71gT1eSqhZhGHEYHIPXraEBd5+3w9qIuLWlp5Ej/K
6Mu4ELioXKCUimCbwy+Cs8UhHFlqcyg4AysOHJwIadXIa8LsY51jnVSGrGOEBZev
opmQPNPtyfFY3dmXSS+6Z3RD2Gd6oDnNGJzpSyEk410Ag5uvNDfYzJLCWX9tU8lI
xNwdFYmIwpd89HijyRyoGnoJ3entd63cvKfuuix5r+GHyKp1Xm1L5j5AWM6P+z0x
igwkiXnt+adexAl1J9wdDxv/pUFEESRF4DG8DFGVtbdH6aR1A5/vD4krO4tC1QYU
SeyL5Mvsw8WRqIFHcXtgybtxylljvNcGMV1KXQC8UFDmpGZVDSHx6v3e/BHMrZ7g
joCCfVMZ/cFcQi0W2AIHPYEMH/C95J2r4XbHMRdYXpovpOoT5Ca78gsCAwEAAQ==
-----END RSA PUBLIC KEY-----`

// ctfe.sigstage.dev/2022
const ctStaging2022 = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEh99xuRi6slBFd8VUJoK/rLigy4bY
eSYWO/fE6Br7r0D8NpMI94+A63LR/WvLxpUUGBpY8IJA3iU2telag5CRpA==
-----END PUBLIC KEY-----`

// ctfe.sigstage.dev/2022-2
const ctStaging2022_2 = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE8gEDKNme8AnXuPBgHjrtXdS6miHq
c24CRblNEOFpiJRngeq8Ko73Y+K18yRYVf1DXD4AVLwvKyzdNdl5n0jUSQ==
-----END PUBLIC KEY-----`

// log2026-1.ctfe.sigstage.dev
const ctStagingLog2026_1 = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEv8+Fp+klTMlOd0FU+eekotPzlaF9
orvv9ZgdLXq5+MmoGThLNigXIapXjW0lujsU6+ZHKZ6UPzSuz+V8YxLoQw==
-----END PUBLIC KEY-----`

type shardInfo struct {
	url                 string
	key                 string
	origin              string
	sthEndpoint         string
	checkpointEndpoint  string
	writeHealthEndpoint string
}

// envToShardToInfo holds each environment's URL, shards, public keys and read/write paths
var envToShardInfo = map[string]map[string]shardInfo{
	"production": {
		"test": shardInfo{
			url:         "https://ctfe.sigstore.dev",
			key:         ctTest,
			sthEndpoint: "test/ct/v1/get-sth",
		},
		"2022": shardInfo{
			url:         "https://ctfe.sigstore.dev",
			key:         ct2022,
			sthEndpoint: "2022/ct/v1/get-sth",
		},
	},
	"staging": {
		"test": shardInfo{
			url:         "https://ctfe.sigstage.dev",
			key:         ctStagingTest,
			sthEndpoint: "test/ct/v1/get-sth",
		},
		"2022": shardInfo{
			url:         "https://ctfe.sigstage.dev",
			key:         ctStaging2022,
			sthEndpoint: "2022/ct/v1/get-sth",
		},
		"2022-2": shardInfo{
			url:         "https://ctfe.sigstage.dev",
			key:         ctStaging2022_2,
			sthEndpoint: "2022-2/ct/v1/get-sth",
		},
		"log2026-1": shardInfo{
			url:                 "https://log2026-1.ctfe.sigstage.dev",
			origin:              "log2026-1.ctfe.sigstage.dev",
			key:                 ctStagingLog2026_1,
			checkpointEndpoint:  "checkpoint",
			writeHealthEndpoint: "ct/v1/get-roots",
		},
	},
}

var retryClient *retryablehttp.Client

func main() {
	flag.StringVar(&shard, "shard", "", "The shard of the STH to get")
	flag.StringVar(&env, "env", "", "The environment (production or staging)")
	flag.UintVar(&retryCount, "retry-count", 5, "number of times to retry requests")
	flag.Parse()

	if env == "" {
		panic("Need to specify --env")
	}
	if shard == "" {
		panic("Need to specify --shard")
	}

	retryClient = retryablehttp.NewClient()
	retryClient.RetryMax = int(retryCount)

	envInfo, ok := envToShardInfo[env]
	if !ok {
		panic(fmt.Sprintf("environment %s not supported", env))
	}

	shardInfo, ok := envInfo[shard]
	if !ok {
		panic(fmt.Sprintf("shard %s not supported for env %s", shard, env))
	}
	url := shardInfo.url
	pemPubKey := shardInfo.key

	ctx := context.Background()

	switch {
	case shardInfo.sthEndpoint != "":
		sthURL := fmt.Sprintf("%s/%s", strings.TrimRight(url, "/"), shardInfo.sthEndpoint)
		sth, err := getSTH(ctx, sthURL)
		if err != nil {
			panic(fmt.Sprintf("failed to get STH: %v", err))
		}
		fmt.Printf("received STH: %+v\n", sth)
		err = verifySTH(pemPubKey, sth)
		if err != nil {
			panic(err)
		}
	case shardInfo.checkpointEndpoint != "":
		cpURL := fmt.Sprintf("%s/%s", strings.TrimRight(url, "/"), shardInfo.checkpointEndpoint)
		cp, err := getCheckpoint(ctx, cpURL)
		if err != nil {
			panic(err)
		}
		fmt.Printf("received checkpoint: %s\n", cp)
		err = verifyCheckpoint(cp, shardInfo.origin, pemPubKey)
		if err != nil {
			panic(err)
		}
	}

	if shardInfo.writeHealthEndpoint != "" {
		url := fmt.Sprintf("%s/%s", strings.TrimRight(url, "/"), shardInfo.writeHealthEndpoint)
		_, err := httpGet(ctx, url)
		if err != nil {
			panic(fmt.Sprintf("could not reach write service at %s: %v", shardInfo.writeHealthEndpoint, err))
		}
		fmt.Println("verified write service health")
	}
}

func httpGet(ctx context.Context, url string) ([]byte, error) {
	req, err := retryClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer req.Body.Close()
	return io.ReadAll(req.Body)
}

func getSTH(ctx context.Context, url string) (*ct.SignedTreeHead, error) {
	body, err := httpGet(ctx, url)
	if err != nil {
		return nil, err
	}
	var sth ct.SignedTreeHead
	if err := json.Unmarshal(body, &sth); err != nil {
		return nil, err
	}
	return &sth, nil
}

func verifySTH(pemPubKey string, sth *ct.SignedTreeHead) error {
	if sth.Timestamp == 0 {
		return fmt.Errorf("got no timestamp from the STH")
	}

	// verify signature on STH
	pubKey, err := cryptoutils.UnmarshalPEMToPublicKey([]byte(pemPubKey))
	if err != nil {
		return fmt.Errorf("unable to unmarshal PEM public key: %w", err)
	}
	v, err := ct.NewSignatureVerifier(pubKey)
	if err != nil {
		return fmt.Errorf("unable to create verifier: %w", err)
	}
	if err := v.VerifySTHSignature(*sth); err != nil {
		return fmt.Errorf("unable to verify STH: %w", err)
	}
	fmt.Println("STH verified")
	return nil
}

func getCheckpoint(ctx context.Context, url string) ([]byte, error) {
	body, err := httpGet(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("fetching checkpoint: %w", err)
	}
	return body, nil
}

func verifyCheckpoint(checkpoint []byte, origin, pemPubKey string) error {
	pubKey, err := cryptoutils.UnmarshalPEMToPublicKey([]byte(pemPubKey))
	if err != nil {
		return fmt.Errorf("unable to unmarshal PEM public key: %w", err)
	}
	verifierKey, err := tdnote.RFC6962VerifierString(origin, pubKey)
	if err != nil {
		return fmt.Errorf("creating verifier string: %w", err)
	}
	verifier, err := tdnote.NewVerifier(verifierKey)
	if err != nil {
		return fmt.Errorf("creating note verifier: %w", err)
	}
	_, _, _, err = f_log.ParseCheckpoint(checkpoint, verifier.Name(), verifier)
	if err != nil {
		return fmt.Errorf("unable to verify checkpoint: %w", err)
	}
	fmt.Println("Checkpoint verified")
	return nil
}
