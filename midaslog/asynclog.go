package midaslog

import (
	"sync"
	"time"
)

type asyncLog struct {
	// TODO : replace the var

	// to convinience,not thread safety map,so don't modify the route
	route map[Level]IWriterWithTime
	msgCh chan *item
	wg    *sync.WaitGroup
	// just for future
	lock   *sync.Mutex
	stopCh chan struct{}
}

type item struct {
	level Level
	msg   []byte
	t     time.Time
}

// NewAsyncLog returns an instance
// maybe better to use config as params
func NewAsyncLog(route map[Level]IWriterWithTime, msgChCap int) *asyncLog {
	a := &asyncLog{
		route:  route,
		msgCh:  make(chan *item, msgChCap),
		wg:     new(sync.WaitGroup),
		stopCh: make(chan struct{}, 1),
		lock:   new(sync.Mutex),
	}
	a.wg.Add(1)
	go a.asyncLogRun()
	return a
}
func (this *asyncLog) asyncLogRun() {
	defer this.wg.Done()
	for {
		select {
		case <-this.stopCh:
			for _, w := range this.route {
				w.Close()
			}
			return
		case ii := <-this.msgCh:
			if w, ok := this.route[ii.level]; ok {
				w.WriteWithTime(ii.msg, ii.t)
				//w.Write(ii.msg)
			}
		}
	}
}

func (this *asyncLog) Write(level Level, p []byte, now time.Time) (int, error) {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.msgCh != nil {
		this.msgCh <- &item{level: level, msg: p, t: now}
	}
	return len(p), nil
}

// stop and Reset
func (this *asyncLog) StopAndReset() {
	this.lock.Lock()
	defer this.lock.Unlock()
	this.stopCh <- struct{}{}
	this.wg.Wait()
	close(this.msgCh)
	close(this.stopCh)
	this.msgCh = nil
	this.stopCh = nil
	this.wg = nil
	this.route = nil
}
