package channel

type Job func()

type WorkerPool struct {
	worker int
	jobs   chan Job
}

func NewWorkerPool(worker, queue int) *WorkerPool {
	jobs := make(chan Job, queue)
	return &WorkerPool{worker: worker, jobs: jobs}
}

func (pool *WorkerPool) Start() {
	for i := 0; i < pool.worker; i++ {
		go func() {
			for job := range pool.jobs {
				job()
			}
		}()
	}
}

func (pool *WorkerPool) Submit(job Job) {
	pool.jobs <- job
}
func (pool *WorkerPool) Close() {
	close(pool.jobs)
}
