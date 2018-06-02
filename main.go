//
// Copyright [2018] [Dominic Tootell]
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
//

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/cloudfoundry/gosigar"
	"github.com/robfig/cron"
	"github.com/takama/daemon"
	"github.com/tevino/abool"
	"github.com/tootedom/ec2-local-healthchecker/checks"
	"github.com/tootedom/ec2-local-healthchecker/config"
	"github.com/tootedom/ec2-local-healthchecker/health"
)

const (
	// name of the service
	name        = "ec2-local-healthchecker"
	description = "ec2-local-healthchecker"
)

type UptimeCalc func() int64

var stdlog, errlog *log.Logger

// Service is the daemon service struct
type Service struct {
	daemon.Daemon
}

func registerInstanceAsUnhealthy() {
	if instanceIsHealthy.IsSet() {
		sess, err := session.NewSession(&aws.Config{Credentials: creds, Region: aws.String(region)})
		if err == nil {
			asg := autoscaling.New(sess, aws.NewConfig().WithRegion(region))
			input := autoscaling.SetInstanceHealthInput{HealthStatus: aws.String("Unhealthy"), InstanceId: aws.String(instanceID)}
			_, err := asg.SetInstanceHealth(&input)

			if err != nil {
				errlog.Printf("Unable to set instance(%s) as Unhealthy: %v", instanceID, err)
				errlog.Println()
			} else {
				errlog.Println("Marked Instance as Unhealthy")
				instanceIsHealthy.UnSet()
			}
		} else {
			errlog.Println("Unable to create a AWS Session", err)
		}

	}
}

func registerInstanceAsHealthy() {
	if !instanceIsHealthy.IsSet() {
		sess, err := session.NewSession(&aws.Config{Credentials: creds, Region: aws.String(region)})
		if err == nil {
			asg := autoscaling.New(sess, aws.NewConfig().WithRegion(region))
			input := autoscaling.SetInstanceHealthInput{HealthStatus: aws.String("Healthy"), InstanceId: aws.String(instanceID)}
			_, err := asg.SetInstanceHealth(&input)

			if err != nil {
				errlog.Printf("Unable to set instance(%s) as Healthy: %v", instanceID, err)
				errlog.Println()
			} else {
				errlog.Println("Marked Instance as Healthy")
				instanceIsHealthy.Set()
			}
		} else {
			errlog.Println("Unable to create a AWS Session", err)
		}
	}
}

func checkChecks() {
	if len(defaultRegistry.CheckStatus()) > 0 {
		errlog.Println("Health check failure")
		registerInstanceAsUnhealthy()
	} else {
		errlog.Println("Health check success")
		registerInstanceAsHealthy()
	}
}

func WaitForGracePeriod(gracePeriod time.Duration, uptime UptimeCalc) {
	for {
		if int64(gracePeriod.Seconds()) < uptime() {
			break
		}
		time.Sleep(5 * time.Second)
	}
}

func createHealthCheck(gracePeriod time.Duration, uptime UptimeCalc) func() {
	return func() {
		if gracePeriodOver.IsSet() {
			checkChecks()
		} else {
			if int64(gracePeriod.Seconds()) < uptime() {
				gracePeriodOver.Set()
				checkChecks()
			}
		}
	}
}

// Manage by daemon commands or run the daemon
func (service *Service) Manage(conf config.Config, command string, uptimeCalculationFunction UptimeCalc) (string, error) {

	switch command {
	case "install":
		return service.Install()
	case "remove":
		return service.Remove()
	case "start":
		return service.Start()
	case "stop":
		// No need to explicitly stop cron since job will be killed
		return service.Stop()
	case "status":
		return service.Status()
	}

	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, os.Kill, syscall.SIGTERM)

	CreateChecks(conf)
	// Create a new cron manager
	c := cron.New()
	// Run makefile every min
	c.AddFunc(fmt.Sprintf("@every %s", conf.Frequency), createHealthCheck(conf.GracePeriod, uptimeCalculationFunction))
	c.Start()
	// Waiting for interrupt by system signal
	killSignal := <-interrupt
	stdlog.Printf("Signal Received(%v) to exit.  Shutting Down....", killSignal)
	return "Service exited", nil
}

var region string
var instanceID string
var defaultRegistry *health.Registry
var creds *credentials.Credentials
var instanceIsHealthy *abool.AtomicBool
var gracePeriodOver *abool.AtomicBool

