package client_interface

import (
	"fmt"
	"strconv"

	"src/config"
	"src/javascriptAPI"
)

var JobQueue = make(chan Job)

var jobcount uint64 = 0

type Job struct {
	id     uint64
	client *Client_conn
	msg    []byte // Job does not directly store bytes
}

func NewJob(conn *Client_conn, msg []byte) {
	JobQueue <- Job{
		client: conn,
		msg:    msg,
	}
	if config.Debug {
		fmt.Println("job sent to job queue")
	}
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

		for {
			// register the current worker into the worker queue.
			w.WorkerPool <- w.JobChannel

			select {
			case job := <-w.JobChannel:
				if config.Debug {
					fmt.Println("worker recieve job")
				}
				// do the work here
				s, err := javascriptAPI.Javascript_run_isolate(string(job.msg), "",
					[2]string{"gid", strconv.Itoa(int(job.client.gid))}, [2]string{"uid", strconv.Itoa(int(job.client.id))})
				/*
					if err != nil {
						Send(*job.client, []byte(err.Error()))
						continue
					}
					d, err := yaegiAPI.Run(string(job.msg))
				*/

				if err != nil {
					if config.Debug {
						fmt.Println(err)
					}
					Send(*job.client, []byte(err.Error()))
				} else {
					Send(*job.client, []byte(s))
				}
				if config.Debug {
					fmt.Println("worker finish job")
				}

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
		job := <-JobQueue
		if config.Debug {
			fmt.Println("job queue recieve job")
		}
		if jobcount == 18446744073709551615 {
			jobcount = 0
		}
		jobcount += 1
		job.id = jobcount
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

func init() {
	d := NewDispatcher(1)
	d.Run()
}
