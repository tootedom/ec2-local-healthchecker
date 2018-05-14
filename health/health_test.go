package health

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tootedom/ec2-local-healthchecker/checks"
)

// This tests GET request with passing in a parameter.
func TestGracePeriod(t *testing.T) {

	// server returning 200s
	helloHandler := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello")
	}
	// create test server with handler
	ts := httptest.NewServer(http.HandlerFunc(helloHandler))
	defer ts.Close()

	// Check that is expecting 500 responses
	defaultRegistry := NewRegistry()
	defaultRegistry.Register("failing", PeriodicThresholdChecker(checks.HTTPChecker(ts.URL, 500, time.Second*1, nil), time.Second*1, 3, time.Second*10))

	time.Sleep(5 * time.Second)

	checksFailed := defaultRegistry.CheckStatus()

	assert.True(t, len(checksFailed) == 0)
}

func TestNoGracePeriod(t *testing.T) {

	// server returning 200s
	helloHandler := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello")
	}
	// create test server with handler
	ts := httptest.NewServer(http.HandlerFunc(helloHandler))
	defer ts.Close()

	// Check that is expecting 500 responses
	defaultRegistry := NewRegistry()
	defaultRegistry.Register("failing", PeriodicThresholdChecker(checks.HTTPChecker(ts.URL, 500, time.Second*1, nil), time.Second*1, 3, time.Second*0))

	time.Sleep(5 * time.Second)

	checksFailed := defaultRegistry.CheckStatus()

	assert.True(t, len(checksFailed) == 1)
}
