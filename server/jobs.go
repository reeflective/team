package server

import (
	"errors"
	"fmt"
	"sync"
)

// job - Manages background jobs.
type job struct {
	ID          string
	Name        string
	Description string
	kill        chan bool
	Persistent  bool
}

// jobs - Holds refs to all active jobs.
type jobs struct {
	active *sync.Map
}

func newJobs() *jobs {
	return &jobs{
		active: &sync.Map{},
	}
}

// Add - Add a job to the hive (atomically).
func (j *jobs) Add(listener *job) {
	j.active.Store(listener.ID, listener)
}

// Get - Get a Job.
func (j *jobs) Get(jobID string) *job {
	if jobID == "" {
		return nil
	}

	val, ok := j.active.Load(jobID)
	if ok {
		return val.(*job)
	}

	return nil
}

// Listeners - Return a list of all jobs.
func (ts *Server) Listeners() []*job {
	all := []*job{}

	// Active listeners
	ts.jobs.active.Range(func(key, value interface{}) bool {
		all = append(all, value.(*job))
		return true
	})

	return all
}

// AddListenerJob adds a teamserver listener job to the config and saves it.
func (ts *Server) AddListener(name, host string, port uint16) error {
	listener := struct {
		Name string `json:"name"`
		Host string `json:"host"`
		Port uint16 `json:"port"`
		ID   string `json:"id"`
	}{
		Name: name,
		Host: host,
		Port: port,
		ID:   getRandomID(),
	}

	ts.opts.config.Listeners = append(ts.opts.config.Listeners, listener)

	return ts.SaveConfig(ts.opts.config)
}

// RemoveListenerJob removes a server listener job from the configuration and saves it.
func (ts *Server) RemoveListener(listenerID string) {
	if ts.opts.config.Listeners == nil {
		return
	}

	defer ts.SaveConfig(ts.opts.config)

	var listeners []struct {
		Name string `json:"name"`
		Host string `json:"host"`
		Port uint16 `json:"port"`
		ID   string `json:"id"`
	}

	for _, listener := range ts.opts.config.Listeners {
		if listener.ID != listenerID {
			listeners = append(listeners, listener)
		}
	}

	ts.opts.config.Listeners = listeners
}

// CloseListener closes/stops an active teamserver listener.
func (ts *Server) CloseListener(id string) error {
	listener := ts.jobs.Get(id)
	if listener == nil {
		return ts.errorf("%w: %s", ErrListenerNotFound, id)
	}

	listener.kill <- true

	return nil
}

func (ts *Server) StartPersistentListeners(continueOnError bool) error {
	var listenerErrors error

	log := ts.NamedLogger("teamserver", "listeners")

	if ts.opts.config.Listeners == nil {
		return nil
	}

	for _, ln := range ts.opts.config.Listeners {
		handler := ts.handlers[ln.Name]
		if handler == nil {
			handler = ts.self
		}

		if handler == nil {
			if !continueOnError {
				return ts.errorf("Failed to find handler for `%s` listener (%s:%d)", ln.Name, ln.Host, ln.Port)
			}

			continue
		}

		err := ts.ServeHandler(handler, ln.ID, ln.Host, ln.Port)

		if err == nil {
			continue
		}

		log.Errorf("Failed to start %s listener (%s:%d): %s", ln.Name, ln.Host, ln.Port, err)

		if !continueOnError {
			return err
		}

		listenerErrors = errors.Join(listenerErrors, err)
	}

	return nil
}

func (ts *Server) addListenerJob(listenerID, host string, port int, ln Handler[any]) {
	log := ts.NamedLogger("teamserver", "listeners")

	if listenerID == "" {
		listenerID = getRandomID()
	}

	listener := &job{
		ID:          listenerID,
		Name:        ln.Name(),
		Description: fmt.Sprintf("%s:%d", host, port),
		kill:        make(chan bool),
	}

	go func() {
		<-listener.kill

		// Kills listener goroutines but NOT connections.
		log.Infof("Stopping teamserver %s listener (%s)", ln.Name(), listener.ID)
		ln.Close()

		ts.jobs.active.LoadAndDelete(listener.ID)
	}()

	ts.jobs.active.Store(listener.ID, listener)
}
