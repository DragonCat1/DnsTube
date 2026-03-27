package dns

import (
	"context"
	"time"

	"github.com/dnstube/dnstube/internal/store"
)

const (
	queryLogFlushInterval = 2 * time.Second
	// 内存上限：超出则丢弃新日志，避免极端 QPS 撑爆内存
	queryLogMaxBuffered = 50000
)

func (e *Engine) queryLogFlushLoop(ctx context.Context) {
	defer e.queryLogWG.Done()
	tick := time.NewTicker(queryLogFlushInterval)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			e.flushQueryLogBatch()
			return
		case <-tick.C:
			e.flushQueryLogBatch()
		}
	}
}

func (e *Engine) flushQueryLogBatch() {
	var batch []store.QueryLogInsert
	e.queryLogMu.Lock()
	if len(e.queryLogBuf) > 0 {
		batch = e.queryLogBuf
		e.queryLogBuf = nil
	}
	e.queryLogMu.Unlock()
	if len(batch) == 0 {
		return
	}
	c, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := e.store.InsertQueryLogsBatch(c, batch); err != nil {
		if e.log != nil {
			e.log.Debug("query log batch insert failed, dropped", "err", err, "rows", len(batch))
		}
	}
}

func (e *Engine) enqueueQueryLog(_ context.Context, q store.QueryLogInsert) {
	e.queryLogMu.Lock()
	defer e.queryLogMu.Unlock()
	if len(e.queryLogBuf) >= queryLogMaxBuffered {
		n := e.queryLogOverflow.Add(1)
		if n == 1 || n%10000 == 0 {
			e.log.Warn("query log buffer full, dropping new entries", "dropped", n)
		}
		return
	}
	e.queryLogBuf = append(e.queryLogBuf, q)
}
