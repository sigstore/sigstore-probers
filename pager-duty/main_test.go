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

package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// TestMissingPagerDutyIntegrationKey tests whether we fail correctly
func TestMissingPagerDutyIntegrationKey(t *testing.T) {
	_ = os.Unsetenv("PAGERDUTY_INTEGRATION_KEY")
	if err := execute(); err == nil {
		t.Error("expected error when no integration key is provided")
	} else {
		if err.Error() != "no integration key provided, please set PAGERDUTY_INTEGRATION_KEY env var" {
			t.Error("incorrect error message provided for missing integration key")
		}
	}
}

func TestInputValidation(t *testing.T) {
	// set this to something so we don't trigger that test
	_ = os.Setenv("PAGERDUTY_INTEGRATION_KEY", "TEST_KEY")

	// set up fake endpoint
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path[1:] {
			case "all_good/v2/enqueue":
				w.WriteHeader(http.StatusAccepted)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"success","message":"Event processed"}`))
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
		}))
	defer server.Close()
	// this mocks all calls to PagerDuty to return 202 Accepted
	endpoint = server.URL + "/all_good"

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "invalid JSON",
			input:   "asdf[3}",
			wantErr: true,
		},
		{
			name:    "valid JSON, missing required fields",
			input:   "{}",
			wantErr: true,
		},
		{
			name: "missing summary",
			input: `{
				"source": "test",
				"severity": "critical",
				"component": "test",
				"group": "test"
			}`,
			wantErr: true,
		},
		{
			name: "blank summary",
			input: `{
				"summary": "",
				"severity": "critical",
				"source": "test",
				"component": "test",
				"group": "test"
			}`,
			wantErr: true,
		},
		{
			name: "summary too long",
			input: `{
				"summary": "thisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024CharactersthisStringIsLongerThan1024Characters",
				"source": "test",
				"severity": "critical",
				"component": "test",
				"group": "test"
			}`,
			wantErr: true,
		},
		{
			name: "missing severity",
			input: `{
				"summary": "",
				"source": "test",
				"component": "test",
				"group": "test"
			}`,
			wantErr: true,
		},
		{
			name: "blank severity",
			input: `{
				"summary": "",
				"severity": "",
				"source": "test",
				"component": "test",
				"group": "test"
			}`,
			wantErr: true,
		},
		{
			name: "critical severity",
			input: `{
				"summary": "test",
				"severity": "critical",
				"source": "test",
				"component": "test",
				"group": "test"
			}`,
			wantErr: false,
		},
		{
			name: "error severity",
			input: `{
				"summary": "test",
				"severity": "error",
				"source": "test",
				"component": "test",
				"group": "test"
			}`,
			wantErr: false,
		},
		{
			name: "warning severity",
			input: `{
				"summary": "test",
				"severity": "warning",
				"source": "test",
				"component": "test",
				"group": "test"
			}`,
			wantErr: false,
		},
		{
			name: "info severity",
			input: `{
				"summary": "test",
				"severity": "info",
				"source": "test",
				"component": "test",
				"group": "test"
			}`,
			wantErr: false,
		},
		{
			name: "bogus severity",
			input: `{
				"summary": "test",
				"severity": "bogus",
				"source": "test",
				"component": "test",
				"group": "test"
			}`,
			wantErr: true,
		},
		{
			name: "missing source",
			input: `{
				"summary": "test",
				"severity": "error",
				"component": "test",
				"group": "test"
			}`,
			wantErr: true,
		},
		{
			name: "empty source",
			input: `{
				"summary": "test",
				"severity": "error",
				"source": "",
				"component": "test",
				"group": "test"
			}`,
			wantErr: true,
		},
		{
			name: "missing component",
			input: `{
				"summary": "test",
				"severity": "error",
				"source": "test",
				"group": "test"
			}`,
			wantErr: true,
		},
		{
			name: "empty component",
			input: `{
				"summary": "test",
				"severity": "error",
				"source": "test",
				"component": "",
				"group": "test"
			}`,
			wantErr: true,
		},
		{
			name: "missing group",
			input: `{
				"summary": "test",
				"severity": "error",
				"source": "test",
				"component": "test",
			}`,
			wantErr: true,
		},
		{
			name: "empty group",
			input: `{
				"summary": "test",
				"severity": "error",
				"source": "test",
				"component": "test",
				"group": ""
			}`,
			wantErr: true,
		},
		{
			name: "invalid details",
			input: `{
				"summary": "test",
				"severity": "error",
				"source": "test",
				"component": "test",
				"group": "test",
				"details": "notJSON"
			}`,
			wantErr: true,
		},
		{
			name: "valid details",
			input: `{
				"summary": "test",
				"severity": "error",
				"source": "test",
				"component": "test",
				"group": "test",
				"details": {
					"key": "value"
				}
			}`,
			wantErr: false,
		},
		{
			name: "valid details with nesting",
			input: `{
				"summary": "test",
				"severity": "error",
				"source": "test",
				"component": "test",
				"group": "test",
				"details": {
					"key": "value",
					"object": {
						"key2": "value2"
					}
				}
			}`,
			wantErr: false,
		},
		{
			name: "valid link without text",
			input: `{
				"summary": "test",
				"severity": "error",
				"source": "test",
				"component": "test",
				"group": "test",
				"links": [{
					"href": "https://example.com"
				}]
			}`,
			wantErr: false,
		},
		{
			name: "valid link with text",
			input: `{
				"summary": "test",
				"severity": "error",
				"source": "test",
				"component": "test",
				"group": "test",
				"links": [{
					"href": "https://example.com",
					"text": "Example"
				}]
			}`,
			wantErr: false,
		},
		{
			name: "valid links with text",
			input: `{
				"summary": "test",
				"severity": "error",
				"source": "test",
				"component": "test",
				"group": "test",
				"links": [{
					"href": "https://example.com",
					"text": "Example"
				},
				{
					"href": "https://example2.com",
					"text": "Example2"
				}]
			}`,
			wantErr: false,
		},
		{
			name: "invalid link - missing href",
			input: `{
				"summary": "test",
				"severity": "error",
				"source": "test",
				"component": "test",
				"group": "test",
				"links": [{
					"href2": "https://example.com",
					"text": "Example"
				}]
			}`,
			wantErr: true,
		},
		{
			name: "invalid link - href not valid URL",
			input: `{
				"summary": "test",
				"severity": "error",
				"source": "test",
				"component": "test",
				"group": "test",
				"links": [{
					"href": "not_a_valid_url",
				}]
			}`,
			wantErr: true,
		},
		{
			name: "one valid link and one invalid link",
			input: `{
				"summary": "test",
				"severity": "error",
				"source": "test",
				"component": "test",
				"group": "test",
				"links": [{
					"href": "https://example.com"
				},
				{
					"href": "not_a_valid_url"
				}]
			}`,
			wantErr: true,
		},
	}
	for _, test := range tests {
		// set os.stdin to the test input
		OsStdIn = bytes.NewBufferString(test.input)
		if err := execute(); (err != nil) != test.wantErr {
			t.Errorf("%v: execute() error = %v, wantErr %v", test.name, err, test.wantErr)
		}
	}
}

func TestRetryLogic(t *testing.T) {
	// set this to something so we don't trigger that test
	_ = os.Setenv("PAGERDUTY_INTEGRATION_KEY", "TEST_KEY")

	input := `{
		"summary": "test",
		"severity": "error",
		"source": "test",
		"component": "test",
		"group": "test",
		"details": {
			"key": "value"
		},
		"links": [{
			"href": "https://example.com",
			"text": "Example"
		},
		{
			"href": "https://example2.com",
			"text": "Example2"
		}]
	}`
	// set up fake endpoint
	var retryCounter, requestCount int
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			requestCount += 1
			fmt.Printf("HTTP Server: requestCount = %d, retryCounter = %d\n", requestCount, retryCounter)
			switch r.URL.Path[1:] {
			case "400/v2/enqueue":
				w.WriteHeader(http.StatusBadRequest)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"error","message":"Bad request"}`))
				return
			case "429/v2/enqueue":
				if retryCounter > 0 {
					w.WriteHeader(http.StatusTooManyRequests)
					retryCounter -= 1
					w.Header().Set("Retry-After", "1")
				} else {
					w.WriteHeader(http.StatusAccepted)
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"status":"success","message":"Event processed"}`))
				}
				return
			case "429_missing/v2/enqueue":
				if retryCounter > 0 {
					w.WriteHeader(http.StatusTooManyRequests)
					retryCounter -= 1
				} else {
					w.WriteHeader(http.StatusAccepted)
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"status":"success","message":"Event processed"}`))
				}
				return
			case "500/v2/enqueue":
				if retryCounter > 0 {
					w.WriteHeader(http.StatusInternalServerError)
					retryCounter -= 1
				} else {
					w.WriteHeader(http.StatusAccepted)
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"status":"success","message":"Event processed"}`))
				}
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
		}))
	defer server.Close()

	tests := []struct {
		name                       string
		responseCode               string
		clientRetries              uint
		serverRetriesBeforeSuccess int
		expectedRequestCount       int
		expectedDuration           time.Duration
		wantErr                    bool
	}{
		{
			name:                 "client doesn't retry on 400 error",
			responseCode:         "400",
			clientRetries:        5,
			expectedRequestCount: 1,
			wantErr:              true,
		},
		{
			name:                       "client only retries once, but server takes more than that",
			responseCode:               "429",
			clientRetries:              1,
			serverRetriesBeforeSuccess: 2,
			expectedRequestCount:       2,
			wantErr:                    true,
		},
		{
			name:                       "client allows sufficient retries, and server responds in time",
			responseCode:               "429",
			clientRetries:              2,
			serverRetriesBeforeSuccess: 1,
			expectedRequestCount:       2,
			expectedDuration:           1 * time.Second,
			wantErr:                    false,
		},
		{
			name:                       "client allows sufficient retries, and server responds in time (without Retry-Afer header)",
			responseCode:               "429_missing",
			clientRetries:              2,
			serverRetriesBeforeSuccess: 1,
			expectedRequestCount:       2,
			expectedDuration:           1 * time.Second,
			wantErr:                    false,
		},
		{
			name:                       "client allows sufficient retries, and server responds in time",
			responseCode:               "500",
			clientRetries:              2,
			serverRetriesBeforeSuccess: 1,
			expectedRequestCount:       2,
			expectedDuration:           1 * time.Second,
			wantErr:                    false,
		},
		{
			name:                       "exponential backoff",
			responseCode:               "500",
			clientRetries:              4,
			serverRetriesBeforeSuccess: 3,
			expectedRequestCount:       4,
			expectedDuration:           7 * time.Second, // 1 + 2 + 4 seconds
			wantErr:                    false,
		},
	}

	for _, test := range tests {
		retryCount = test.clientRetries
		retryCounter = test.serverRetriesBeforeSuccess
		endpoint = server.URL + "/" + test.responseCode
		startTime := time.Now()

		// reset server counters & input buffer
		requestCount = 0
		OsStdIn = bytes.NewBufferString(input)
		if err := execute(); (err != nil) != test.wantErr {
			t.Errorf("unexpected error in '%v': %v", test.name, err)
		}
		if requestCount != test.expectedRequestCount {
			t.Errorf("unexpected request count in '%v': %v", test.name, requestCount)
		}

		if !test.wantErr {
			duration := time.Since(startTime)
			t.Log("computed duration of ", duration)
			if duration < test.expectedDuration {
				t.Errorf("shorter wait than expected in '%v': %v", test.name, duration)
			}
		}
	}
}
