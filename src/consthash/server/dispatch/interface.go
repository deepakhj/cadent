package dispatch

/************************************************************************/
/**********  Standard Worker Dispatcher Queue ***************************/
/************************************************************************/
// insert job queue workers

type IJob interface {
	DoWork()
}

type IWorker interface {
	Workpool() chan chan IJob
	Jobs() chan IJob
	Shutdown() chan bool

	Start() error
	Stop() error
}

type IDispatcher interface {
	Workpool() chan chan IJob
	JobsQueue() chan IJob

	Run() error
}
