package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
)

var (
	url   string
	limit int
)

func init() {
	flag.StringVar(&url, "url", "", "The URL to test rate limiting against")
	flag.IntVar(&limit, "limit", 1000, "The max number of requests to make. May begin to see connection failures at higher values")
	flag.Parse()
}

func main() {
	if err := rateLimit(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func rateLimit() error {
	if url == "" {
		return fmt.Errorf("please set url to rate limit via --url flag")
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
					b, err := ioutil.ReadAll(resp.Body)
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
		return fmt.Errorf("No 429 status code was received, rate limiting may not have worked")
	}
	return nil
}
