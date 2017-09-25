package fetch

import (
	"github.com/open-horizon/horizon-pkg-fetch/horizonpkg"
	"net/http"
)

const (
	maxCancelationsBuffer = 100
	maxFetchesBuffer      = 100
)

// FetchPool is a type that specifies the configuration of FetchTasks.
type FetchPool struct {
	DestinationDirectory string // DestinationDirectory needs to be an environment variable; it can't be changed during execution
	HTTPClientProducer   func(domain string) *http.Client
	FetchBuffer          chan *FetchTask
	CancelationBuffer    chan *FetchCancelation
}

// EnqueueFetch takes a ??? TODO ???, generates a FetchTask and enqueues it for
// fetching.
func (pool *FetchPool) EnqueueFetch(fetchTask *FetchTask) error {
	return nil
}

// CancelFetch takes a cancelation instance and applies it to enqueued
// FetchTasks.
func (pool *FetchPool) CancelFetch(fetchCancelation *FetchCancelation) error {
	return nil
}

// NewFetchPool configures and returns an instantiation of a FetchPool or an
// error. The given HTTPClient producer function is expected to produce a
// configured HTTPClient that can authenticate with the given domain).
func NewFetchPool(destDir string, clientProducer func(domain string) *http.Client) (*FetchPool, error) {

	// bounded, async channels
	pool := &FetchPool{
		DestinationDirectory: destDir,
		HTTPClientProducer:   clientProducer,
		FetchBuffer:          make(chan *FetchTask, maxFetchesBuffer),
		CancelationBuffer:    make(chan *FetchCancelation, maxCancelationsBuffer),
	}

	return pool, nil
}

// FetchQueueProcessor consumes enqueued FetchTasks if:
// 0. Canceling a FetchTask (by setting CanceledOn timestamp), then dropping it
// 1. Instantiating a new HTTPClient from the Pool's factory *just before* a
// fetch attempt and then trying the fetch.
// 2. Adding a new FetchTry to the FetchTask's FetchTryHistory w/ success or
// failure.
//
// A FetchTask can be canceled before download (TODO: implement during-fetch
// cancelation too, later). That amounts to consuming a cancelation message on
// the cancelation queue and then attaching it to a FetchTask
type FetchQueueProcessor struct {
}

// FetchTask is a job to track fetching a particular fetchable unit by a
// FetchWorker.
type FetchTask struct {
	DestinationPath string // the identifier for the FetchTask
	Cancelation     FetchCancelation
	FetchTryHistory []FetchTry
	Pkg             *horizonpkg.Pkg
}

// FetchTry is a historical record of a fetch attempt, either successful or
// failed.
type FetchTry struct {
	FetchSuccess   bool
	FetchMsg       string
	FetchErr       error
	FetchStartedAt int
}

// FetchCancelation indicates a cancelation of an enqueued FetchTask
type FetchCancelation struct {
	DestinationPath               string // the identifier for the cancelation that matches the FetchTask
	CanceledOn                    uint
	CanceledBy                    string // loose description of the cancel request agent
	CancelationRequestSubmittedAt uint   // timestamp indicating when cancelation was submitted; the pool will expire requests to cancel after a certain duration
}
