package gmcp

import (
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/events"
)

// GMCPBatcher batches multiple GMCP updates for the same user to reduce network overhead
type GMCPBatcher struct {
	mu       sync.Mutex
	pending  map[int]*userBatch // userId -> batch
	interval time.Duration
}

type userBatch struct {
	userId  int
	modules map[string]interface{} // module name -> payload
	timer   *time.Timer
}

var (
	batcher = &GMCPBatcher{
		pending:  make(map[int]*userBatch),
		interval: 50 * time.Millisecond, // Batch for 50ms
	}
)

// BatchGMCPUpdate queues a GMCP update for batching
func BatchGMCPUpdate(userId int, module string, payload interface{}) {
	batcher.mu.Lock()
	defer batcher.mu.Unlock()

	batch, exists := batcher.pending[userId]
	if !exists {
		batch = &userBatch{
			userId:  userId,
			modules: make(map[string]interface{}),
		}
		batcher.pending[userId] = batch

		// Start timer to flush this batch
		batch.timer = time.AfterFunc(batcher.interval, func() {
			flushUserBatch(userId)
		})
	}

	// Add or update the module payload
	batch.modules[module] = payload
}

func flushUserBatch(userId int) {
	batcher.mu.Lock()
	batch, exists := batcher.pending[userId]
	if !exists {
		batcher.mu.Unlock()
		return
	}
	delete(batcher.pending, userId)
	batcher.mu.Unlock()

	// Send all batched updates as a single GMCP message
	if len(batch.modules) == 1 {
		// Single module - send normally
		for module, payload := range batch.modules {
			events.AddToQueue(GMCPOut{
				UserId:  userId,
				Module:  module,
				Payload: payload,
			})
		}
	} else if len(batch.modules) > 1 {
		// Multiple modules - could implement a combined format if clients support it
		// For now, send them sequentially but in the same event processing cycle
		for module, payload := range batch.modules {
			events.AddToQueue(GMCPOut{
				UserId:  userId,
				Module:  module,
				Payload: payload,
			})
		}
	}
}

// FlushAllBatches forces all pending batches to be sent immediately
func FlushAllBatches() {
	batcher.mu.Lock()
	userIds := make([]int, 0, len(batcher.pending))
	for userId, batch := range batcher.pending {
		userIds = append(userIds, userId)
		batch.timer.Stop()
	}
	batcher.mu.Unlock()

	for _, userId := range userIds {
		flushUserBatch(userId)
	}
}
