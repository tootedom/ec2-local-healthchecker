package health

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tootedom/ec2-local-healthchecker/checks"
)

// This tests GET request with passing in a parameter.
func TestConsecutiveSuccess(t *testing.T) {

	// server returning 200s
	helloHandler := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello")
	}
	// create test server with handler
	ts := httptest.NewServer(http.HandlerFunc(helloHandler))
	defer ts.Close()

	// Check that is expecting 500 responses
	defaultRegistry := NewRegistry()
	defaultRegistry.Register("failing", PeriodicThresholdChecker(checks.HTTPChecker(ts.URL, 200, time.Second*1, nil), time.Second*1, 10))

	time.Sleep(5 * time.Second)

	checksFailed := defaultRegistry.CheckStatus()

	assert.True(t, len(checksFailed) == 1)

	time.Sleep(10 * time.Second)

	checksFailed = defaultRegistry.CheckStatus()

	assert.True(t, len(checksFailed) == 0)
}

func TestConsecutiveFailures(t *testing.T) {
	var ops uint64

	// server returning 200s
	helloHandler := func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint64(&ops, 1)
		call := atomic.LoadUint64(&ops)
		if call > 5 {
			w.WriteHeader(200)
		} else {
			w.WriteHeader(500)
		}
		fmt.Fprint(w, "hello")
	}
	// create test server with handler
	ts := httptest.NewServer(http.HandlerFunc(helloHandler))
	defer ts.Close()

	// Check that is expecting 200 responses
	defaultRegistry := NewRegistry()
	defaultRegistry.Register("failing", PeriodicThresholdChecker(checks.HTTPChecker(ts.URL, 200, time.Second*1, nil), time.Second*1, 3))

	time.Sleep(5 * time.Second)

	checksFailed := defaultRegistry.CheckStatus()

	assert.True(t, len(checksFailed) == 1)

	checksFailed = defaultRegistry.CheckStatus()
	assert.True(t, len(checksFailed) == 1)
	time.Sleep(10 * time.Second)
	checksFailed = defaultRegistry.CheckStatus()

	assert.True(t, len(checksFailed) == 0)

}
