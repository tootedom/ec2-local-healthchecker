package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tootedom/ec2-local-healthchecker/config"
)

// This tests GET request with passing in a parameter.
func TestUptimeCalculator(t *testing.T) {

	calcFunc := CreateUpdateCalculationFunction(time.Now().Unix())
	start := time.Now().Unix()
	WaitForGracePeriod(10*time.Second, calcFunc)
	end := time.Now().Unix()
	diff := end - start
	assert.True(t, diff >= 10)

}

func TestChecksAfterGrace(t *testing.T) {

	// echoHandler, passes back form parameter p
	helloHandler := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello")
	}

	// create test server with handler
	ts := httptest.NewServer(http.HandlerFunc(helloHandler))
	defer ts.Close()

	CreateChecks(config.Config{
		GracePeriod: 10 * time.Second,
		Frequency:   2 * time.Second,
		Checks: map[string]config.Check{
			"http": config.Check{
				Threshold: 2,
				Endpoint:  ts.URL,
				Timeout:   1 * time.Second,
				Frequency: 2 * time.Second,
				Type:      "http",
			},
		},
	})

	calcFunc := CreateUpdateCalculationFunction(time.Now().Unix())
	start := time.Now().Unix()
	WaitForGracePeriod(10*time.Second, calcFunc)
	end := time.Now().Unix()
	diff := end - start
	assert.True(t, diff >= 10)

	fmt.Println(len(defaultRegistry.CheckStatus()))
	assert.True(t, len(defaultRegistry.CheckStatus()) == 0)

}
