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
	flag.IntVar(&limit, "limit", 1000, "The max number of requests to make")
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
	rateLimited := false
	// Currently 975 req/min is allowed, or 16/s
	for i := 0; i < limit; i++ {
		if rateLimited {
			break
		}
		wg.Add(1)
		go func() {
			req, err := http.Get(url)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				atomic.AddUint64(&errCount, 1)
			} else if req.StatusCode == 429 {
				// We want a 429 status code to show that rate limiting worked
				b, _ := ioutil.ReadAll(req.Body)
				if strings.Contains(string(b), "Too Many Requests") {
					rateLimited = true
				} else {
					fmt.Fprintln(os.Stderr, string(b))
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
	if errCount > 0 {
		fmt.Printf("%d out of %d requests had connection errors\n", errCount, limit)
	}
	if !rateLimited {
		return fmt.Errorf("No 429 status code was received, rate limiting may not have worked")
	}
	return nil
}
