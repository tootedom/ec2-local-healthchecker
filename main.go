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
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
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

var stdlog, errlog *log.Logger

// Service is the daemon service struct
type Service struct {
	daemon.Daemon
}

func registerInstanceAsUnhealthy(asg autoscalingiface.AutoScalingAPI) {
	if instanceIsHealthy.IsSet() {
		input := autoscaling.SetInstanceHealthInput{HealthStatus: aws.String("Unhealthy"), InstanceId: aws.String(instanceId)}
		_, err := asg.SetInstanceHealth(&input)

		if err != nil {
			errlog.Printf("Unabled to set instance(%s) as Unhealthy: %v", instanceId, err)
			errlog.Println()
		} else {
			errlog.Println("Marked Instance as Unhealthy")
			instanceIsHealthy.UnSet()
		}
	}
}

func createHealthCheck(started time.Time, gracePeriod time.Duration) func() {
	return func() {
		t := time.Now()
		elapsed := t.Sub(started)
		if gracePeriod < elapsed {
			// create a simple file (current time).txt
			if len(defaultRegistry.CheckStatus()) > 0 {
				sess, err := session.NewSession(&aws.Config{Credentials: creds, Region: aws.String(region)})
				if err == nil {
					registerInstanceAsUnhealthy(autoscaling.New(sess, aws.NewConfig().WithRegion(region)))
				} else {
					errlog.Println("Unable to create a AWS Session", err)
				}
			}
		}
	}
}

// Manage by daemon commands or run the daemon
func (service *Service) Manage(conf config.Config, command string) (string, error) {

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

	startedCron := time.Now()
	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, os.Kill, syscall.SIGTERM)

	createChecks(conf)
	// Create a new cron manager
	c := cron.New()
	// Run makefile every min
	c.AddFunc(fmt.Sprintf("@every %s", conf.Frequency), createHealthCheck(startedCron, conf.GracePeriod))
	c.Start()
	// Waiting for interrupt by system signal
	killSignal := <-interrupt
	stdlog.Printf("Signal Received(%v) to exit.  Shutting Down....", killSignal)
	return "Service exited", nil
}

var region string
var instanceId string
var defaultRegistry *health.Registry
var creds *credentials.Credentials
var instanceIsHealthy *abool.AtomicBool

func createChecks(conf config.Config) {
	defaultRegistry = health.NewRegistry()
	for checkName, check := range conf.Checks {
		var checker checks.Checker
		if strings.ToLower(check.Type) == "tcp" {
			checker = checks.TCPChecker(check.Endpoint, check.Timeout)
		} else {
			checker = checks.HTTPChecker(check.Endpoint, 200, check.Timeout, nil)
		}

		defaultRegistry.Register(checkName, health.PeriodicThresholdChecker(checker, check.Frequency, check.Threshold, conf.GracePeriod))
	}
}

func init() {
	stdlog = log.New(os.Stdout, "", log.Ldate|log.Ltime)
	errlog = log.New(os.Stderr, "", log.Ldate|log.Ltime)
}

func main() {
	instanceIsHealthy = abool.NewBool(true)

	checkfilePtr := flag.String("healthcheckfile", "/etc/sysconfig/ec2-local-healthchecker.yml", "The location of healthchecks yaml file")
	testConfigPtr := flag.Bool("testconfig", false, "test the healthcheck file is parseable")
	commandPtr := flag.String("command", "", "The command to run")

	instanceId = os.Getenv("INSTANCE_ID")
	region = os.Getenv("AWS_REGION")

	sess := session.Must(session.NewSession(&aws.Config{}))
	svc := ec2metadata.New(sess)

	if len(instanceId) == 0 || len(region) == 0 {
		// Create a EC2Metadata client from just a session.
		doc, err := svc.GetInstanceIdentityDocument()
		if err == nil {
			instanceId = doc.InstanceID
			region = doc.Region
		} else {
			errlog.Println("Unable to obtain instance id and region", err)
			os.Exit(1)
		}
	}

	flag.Parse()

	creds = credentials.NewChainCredentials(
		[]credentials.Provider{
			&credentials.EnvProvider{},
			&ec2rolecreds.EC2RoleProvider{
				Client: svc,
			},
		})

	checkfile := *checkfilePtr

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

	srv, err := daemon.New(name, description)
	if err != nil {
		errlog.Println("Error: ", err)
		os.Exit(1)
	}
	service := &Service{srv}
	status, err := service.Manage(*conf, *commandPtr)
	if err != nil {
		errlog.Println(status, "\nError: ", err)
		os.Exit(1)
	}
	fmt.Println(status)
}
