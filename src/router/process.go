package router

import (
	"fmt"
	"net"

	"rogchap.com/v8go"
)

var JobQueue chan Job

type Job struct {
	id     uint64
	shard  *net.Conn
	client *net.Conn
	msg    []byte // Job does not directly store bytes
}

type Worker struct {
	WorkerPool chan chan Job
	JobChannel chan Job
	quit       chan bool
}

func NewWorker(workerPool chan chan Job) Worker {
	return Worker{
		WorkerPool: workerPool,
		JobChannel: make(chan Job),
		quit:       make(chan bool)}
}

// Start method starts the run loop for the worker, listening for a quit channel in
// case we need to stop it
func (w Worker) Start() {
	go func() {
		iso, _ := v8go.NewIsolate()
		printfn, _ := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			fmt.Printf("%v", info.Args()) // when the JS function is called this Go callback will execute
			val, err := info.Context().RunScript("'hello'", "hi.js")
			if err != nil {
				panic(err)
			}
			return val // you can return a value back to the JS caller if required
		})
		global, _ := v8go.NewObjectTemplate(iso) // a template that represents a JS Object
		global.Set("print", printfn)             // sets the "print" property of the Object to our function
		for {
			// register the current worker into the worker queue.
			w.WorkerPool <- w.JobChannel

			select {
			case job := <-w.JobChannel:
				// do the work here
				jobHandler(job)

			case <-w.quit:
				// we have received a signal to stop
				return
			}
		}
	}()
}

// Stop signals the worker to stop listening for work requests.
func (w Worker) Stop() {
	go func() {
		w.quit <- true
	}()
}

type Dispatcher struct {
	// A pool of workers channels that are registered with the dispatcher
	WorkerPool chan chan Job
	maxWorkers uint32
}

func NewDispatcher(maxWorkers uint32) *Dispatcher {
	pool := make(chan chan Job, maxWorkers)
	return &Dispatcher{WorkerPool: pool, maxWorkers: maxWorkers}
}

func (d *Dispatcher) Run() {
	// starting n number of workers
	var i uint32 = 0
	for i < d.maxWorkers {
		worker := NewWorker(d.WorkerPool)
		worker.Start()
		i++
	}

	go d.dispatch()
}

func (d *Dispatcher) NewWorker() {
	worker := NewWorker(d.WorkerPool)
	worker.Start()
}

func (d *Dispatcher) dispatch() {
	for {
		select {
		case job := <-JobQueue:
			// a job request has been received
			go func(job Job) {
				// try to obtain a worker job channel that is available.
				// this will block until a worker is idle
				jobChannel := <-d.WorkerPool

				// dispatch the job to the worker job channel
				jobChannel <- job
			}(job)
		}
	}
}

func jobHandler(j Job) {

}
