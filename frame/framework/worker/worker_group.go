package worker

type WorkerGroup struct {
	workers []IWorker
	len     int32
}

func NewWorkerGroup(count int32) *WorkerGroup {
	workers := &WorkerGroup{
		workers: make([]IWorker, count),
		len:     count,
	}
	for i := int32(0); i < count; i++ {
		workers.workers[i] = NewWorker("group", 1e5)
	}
	return workers
}

func (wg *WorkerGroup) Post(ID int32, f func()) {
	if wg.len <= 0 {
		return
	}
	x := ID % wg.len
	wg.workers[x].Post(f)
}

func (wg *WorkerGroup) Run() {
	for _, worker := range wg.workers {
		worker.Run()
	}
}

func (wg *WorkerGroup) Finish() {
	for _, worker := range wg.workers {
		worker.Fini()
	}
}
