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

	calcFunc := CreateUpdateCalculationFunction(time.Now().Unix(), 10*time.Second)
	start := time.Now().Unix()
	WaitForGracePeriod(10*time.Second, calcFunc, false)
	end := time.Now().Unix()
	diff := end - start
	assert.True(t, diff >= 10)

}

func TestCalculateMaxCheckWaitTime(t *testing.T) {
	conf := config.Config{
		GracePeriod: 10 * time.Second,
		Frequency:   2 * time.Second,
		Checks: map[string]config.Check{
			"http": config.Check{
				Threshold: 2,
				Endpoint:  "",
				Timeout:   1 * time.Second,
				Frequency: 10 * time.Second,
				Type:      "http",
			},
			"tcp": config.Check{
				Threshold: 4,
				Endpoint:  "",
				Timeout:   2 * time.Second,
				Frequency: 2 * time.Second,
				Type:      "http",
			},
		},
	}

	assert.True(t, CalculateMaxCheckWaitTime(conf.Checks) == 20)
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

	calcFunc := CreateUpdateCalculationFunction(time.Now().Unix(), 10*time.Second)
	start := time.Now().Unix()
	WaitForGracePeriod(10*time.Second, calcFunc, false)
	end := time.Now().Unix()
	diff := end - start
	assert.True(t, diff < 20)
	assert.True(t, diff >= 10)

	fmt.Println(len(defaultRegistry.CheckStatus()))
	assert.True(t, len(defaultRegistry.CheckStatus()) == 0)

}

func TestChecksDuringGraceAndExitsEarlyIfHealthy(t *testing.T) {

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

	calcFunc := CreateUpdateCalculationFunction(time.Now().Unix(), 10*time.Second)
	start := time.Now().Unix()
	healthy := WaitForGracePeriod(10*time.Second, calcFunc, true)
	end := time.Now().Unix()
	diff := end - start
	assert.True(t, diff <= 10)

	fmt.Println(len(defaultRegistry.CheckStatus()))
	assert.True(t, healthy)
	assert.True(t, len(defaultRegistry.CheckStatus()) == 0)

}

func TestChecksDuringGraceAndExitsEarlyIfHealthyAndPriorCurrentEpoch(t *testing.T) {

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

	calcFunc := CreateUpdateCalculationFunction(time.Now().Unix()-1000, 10*time.Second)
	start := time.Now().Unix()
	healthy := WaitForGracePeriod(10*time.Second, calcFunc, true)
	fmt.Println(healthy)
	end := time.Now().Unix()
	diff := end - start
	assert.True(t, diff <= 10)

	fmt.Println(len(defaultRegistry.CheckStatus()))
	assert.True(t, healthy)
	assert.True(t, len(defaultRegistry.CheckStatus()) == 0)

}

func TestChecksHealthBeforeGracePeriod(t *testing.T) {

	// echoHandler, passes back form parameter p
	helloHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}

	// create test server with handler
	ts := httptest.NewServer(http.HandlerFunc(helloHandler))
	defer ts.Close()

	CreateChecks(config.Config{
		GracePeriod: 10 * time.Second,
		Frequency:   2 * time.Second,
		Checks: map[string]config.Check{
			"http": config.Check{
				Threshold: 5,
				Endpoint:  ts.URL,
				Timeout:   1 * time.Second,
				Frequency: 2 * time.Second,
				Type:      "http",
			},
		},
	})

	fmt.Println(len(defaultRegistry.CheckStatus()))
	assert.True(t, len(defaultRegistry.CheckStatus()) == 0)

	time.Sleep(15 * time.Second)
	assert.True(t, len(defaultRegistry.CheckStatus()) == 1)
}
