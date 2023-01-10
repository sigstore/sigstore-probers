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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	validate "github.com/go-playground/validator/v10"
	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/spf13/pflag"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type linkStruct struct {
	Href string `json:"href" validate:"required,url"`
	Text string `json:"text,omitempty"`
}

type inputStruct struct {
	Summary   string                 `json:"summary" validate:"required,max=1024"`
	Source    string                 `json:"source" validate:"required"`
	Severity  string                 `json:"severity" validate:"required,oneof=critical error warning info"`
	Component string                 `json:"component" validate:"required"`
	Group     string                 `json:"group" validate:"required"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Links     []linkStruct           `json:"links,omitempty" validate:"dive"`
}

var (
	retryCount uint
	validator  *validate.Validate

	// specified so we can override in unit tests
	endpoint string
	OsStdIn  io.Reader
)

func init() {
	pflag.UintVar(&retryCount, "retry-count", 5, "number of times to retry requests")
	pflag.Parse()

	// set up logging
	config := zap.NewDevelopmentConfig()
	config.DisableStacktrace = true                                     // don't print stacktraces
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder // outputs color to the terminal
	config.EncoderConfig.EncodeTime = zapcore.TimeEncoder(func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.UTC().Format("2006-01-02T15:04:05.000Z"))
	}) // sets time to UTC
	logger, _ := config.Build()
	zap.ReplaceGlobals(logger)

	validator = validate.New()

	// specified so we can override in unit tests
	endpoint = "https://events.pagerduty.com"
	OsStdIn = os.Stdin
}

func main() {
	if err := execute(); err != nil {
		zap.S().Fatal(err)
	}
}

func execute() error {
	integrationKey := os.Getenv("PAGERDUTY_INTEGRATION_KEY")
	if integrationKey == "" {
		return fmt.Errorf("no integration key provided, please set PAGERDUTY_INTEGRATION_KEY env var")
	}

	var input inputStruct
	// parse input
	err := json.NewDecoder(OsStdIn).Decode(&input)
	if err != nil {
		return fmt.Errorf("parsing JSON input: %w", err)
	}
	// validate input
	if err := validator.Struct(input); err != nil {
		return fmt.Errorf("validating JSON input: %w", err)
	}

	event := &pagerduty.V2Event{
		Payload: &pagerduty.V2Payload{
			Summary:   input.Summary,
			Source:    input.Source,
			Severity:  input.Severity,
			Component: input.Component,
			Group:     input.Group,
		},
		RoutingKey: integrationKey,
		Action:     "trigger",
	}
	if input.Details != nil {
		event.Payload.Details = input.Details
	}
	if len(input.Links) > 0 {
		listOfLinks := make([]interface{}, len(input.Links))
		for i, v := range input.Links {
			listOfLinks[i] = v
		}
		event.Links = listOfLinks
	}

	retryClient := retryablehttp.NewClient() // use HTTP client that respects Retry-After headers
	retryClient.Logger = nil                 // disable logging
	retryClient.RetryMax = int(retryCount)   // set maximum number of retries
	retryClient.RequestLogHook = func(_ retryablehttp.Logger, req *http.Request, retryNumber int) {
		zap.S().Debugf("attempt #%d: %s %s", retryNumber, req.Method, req.URL)
	}
	retryClient.ResponseLogHook = func(_ retryablehttp.Logger, resp *http.Response) {
		if resp.StatusCode != http.StatusAccepted {
			zap.S().Errorf("request failed: %s", resp.Status)
			body, _ := io.ReadAll(resp.Body)
			if len(body) > 0 {
				zap.S().Errorf("response body: %v", string(body))
			}
		}
	}

	// token is set to empty here since it is not used for the Events API v2
	pgClient := pagerduty.NewClient("", pagerduty.WithV2EventsAPIEndpoint(endpoint))
	// use the retryable client instead of the standard one to get exponential backoff, auto-retries
	pgClient.HTTPClient = retryClient.StandardClient()

	resp, err := pgClient.ManageEventWithContext(context.Background(), event)
	if err == nil {
		zap.S().Infof("successfully created event with dedup_key %s", resp.DedupKey)
		return nil
	}
	return fmt.Errorf("error communicating with PagerDuty: %w", err)
}
