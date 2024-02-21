// Package prometheus ...
// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package prometheus

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	prom "github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

// MetricFamiliesByName is a map of Prometheus metrics family names and their
// representation.
type MetricFamiliesByName map[string]dto.MetricFamily

// HTTPDoer executes http requests. It is implemented by *http.Client.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// ResetTotalScrapedPayload resets the integration totalScrapedPayload
// metric.
func ResetTotalScrapedPayload() {
	totalScrapedPayload.Set(0)
}

// ResetTargetSize resets the integration targetSize
// metric.
func ResetTargetSize() {
	targetSize.Reset()
}

const (
	// XPrometheusScrapeTimeoutHeader included in all requests. It informs exporters about its timeout.
	XPrometheusScrapeTimeoutHeader = "X-Prometheus-Scrape-Timeout-Seconds"
	// AcceptHeader included in all requests
	AcceptHeader = "Accept"
)

// Get scrapes the given URL and decodes the retrieved payload.
func Get(client HTTPDoer, url string, acceptHeader string, fetchTimeout string) (MetricFamiliesByName, error) {
	mfs := MetricFamiliesByName{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return mfs, err
	}

	req.Header.Add(AcceptHeader, acceptHeader)
	req.Header.Add(XPrometheusScrapeTimeoutHeader, fetchTimeout)

	resp, err := client.Do(req)
	if err != nil {
		return mfs, err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 300 {
		return nil, fmt.Errorf("status code returned by the prometheus exporter indicates an error occurred: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return mfs, err
	}
	r := bytes.NewReader(body)

	d := expfmt.NewDecoder(r, expfmt.FmtText)
	for {
		var mf dto.MetricFamily
		if err := d.Decode(&mf); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		mfs[mf.GetName()] = mf
	}

	bodySize := float64(len(body))
	targetSize.With(prom.Labels{"target": url}).Set(bodySize)
	totalScrapedPayload.Add(bodySize)
	return mfs, nil
}
