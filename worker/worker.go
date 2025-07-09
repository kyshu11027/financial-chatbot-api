package worker

import (
	"context"
	"encoding/json"
	"finance-chatbot/api/logger"
	"finance-chatbot/api/models"
	"finance-chatbot/api/sse"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

type WorkerPool struct {
	workers    int
	partitions []chan []byte
	wg         sync.WaitGroup
	ctx        context.Context
	cancelFunc context.CancelFunc

	// Metrics
	mu                 sync.RWMutex
	messagesProcessed  uint64
	processingDuration uint64
	bufferFillLevels   []uint64
	messagesDropped    uint64
}

func NewWorkerPool(workers int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	partitions := make([]chan []byte, workers)
	bufferLevels := make([]uint64, workers)
	for i := range partitions {
		partitions[i] = make(chan []byte, 100) // Buffer size of 100 per partition
	}
	return &WorkerPool{
		workers:          workers,
		partitions:       partitions,
		ctx:              ctx,
		cancelFunc:       cancel,
		bufferFillLevels: bufferLevels,
	}
}

func (wp *WorkerPool) Start() {
	logger.Get().Info("Starting worker pool", zap.Int("workers", wp.workers))
	for i := range wp.partitions {
		wp.wg.Add(1)
		go wp.worker(i)
	}
}

func (wp *WorkerPool) Stop() {
	logger.Get().Info("Stopping worker pool")
	wp.cancelFunc()
	for _, ch := range wp.partitions {
		close(ch)
	}
	wp.wg.Wait()
}

func (wp *WorkerPool) Submit(job []byte, partition int32) {
	if int(partition) >= len(wp.partitions) {
		wp.mu.Lock()
		wp.messagesDropped++
		wp.mu.Unlock()
		logger.Get().Error("Invalid partition number",
			zap.Int32("partition", partition),
			zap.Int("max_partitions", len(wp.partitions)))
		return
	}

	wp.mu.Lock()
	wp.bufferFillLevels[partition]++
	wp.mu.Unlock()

	select {
	case wp.partitions[partition] <- job:
		logger.Get().Debug("Job submitted to worker pool",
			zap.Int32("partition", partition))
	case <-wp.ctx.Done():
		wp.mu.Lock()
		wp.messagesDropped++
		wp.mu.Unlock()
		logger.Get().Warn("Worker pool is stopped, job not submitted")
	}
}

func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()
	logger.Get().Info("Worker started", zap.Int("worker_id", id))

	for {
		select {
		case job, ok := <-wp.partitions[id]:
			if !ok {
				logger.Get().Info("Worker stopping", zap.Int("worker_id", id))
				return
			}

			wp.mu.Lock()
			wp.bufferFillLevels[id]--
			wp.mu.Unlock()

			startTime := time.Now()

			var aiResponse models.AIResponse
			if err := json.Unmarshal(job, &aiResponse); err != nil {
				wp.mu.Lock()
				wp.messagesDropped++
				wp.mu.Unlock()
				logger.Get().Error("Failed to unmarshal message",
					zap.Int("worker_id", id),
					zap.Error(err))
				continue
			}

			logger.Get().Debug("Processing message",
				zap.Int("worker_id", id),
				zap.String("conversation_id", aiResponse.ConversationID))

			// Process the message
			sse.SendChunkToClient(aiResponse.ConversationID, string(job))

			wp.mu.Lock()
			wp.messagesProcessed++
			wp.processingDuration += uint64(time.Since(startTime).Milliseconds())
			wp.mu.Unlock()

		case <-wp.ctx.Done():
			logger.Get().Info("Worker stopping due to context cancellation",
				zap.Int("worker_id", id))
			return
		}
	}
}

// MetricsHandler returns the current metrics as JSON
func (wp *WorkerPool) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	var avgProcessingTime float64
	if wp.messagesProcessed > 0 {
		avgProcessingTime = float64(wp.processingDuration) / float64(wp.messagesProcessed)
	}

	metrics := map[string]any{
		"messages_processed": wp.messagesProcessed,
		"messages_dropped":   wp.messagesDropped,
		"avg_processing_ms":  avgProcessingTime,
		"buffer_levels":      wp.bufferFillLevels,
		"active_workers":     wp.workers,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}
