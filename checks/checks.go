// Taken from https://github.com/docker/go-healthcheck
// Copyright [yyyy] [name of copyright owner]

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

// 	http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package checks

import (
	"errors"
	"net"
	"net/http"
	"strconv"
	"time"
)

// Checker is the interface for a Health Checker
type Checker interface {
	// Check returns nil if the service is okay.
	Check() error
}

// CheckFunc is a convenience type to create functions that implement
// the Checker interface
type CheckFunc func() error

// Check Implements the Checker interface to allow for any func() error method
// to be passed as a Checker
func (cf CheckFunc) Check() error {
	return cf()
}

// HTTPChecker does a GET request and verifies that the HTTP status code
// returned matches statusCode.
func HTTPChecker(r string, statusCode int, timeout time.Duration, headers http.Header) Checker {
	return CheckFunc(func() error {
		client := http.Client{
			Timeout: timeout,
		}
		req, err := http.NewRequest("GET", r, nil)
		if err != nil {
			return errors.New("error creating request: " + r)
		}
		for headerName, headerValues := range headers {
			for _, headerValue := range headerValues {
				req.Header.Add(headerName, headerValue)
			}
		}
		response, err := client.Do(req)
		if err != nil {
			return errors.New("error while checking: " + r)
		}
		defer response.Body.Close()
		if response.StatusCode != statusCode {
			return errors.New("downstream service returned unexpected status: " + strconv.Itoa(response.StatusCode))
		}
		return nil
	})
}

// TCPChecker attempts to open a TCP connection.
func TCPChecker(addr string, timeout time.Duration) Checker {
	return CheckFunc(func() error {
		conn, err := net.DialTimeout("tcp", addr, timeout)
		if err != nil {
			return errors.New("connection to " + addr + " failed")
		}
		conn.Close()
		return nil
	})
}
