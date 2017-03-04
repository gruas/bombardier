package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/goware/urlx"
	"github.com/valyala/fasthttp"
)

type config struct {
	numConns                             uint64
	numReqs                              *uint64
	duration                             *time.Duration
	url, method, body, certPath, keyPath string
	headers                              *headersList
	timeout                              time.Duration
	printLatencies, insecure             bool
	rate                                 *uint64
}

type testTyp int

const (
	none testTyp = iota
	timed
	counted
)

type invalidHTTPMethodError struct {
	method string
}

func (i *invalidHTTPMethodError) Error() string {
	return fmt.Sprintf("Unknown HTTP method: %v", i.method)
}

func (c *config) checkArgs() error {
	c.checkOrSetDefaultTestType()

	checks := []func() error{
		c.checkURL,
		c.checkRate,
		c.checkRunParameters,
		c.checkTimeoutDuration,
		c.checkHTTPParameters,
		c.checkCertPaths,
	}

	for _, check := range checks {
		if err := check(); err != nil {
			return err
		}
	}

	return nil
}

func (c *config) checkOrSetDefaultTestType() {
	if c.testType() == none {
		c.duration = &defaultTestDuration
	}
}

func (c *config) testType() testTyp {
	typ := none
	if c.numReqs != nil {
		typ = counted
	} else if c.duration != nil {
		typ = timed
	}
	return typ
}

func (c *config) checkURL() error {
	url, err := urlx.Parse(c.url)
	if err != nil {
		return err
	}
	if url.Host == "" || (url.Scheme != "http" && url.Scheme != "https") {
		return errInvalidURL
	}
	c.url = url.String()
	return nil
}

func (c *config) checkRate() error {
	if c.rate != nil && *c.rate < 1 {
		return errZeroRate
	}
	return nil
}

func (c *config) checkRunParameters() error {
	if c.numConns < uint64(1) {
		return errInvalidNumberOfConns
	}
	if c.testType() == counted && *c.numReqs < uint64(1) {
		return errInvalidNumberOfRequests
	}
	if c.testType() == timed && *c.duration < time.Second {
		return errInvalidTestDuration
	}
	return nil
}

func (c *config) checkTimeoutDuration() error {
	if c.timeout < 0 {
		return errNegativeTimeout
	}
	if c.timeout > 10*time.Second {
		return errLargeTimeout
	}
	return nil
}

func (c *config) checkHTTPParameters() error {
	if !allowedHTTPMethod(c.method) {
		return &invalidHTTPMethodError{method: c.method}
	}
	if !canHaveBody(c.method) && len(c.body) > 0 {
		return errBodyNotAllowed
	}
	return nil
}

func (c *config) checkCertPaths() error {
	if c.certPath != "" && c.keyPath == "" {
		return errNoPathToKey
	} else if c.certPath == "" && c.keyPath != "" {
		return errNoPathToCert
	}
	return nil
}

func (c *config) timeoutMillis() uint64 {
	return uint64(c.timeout.Nanoseconds() / 1000)
}

func (c *config) requestHeaders() *fasthttp.RequestHeader {
	return c.headers.toRequestHeader()
}

func allowedHTTPMethod(method string) bool {
	i := sort.SearchStrings(httpMethods, method)
	return i < len(httpMethods) && httpMethods[i] == method
}

func canHaveBody(method string) bool {
	i := sort.SearchStrings(cantHaveBody, method)
	return !(i < len(cantHaveBody) && cantHaveBody[i] == method)
}
