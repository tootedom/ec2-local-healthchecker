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

	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	"github.com/robfig/cron"
	"github.com/takama/daemon"
	"github.com/tootedom/ec2-local-healthchecker/checks"
	"github.com/tootedom/ec2-local-healthchecker/config"
	"github.com/tootedom/ec2-local-healthchecker/health"
)

const (
	// name of the service
	name        = "cron_job"
	description = "Cron job service example"
)

var stdlog, errlog *log.Logger

// Service is the daemon service struct
type Service struct {
	daemon.Daemon
}

func registerInstanceAsUnhealthy(asg autoscalingiface.AutoScalingAPI) {
}

func createHealthCheck(started time.Time, gracePeriod time.Duration) func() {
	return func() {
		t := time.Now()
		elapsed := t.Sub(started)
		if gracePeriod < elapsed {
			// create a simple file (current time).txt
			// f, err := os.Create(fmt.Sprintf("%s/%s.txt", os.TempDir(), time.Now().Format(time.RFC3339)))
			fmt.Println("created file:" + time.Now().Format(time.RFC3339))
			fmt.Println(defaultRegistry.CheckStatus())
			if len(defaultRegistry.CheckStatus()) > 0 {
				fmt.Println("failed")
			} else {
				fmt.Println("ok")
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
	stdlog.Println("Got signal:", killSignal)
	return "Service exited", nil
}

var defaultRegistry *health.Registry

func createChecks(conf config.Config) {
	defaultRegistry = health.NewRegistry()
	for checkName, check := range conf.Checks {
		var checker health.Checker
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

func main() {
	checkfilePtr := flag.String("healthcheckfile", "/etc/sysconfig/healthchecks.yml", "The location of healthchecks yaml file")
	testConfigPtr := flag.Bool("testconfig", false, "test the healthcheck file is parseable")
	commandPtr := flag.String("command", "", "The command to run")

	flag.Parse()

	checkfile := *checkfilePtr

	conf, err := config.Load(checkfile)

	if err != nil {
		errlog.Println("Error Parsing Configuration File: ", err)
		os.Exit(1)
	}

	if *testConfigPtr {
		fmt.Println("Parsed Configuration:")
		fmt.Println(conf.Checks)
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