func CreateChecks(conf config.Config) {
	defaultRegistry = health.NewRegistry()
	for checkName, check := range conf.Checks {
		var checker checks.Checker
		if strings.ToLower(check.Type) == "tcp" {
			checker = checks.TCPChecker(check.Endpoint, check.Timeout)
		} else {
			checker = checks.HTTPChecker(check.Endpoint, 200, check.Timeout, nil)
		}

		defaultRegistry.Register(checkName, health.PeriodicThresholdChecker(checker, check.Frequency, check.Threshold))
	}
}

func init() {
	stdlog = log.New(os.Stdout, "", log.Ldate|log.Ltime)
	errlog = log.New(os.Stderr, "", log.Ldate|log.Ltime)
}

func CreateUpdateCalculationFunction(launchtime int64) UptimeCalc {
	if launchtime < 0 {
		return func() int64 {
			uptime := sigar.Uptime{}
			uptime.Get()
			uptimeInSeconds := uptime.Length
			return int64(uptimeInSeconds)
		}
	} else {
		return func() int64 {
			now := time.Now()
			secs := now.Unix()
			return secs - launchtime
		}
	}
}

func CalculateMaxCheckWaitTime(checks map[string]config.Check) int {
	maxTime := 0
	for _, check := range checks {
		seconds := int(check.Frequency.Seconds()) * int(check.Threshold)
		if seconds > maxTime {
			maxTime = seconds
		}
	}
	return maxTime
}

func main() {
	instanceIsHealthy = abool.NewBool(true)
	gracePeriodOver = abool.NewBool(false)

	checkfilePtr := flag.String("healthcheckfile", "/etc/sysconfig/ec2-local-healthchecker.yml", "The location of healthchecks yaml file")
	testConfigPtr := flag.Bool("testconfig", false, "test the healthcheck file is parseable")
	foregroundPtr := flag.Bool("foreground", false, "run the healthchecks in the foreground, exiting with nonzero if checks fail")
	launchTime := flag.Int64("launchtime", -1, "The launch time of the server that is running")
	commandPtr := flag.String("command", "", "The command to run")

	flag.Parse()
	runInForeground := *foregroundPtr
	uptimeCalculationFunction := CreateUpdateCalculationFunction(*launchTime)
	checkfile := *checkfilePtr

	instanceID := os.Getenv("INSTANCE_ID")
	region := os.Getenv("AWS_REGION")

	sess := session.Must(session.NewSession(&aws.Config{}))
	svc := ec2metadata.New(sess)

	if len(instanceID) == 0 || len(region) == 0 {
		// Create a EC2Metadata client from just a session.
		doc, err := svc.GetInstanceIdentityDocument()
		if err == nil {
			instanceID = doc.InstanceID
			region = doc.Region
		} else {
			errlog.Println("Unable to obtain instance id and region", err)
			os.Exit(1)
		}
	}

	creds = credentials.NewChainCredentials(
		[]credentials.Provider{
			&credentials.EnvProvider{},
			&ec2rolecreds.EC2RoleProvider{
				Client: svc,
			},
		})

	conf, err := config.Load(checkfile)

	if err != nil {
		errlog.Println("Error Parsing Configuration File: ", err)
		os.Exit(1)
	}

	if *testConfigPtr {
		stdlog.Println("Parsed Configuration:")
		stdlog.Println(conf.Checks)
		os.Exit(1)
	}

	if runInForeground {
		// Start the checks running
		CreateChecks(*conf)
		startTime := time.Now().Unix()
		timeToWait := CalculateMaxCheckWaitTime(conf.Checks) + 1
		// Do not check the result until grace is over
		WaitForGracePeriod(conf.GracePeriod, uptimeCalculationFunction)
		endTime := time.Now().Unix()
		waited := endTime - startTime
		extra := timeToWait - int(waited)
		if extra > 0 {
			time.Sleep(time.Duration(extra) * time.Second)
		}

		if len(defaultRegistry.CheckStatus()) > 0 {
			os.Exit(1)
		} else {
			os.Exit(0)
		}

	} else {

		srv, err := daemon.New(name, description)
		if err != nil {
			errlog.Println("Error: ", err)
			os.Exit(1)
		}
		service := &Service{srv}
		status, err := service.Manage(*conf, *commandPtr, uptimeCalculationFunction)
		if err != nil {
			errlog.Println(status, "\nError: ", err)
			os.Exit(1)
		}
		fmt.Println(status)
	}
}
