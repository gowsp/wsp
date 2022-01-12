package pkg

import (
	"bytes"
	"sync"
)

var bufPool = &sync.Pool{
	New: func() interface{} {
		size := 32 * 1024
		buf := make([]byte, size)
		return &buf
	},
}
var BufPool = &sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

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
