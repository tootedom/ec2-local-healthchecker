package health

import (
	"sync"
	"time"

	"github.com/tootedom/ec2-local-healthchecker/checks"
)

// A Registry is a collection of checks. Most applications will use the global
// registry defined in DefaultRegistry. However, unit tests may need to create
// separate registries to isolate themselves from other tests.
type Registry struct {
	mu               sync.RWMutex
	registeredChecks map[string]checks.Checker
}

// NewRegistry creates a new registry. This isn't necessary for normal use of
// the package, but may be useful for unit tests so individual tests have their
// own set of checks.
func NewRegistry() *Registry {
	return &Registry{
		registeredChecks: make(map[string]checks.Checker),
	}
}

// DefaultRegistry is the default registry where checks are registered. It is
// the registry used by the HTTP handler.
var DefaultRegistry *Registry

// Updater implements a health check that is explicitly set.
type Updater interface {
	checks.Checker

	// Update updates the current status of the health check.
	Update(status error)
}

// updater implements Checker and Updater, providing an asynchronous Update
// method.
// This allows us to have a Checker that returns the Check() call immediately
// not blocking on a potentially expensive check.
type updater struct {
	mu     sync.Mutex
	status error
}

// Check implements the Checker interface
func (u *updater) Check() error {
	u.mu.Lock()
	defer u.mu.Unlock()

	return u.status
}

// thresholdUpdater implements Checker and Updater, providing an asynchronous Update
// method.
// This allows us to have a Checker that returns the Check() call immediately
// not blocking on a potentially expensive check.
type thresholdUpdater struct {
	mu          sync.Mutex
	status      error
	threshold   int
	count       int
	gracePeriod time.Duration
	created     time.Time
}

// Check implements the Checker interface
func (tu *thresholdUpdater) Check() error {
	tu.mu.Lock()
	defer tu.mu.Unlock()

	if tu.count >= tu.threshold {
		return tu.status
	}

	return nil
}

// thresholdUpdater implements the Updater interface, allowing asynchronous
// access to the status of a Checker.
func (tu *thresholdUpdater) Update(status error) {
	tu.mu.Lock()
	defer tu.mu.Unlock()
	t := time.Now()
	elapsed := t.Sub(tu.created)
	if tu.gracePeriod < elapsed {
		if status == nil {
			tu.count = 0
		} else if tu.count < tu.threshold {
			tu.count++
		}

		tu.status = status
	}
}

// NewThresholdStatusUpdater returns a new thresholdUpdater
func NewThresholdStatusUpdater(t int, grace time.Duration) Updater {
	return &thresholdUpdater{threshold: t, created: time.Now(), gracePeriod: grace}
}

// PeriodicThresholdChecker wraps an updater to provide a periodic checker that
// uses a threshold before it changes status
func PeriodicThresholdChecker(check checks.Checker, period time.Duration, threshold int, grace time.Duration) checks.Checker {
	tu := NewThresholdStatusUpdater(threshold, grace)
	go func() {
		t := time.NewTicker(period)
		for {
			<-t.C
			tu.Update(check.Check())
		}
	}()

	return tu
}

// CheckStatus returns a map with all the current health check errors
func (registry *Registry) CheckStatus() map[string]string { // TODO(stevvooe) this needs a proper type
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	statusKeys := make(map[string]string)
	for k, v := range registry.registeredChecks {
		err := v.Check()
		if err != nil {
			statusKeys[k] = err.Error()
		}
	}

	return statusKeys
}

// CheckStatus returns a map with all the current health check errors from the
// default registry.
func CheckStatus() map[string]string {
	return DefaultRegistry.CheckStatus()
}

// Register associates the checker with the provided name.
func (registry *Registry) Register(name string, check checks.Checker) {
	if registry == nil {
		registry = DefaultRegistry
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	_, ok := registry.registeredChecks[name]
	if ok {
		panic("Check already exists: " + name)
	}
	registry.registeredChecks[name] = check
}
