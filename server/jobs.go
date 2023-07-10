package server

import (
	"fmt"
	"sync"
)

// job - Manages background jobs
type job struct {
	ID          string
	Name        string
	Description string
	Protocol    string
	JobCtrl     chan bool
}

// jobs - Holds refs to all active jobs
type jobs struct {
	active *sync.Map
}

func newJobs() *jobs {
	return &jobs{
		active: &sync.Map{},
	}
}

func (ts *Server) addListener(id, host string, port int, ln Handler[any]) {
	log := ts.NamedLogger("teamserver", "listeners")

	if id == "" {
		id = getRandomID()
	}

	listener := &job{
		ID:          id,
		Name:        ln.Name(),
		Description: fmt.Sprintf("%s:%d", host, port),
		JobCtrl:     make(chan bool),
	}

	go func() {
		<-listener.JobCtrl

		// Kills listener goroutines but NOT connections.
		log.Infof("Stopping teamserver %s listener (%s)", ln.Name(), listener.ID)
		ln.Close()

		ts.jobs.active.LoadAndDelete(listener.ID)
	}()

	ts.jobs.active.Store(listener.ID, listener)
}

// CloseListener closes/stops an active teamserver listener.
func (ts *Server) CloseListener(id string) error {
	listener := ts.jobs.Get(id)
	if listener == nil {
		return fmt.Errorf("no listener exists with ID %s", id)
	}

	listener.JobCtrl <- true

	return nil
}

// All - Return a list of all jobs
func (j *jobs) All() []*job {
	all := []*job{}
	j.active.Range(func(key, value interface{}) bool {
		all = append(all, value.(*job))
		return true
	})
	return all
}

// Add - Add a job to the hive (atomically)
func (j *jobs) Add(listener *job) {
	j.active.Store(listener.ID, listener)
}

// Get - Get a Job
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
