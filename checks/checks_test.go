package checks

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// This tests GET request with passing in a parameter.
func TestHttpChecker(t *testing.T) {

	// echoHandler, passes back form parameter p
	helloHandler := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello")
	}

	// create test server with handler
	ts := httptest.NewServer(http.HandlerFunc(helloHandler))
	defer ts.Close()

	check := HTTPChecker(ts.URL, 200, time.Second*1, nil)
	assert.Equal(t, nil, check.Check())

	check = HTTPChecker(ts.URL, 500, time.Second*1, nil)
	assert.True(t, check.Check() != nil)
}

func TestTCPChecker(t *testing.T) {

	// echoHandler, passes back form parameter p
	helloHandler := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello")
	}

	// create test server with handler
	ts := httptest.NewServer(http.HandlerFunc(helloHandler))
	defer ts.Close()

	check := TCPChecker(strings.Replace(ts.URL, "http://", "", 1), time.Second*1)
	assert.Equal(t, nil, check.Check())

}
