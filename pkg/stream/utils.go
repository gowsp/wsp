package stream

import (
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

var globalTaskID uint64

func nextTaskID() uint64 {
	return atomic.AddUint64(&globalTaskID, 1)
}

func Copy(local, remote io.ReadWriteCloser) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(local, remote)
		local.Close()
	}()
	go func() {
		defer wg.Done()
		io.Copy(remote, local)
		remote.Close()
	}()
	wg.Wait()
}

func NewTaskPool(parallel, capacity int) *TaskPool {
	tasks := make(chan func(), capacity)
	for i := 0; i < parallel; i++ {
		go func() {
			for task := range tasks {
				task()
			}
		}()
	}
	return &TaskPool{tasks: tasks}
}

type TaskPool struct {
	wg    sync.WaitGroup
	tasks chan func()
}

func (p *TaskPool) Add(task func()) {
	p.wg.Add(1)
	p.tasks <- func() {
		defer p.wg.Done()
		task()
	}
}
func (p *TaskPool) Wait() {
	p.wg.Wait()
}
func (p *TaskPool) Close() {
	p.Wait()
	close(p.tasks)
}

func NewPollTaskPool(parallel, capacity int) *PollTaskPool {
	tasks := make([]chan func(), parallel)
	for i := 0; i < parallel; i++ {
		item := make(chan func(), capacity)
		go func() {
			for task := range item {
				task()
			}
		}()
		tasks[i] = item
	}
	return &PollTaskPool{parallel: uint64(parallel), tasks: tasks}
}

type PollTaskPool struct {
	parallel uint64

	wg    sync.WaitGroup
	tasks []chan func()
}

func (p *PollTaskPool) Add(id uint64, task func()) {
	p.wg.Add(1)
	p.tasks[id%p.parallel] <- func() {
		defer p.wg.Done()
		task()
	}
}

func (p *PollTaskPool) Wait() {
	p.wg.Wait()
}
func (p *PollTaskPool) Close() {
	p.Wait()
	for _, v := range p.tasks {
		close(v)
	}
}
func NewSignal() *Signal {
	return &Signal{done: make(chan error, 1)}
}

type Signal struct {
	done chan error
}

func (s *Signal) Notify(err error) {
	s.done <- err
}
func (s *Signal) Wait(timeout time.Duration) error {
	select {
	case res := <-s.done:
		return res
	case <-time.After(timeout):
		return errors.New("timeout")
	}
}
