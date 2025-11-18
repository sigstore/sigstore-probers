package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/tuf"
)

var (
	scPath  string
	staging bool
	limit   int
)

func init() {
	flag.StringVar(&scPath, "signing-config", "", "Path to the signing config")
	flag.BoolVar(&staging, "staging", false, "Whether to use the public instance staging environment (otherwise use the public instance production environment)")
	flag.IntVar(&limit, "limit", 1000, "The max number of requests to make. May begin to see connection failures at higher values")
	flag.Parse()
}

func main() {
	var err error

	var signingConfig *root.SigningConfig
	if scPath != "" {
		signingConfig, err = root.NewSigningConfigFromPath(scPath)
		if err != nil {
			log.Fatal("Failed to load signing config: ", err)
		}
	} else {
		if staging {
			opts := tuf.DefaultOptions()
			opts.Root = tuf.StagingRoot()
			opts.RepositoryBaseURL = tuf.StagingMirror
			signingConfig, err = root.FetchSigningConfigWithOptions(opts)
			if err != nil {
				log.Fatal("Failed to fetch staging signing config: ", err)
			}
		} else {
			signingConfig, err = root.FetchSigningConfig()
			if err != nil {
				log.Fatal("Failed to fetch prod signing config: ", err)
			}
		}
	}

	urls, err := getURLsFromSigningConfig(signingConfig)
	if err != nil {
		log.Fatal("Failed to get URLs from signing config: ", err)
	}

	var wg sync.WaitGroup
	errs := make(chan error, len(urls))
	for _, u := range urls {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			fmt.Printf("Rate limiting %s\n", u)
			if err := rateLimit(u); err != nil {
				errs <- fmt.Errorf("error rate limiting %s: %v", u, err)
			}
		}(u)
	}
	wg.Wait()
	close(errs)

	hasErrors := false
	for err := range errs {
		fmt.Println(err)
		hasErrors = true
	}
	if hasErrors {
		os.Exit(1)
	}
	fmt.Println("Successfully triggered rate limiting on all targets.")
}

func getURLsFromSigningConfig(sc *root.SigningConfig) ([]string, error) {
	rekorV1Service, err := root.SelectService(sc.RekorLogURLs(), []uint32{1}, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to select Rekor v1 service: %w", err)
	}
	rekorV1URL := rekorV1Service.URL + "/api/v1/log"

	var rekorV2URL string
	rekorV2Service, err := root.SelectService(sc.RekorLogURLs(), []uint32{2}, time.Now())
	if err != nil {
		// Hardcode Rekor v2 URL in prod until available in signing config
		if !staging {
			rekorV2URL = "https://log2025-1.rekor.sigstore.dev/healthz"
		} else {
			return nil, fmt.Errorf("failed to select Rekor v2 service for staging: %w", err)
		}
	} else {
		rekorV2URL = rekorV2Service.URL + "/healthz"
	}

	fulcioService, err := root.SelectService(sc.FulcioCertificateAuthorityURLs(), []uint32{1}, time.Now())
	if err != nil {
		return nil, fmt.Errorf("selecting fulcio service: %w", err)
	}
	fulcioURL := fulcioService.URL + "/api/v1/rootCert"

	tsaService, err := root.SelectService(sc.TimestampAuthorityURLs(), []uint32{1}, time.Now())
	if err != nil {
		return nil, fmt.Errorf("selecting tsa service: %w", err)
	}
	tsaURL := tsaService.URL + "/certchain"

	return []string{rekorV1URL, rekorV2URL, fulcioURL, tsaURL}, nil
}

func rateLimit(url string) error {
	if url == "" {
		return fmt.Errorf("no url provided for rate limiting test")
	}
	var errCount uint64
	wg := sync.WaitGroup{}
	var rateLimited atomic.Bool

	// thread-safe client, created once
	tr := &http.Transport{
		MaxConnsPerHost: 10,
	}
	client := &http.Client{Transport: tr}

	jobs := make(chan int, limit)
	// create a worker pool of 3 concurrent connections to slow down requests
	for w := 1; w <= 3; w++ {
		go func() {
			for range jobs {
				defer wg.Done()

				resp, err := client.Get(url)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					atomic.AddUint64(&errCount, 1)
				} else if resp.StatusCode == 429 {
					// We want a 429 status code to show that rate limiting worked
					b, err := io.ReadAll(resp.Body)
					switch {
					case err != nil:
						fmt.Fprintln(os.Stderr, err)
						atomic.AddUint64(&errCount, 1)
					case strings.Contains(string(b), "Too Many Requests"):
						rateLimited.Store(true)
					default:
						// 429 was returned but did not contain expected string
						fmt.Fprintln(os.Stderr, string(b))
					}
					resp.Body.Close()
				}
			}
		}()
	}

	// Currently 975 req/min is allowed, or 16/s
	for i := 0; i < limit; i++ {
		if rateLimited.Load() {
			break
		}
		wg.Add(1)
		jobs <- i
	}

	// close channel and wait for completion
	close(jobs)
	wg.Wait()

	if errCount > 0 {
		fmt.Printf("%d out of %d requests had connection errors\n", errCount, limit)
	}
	if !rateLimited.Load() {
		return fmt.Errorf("no 429 status code was received, rate limiting may not have worked")
	}
	return nil
}
