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
)

var (
	shard string
	env   string
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

type environmentInfo struct {
	URL        string
	ShardToKey map[string]string
}

// envToShardToKey holds each environment's URL, shards, and public keys
var envToShardToKey = map[string]environmentInfo{
	"production": {
		URL: "https://ctfe.sigstore.dev",
		ShardToKey: map[string]string{
			"test": ctTest,
			"2022": ct2022,
		},
	},
	"staging": {
		URL: "https://ctfe.sigstage.dev",
		ShardToKey: map[string]string{
			"test":   ctStagingTest,
			"2022":   ctStaging2022,
			"2022-2": ctStaging2022_2,
		},
	},
}

const sthPath = "/ct/v1/get-sth"

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

	envInfo, ok := envToShardToKey[env]
	if !ok {
		panic(fmt.Sprintf("environment %s not supported", env))
	}

	url := envInfo.URL
	pemPubKey, ok := envInfo.ShardToKey[shard]
	if !ok {
		panic(fmt.Sprintf("shard %s not supported for env %s", shard, env))
	}

	sthURL := fmt.Sprintf("%s/%s%s", strings.TrimRight(url, "/"), shard, sthPath)
	sth, err := getSTH(context.TODO(), sthURL)
	if err != nil {
		panic(fmt.Sprintf("failed to get STH: %v", err))
	}

	fmt.Printf("received STH: %+v\n", sth)

	if sth.Timestamp == 0 {
		panic("got no timestamp from the STH")
	}

	// verify signature on STH
	pubKey, err := cryptoutils.UnmarshalPEMToPublicKey([]byte(pemPubKey))
	if err != nil {
		panic("unable to unmarshal PEM public key")
	}
	v, err := ct.NewSignatureVerifier(pubKey)
	if err != nil {
		panic("unable to create verifier")
	}
	if err := v.VerifySTHSignature(*sth); err != nil {
		panic("unable to verify STH!")
	}
	fmt.Println("STH verified")
}

func getSTH(ctx context.Context, url string) (*ct.SignedTreeHead, error) {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = int(retryCount)

	req, err := retryClient.Get(url)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	var sth ct.SignedTreeHead
	if err := json.Unmarshal(body, &sth); err != nil {
		return nil, err
	}
	return &sth, nil
}
