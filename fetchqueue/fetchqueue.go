package fetchqueue

import (
	"github.com/open-horizon/horizon-pkg-fetch/horizonpkg"
	"net/http"
)

const (
	maxCancelationsBuffer = 100
	maxFetchesBuffer      = 100
)

// Pool is a type that specifies the configuration of Tasks.
type Pool struct {
	DestinationDirectory string // DestinationDirectory needs to be an environment variable; it can't be changed during execution
	HTTPClientProducer   func(domain string) *http.Client
	FetchBuffer          chan *Task
	CancelationBuffer    chan *Cancelation
}

// EnqueueFetch takes a ??? TODO ???, generates a Task and enqueues it for
// fetching.
func (pool *Pool) EnqueueFetch(Task *Task) error {
	return nil
}

// CancelFetch takes a cancelation instance and applies it to enqueued
// Tasks.
func (pool *Pool) CancelFetch(fetchCancelation *Cancelation) error {
	return nil
}

// NewPool configures and returns an instantiation of a Pool or an
// error. The given HTTPClient producer function is expected to produce a
// configured HTTPClient that can authenticate with the given domain).
func NewPool(destDir string, clientProducer func(domain string) *http.Client) (*Pool, error) {

	// bounded, async channels
	pool := &Pool{
		DestinationDirectory: destDir,
		HTTPClientProducer:   clientProducer,
		FetchBuffer:          make(chan *Task, maxFetchesBuffer),
		CancelationBuffer:    make(chan *Cancelation, maxCancelationsBuffer),
	}

	return pool, nil
}

// QueueProcessor consumes enqueued Tasks if:
// 0. Canceling a Task (by setting CanceledOn timestamp), then dropping it
// 1. Instantiating a new HTTPClient from the Pool's factory *just before* a
// fetch attempt and then trying the fetch.
// 2. Adding a new Try to the Task's TryHistory w/ success or
// failure.
//
// A Task can be canceled before download (TODO: implement during-fetch
// cancelation too, later). That amounts to consuming a cancelation message on
// the cancelation queue and then attaching it to a Task
type QueueProcessor struct {
}

// Task is a job to track fetching a particular fetchable unit by a
// FetchWorker.
type Task struct {
	DestinationPath string // the identifier for the Task
	Cancelation     Cancelation
	TryHistory      []Try
	Pkg             *horizonpkg.Pkg
}

// Try is a historical record of a fetch attempt, either successful or
// failed.
type Try struct {
	FetchSuccess   bool
	FetchMsg       string
	FetchErr       error
	FetchStartedAt int
}

// Cancelation indicates a cancelation of an enqueued Task
type Cancelation struct {
	DestinationPath               string // the identifier for the cancelation that matches the Task
	CanceledOn                    uint
	CanceledBy                    string // loose description of the cancel request agent
	CancelationRequestSubmittedAt uint   // timestamp indicating when cancelation was submitted; the pool will expire requests to cancel after a certain duration
}
