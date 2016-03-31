/*
 a simple implementation of a dispatcher

 all one needs to do is implement the IJob interface and

	type Job struct{
		to_work_on string
		thing_that_works *FancyObject
	}
	func (j *Job) DoWork(){
		j.thing_that_works(to_work_on)
	}

	workers := 10
	queue_depth := 100
 	job_queue      := make(chan *Job, queue_depth)
 	disp_queue     := make(chan chan *Job, workers)
	job_dispatcher := NewDispatch(workers, disp_queue, job_queue)
	job_dispatcher.Run()

	job_queue <- &Job{to_work_on: "frank", thing_that_works: MyDoDad}

*/

package dispatch

type Worker struct {
	worker_pool chan chan IJob
	job_channel chan IJob
	quit        chan bool
	dispatcher  IDispatcher
}

func NewWorker(workerPool chan chan IJob, dispatcher IDispatcher) *Worker {
	return &Worker{
		worker_pool: workerPool,
		job_channel: make(chan IJob),
		quit:        make(chan bool),
		dispatcher:  dispatcher,
	}
}

func (w *Worker) Workpool() chan chan IJob {
	return w.worker_pool
}
func (w *Worker) Jobs() chan IJob {
	return w.job_channel
}

func (w *Worker) Shutdown() chan bool {
	return w.quit
}

// Start method starts the run loop for the worker, listening for a quit channel in
// case we need to stop it
func (w *Worker) Start() {
	go func() {
		for {
			// register the current worker into the worker queue.
			w.Workpool() <- w.Jobs()

			select {
			case job := <-w.Jobs():
				err := job.DoWork()
				// put back on queue
				if err != nil && w.dispatcher.Retries() < job.OnRetry() {
					job.IncRetry()
					w.dispatcher.JobsQueue() <- job
				}
			case <-w.quit:
				return
			}
		}
		return
	}()
	return
}

// Stop signals the worker to stop listening for work requests.
func (w *Worker) Stop() {
	go func() {
		w.Shutdown() <- true
		return
	}()
}

/************* DISPATCH ********/

type Dispatch struct {
	work_pool  chan chan IJob
	job_queue  chan IJob
	shutdown   chan bool
	workers    []*Worker
	numworkers int
	retries    int
}

func NewDispatch(numworkers int, work_pool chan chan IJob, job_queue chan IJob) *Dispatch {
	dis := &Dispatch{
		work_pool:  work_pool,
		numworkers: numworkers,
		job_queue:  job_queue,
		workers:    make([]*Worker, numworkers, numworkers),
		shutdown:   make(chan bool),
		retries:    0,
	}
	return dis
}

func (d *Dispatch) Retries() int {
	return d.retries
}

func (d *Dispatch) SetRetries(r int) {
	d.retries = r
}

func (d *Dispatch) Workpool() chan chan IJob {
	return d.work_pool
}

func (d *Dispatch) JobsQueue() chan IJob {
	return d.job_queue
}

func (d *Dispatch) Run() error {
	// starting n number of workers
	for i := 0; i < d.numworkers; i++ {
		worker := NewWorker(d.work_pool, d)
		d.workers[i] = worker
		worker.Start()
	}

	go d.dispatch()
	return nil
}

func (d *Dispatch) BackgroundRun() error {
	// starting n number of workers
	for i := 0; i < d.numworkers; i++ {
		worker := NewWorker(d.work_pool, d)
		d.workers[i] = worker
		worker.Start()
	}

	go d.background_dispatch()
	return nil
}

func (d *Dispatch) Shutdown() {
	d.shutdown <- true
}

func (d *Dispatch) dispatch() {
	for {
		select {
		case job := <-d.JobsQueue():
			// a job request has been received
			//func(job IJob) {
			// try to obtain a worker job channel that is available.
			// this will block until a worker is idle
			jobChannel := <-d.Workpool()

			// dispatch the job to the worker job channel
			jobChannel <- job
			//	return
			//}
		case <-d.shutdown:
			for _, w := range d.workers {
				w.Shutdown() <- true
			}
			//close(d.JobsQueue())
			//close(d.Workpool())
			return
		}
	}
	return
}

func (d *Dispatch) background_dispatch() {
	for {
		select {
		case job := <-d.JobsQueue():
			// a job request has been received
			go func(job IJob) {
				// try to obtain a worker job channel that is available.
				// this will block until a worker is idle
				jobChannel := <-d.Workpool()

				// dispatch the job to the worker job channel
				jobChannel <- job
				return
			}(job)
		case <-d.shutdown:
			for _, w := range d.workers {
				w.Shutdown() <- true
			}
			//close(d.JobsQueue())
			//close(d.Workpool())
			return
		}
	}
	return
}
