package worker

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charan/url-shortener/internal/domain"
	"github.com/charan/url-shortener/internal/repository/postgres"
	"github.com/charan/url-shortener/pkg/geoip"
	"github.com/redis/go-redis/v9"
)

const redisOverflowStreamKey = "click_events_overflow"

type ClickWorker struct {
	eventChan      chan domain.ClickEvent
	clickEventRepo *postgres.ClickEventRepo
	geoResolver    *geoip.Resolver
	redisClient    *redis.Client
	done           chan struct{}
	wg             sync.WaitGroup
	droppedEvents  atomic.Uint64
}

func NewClickWorker(clickEventRepo *postgres.ClickEventRepo, geoResolver *geoip.Resolver, redisClient *redis.Client) *ClickWorker {
	return &ClickWorker{
		eventChan:      make(chan domain.ClickEvent, 10000),
		clickEventRepo: clickEventRepo,
		geoResolver:    geoResolver,
		redisClient:    redisClient,
		done:           make(chan struct{}),
	}
}

func (w *ClickWorker) Enqueue(event domain.ClickEvent) {
	// Non-blocking enqueue keeps redirect latency low; events are dropped when the buffer is full.
	select {
	case w.eventChan <- event:
	default:
		if w.pushOverflowEvent(event) {
			return
		}

		dropped := w.droppedEvents.Add(1)
		if dropped == 1 || dropped%100 == 0 {
			log.Printf("Warning: click event channel and overflow stream are unavailable, dropping event (dropped_total=%d)", dropped)
		}
	}
}

func (w *ClickWorker) Start() {
	log.Println("click worker starting")
	for i := 0; i < 2; i++ {
		w.wg.Add(1)
		go w.process(i)
	}
}

func (w *ClickWorker) Stop() {
	log.Println("click worker stopping")
	close(w.done)
	w.wg.Wait()
	log.Println("click worker stopped")
}

func (w *ClickWorker) process(workerID int) {
	defer w.wg.Done()

	batch := make([]domain.ClickEvent, 0, 100)
	overflowIDs := make([]string, 0, 100)
	ticker := time.NewTicker(500 * time.Millisecond)
	reportTicker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	defer reportTicker.Stop()

	consecutiveFlushFailures := 0
	nextFlushAttempt := time.Time{}

	for {
		select {
		case event := <-w.eventChan:
			geo := w.geoResolver.Resolve(event.IPAddress)
			event.Country = geo.Country
			event.City = geo.City

			batch = append(batch, event)
			if len(batch) >= 100 {
				if nextFlushAttempt.IsZero() || !time.Now().Before(nextFlushAttempt) {
					if w.flush(batch) {
						w.ackOverflowEntries(overflowIDs)
						batch = batch[:0]
						overflowIDs = overflowIDs[:0]
						consecutiveFlushFailures = 0
						nextFlushAttempt = time.Time{}
					} else {
						consecutiveFlushFailures++
						nextFlushAttempt = time.Now().Add(calculateFlushBackoff(consecutiveFlushFailures))
					}
				}
			}

		case <-ticker.C:
			if !nextFlushAttempt.IsZero() && time.Now().Before(nextFlushAttempt) {
				continue
			}

			if workerID == 0 {
				w.drainOverflowIntoBatch(&batch, &overflowIDs)
			}
			if len(batch) > 0 {
				if w.flush(batch) {
					w.ackOverflowEntries(overflowIDs)
					batch = batch[:0]
					overflowIDs = overflowIDs[:0]
					consecutiveFlushFailures = 0
					nextFlushAttempt = time.Time{}
				} else {
					consecutiveFlushFailures++
					nextFlushAttempt = time.Now().Add(calculateFlushBackoff(consecutiveFlushFailures))
				}
			}

		case <-reportTicker.C:
			dropped := w.droppedEvents.Load()
			if dropped > 0 {
				log.Printf("click worker queue health dropped_total=%d queue_depth=%d", dropped, len(w.eventChan))
			}

		case <-w.done:
			for {
				select {
				case event := <-w.eventChan:
					geo := w.geoResolver.Resolve(event.IPAddress)
					event.Country = geo.Country
					event.City = geo.City
					batch = append(batch, event)
				default:
					if len(batch) > 0 {
						if w.flush(batch) {
							w.ackOverflowEntries(overflowIDs)
						}
					}
					return
				}
			}
		}
	}
}

func (w *ClickWorker) pushOverflowEvent(event domain.ClickEvent) bool {
	if w.redisClient == nil {
		return false
	}

	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf("Warning: failed to marshal overflow click event: %v", err)
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	if err := w.redisClient.XAdd(ctx, &redis.XAddArgs{
		Stream: redisOverflowStreamKey,
		MaxLen: 100000,
		Approx: true,
		Values: map[string]interface{}{"payload": payload},
	}).Err(); err != nil {
		log.Printf("Warning: failed to persist overflow click event: %v", err)
		return false
	}

	return true
}

func (w *ClickWorker) drainOverflowIntoBatch(batch *[]domain.ClickEvent, overflowIDs *[]string) {
	if w.redisClient == nil || len(*batch) >= 100 || len(*overflowIDs) > 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	remaining := int64(100 - len(*batch))
	entries, err := w.redisClient.XRangeN(ctx, redisOverflowStreamKey, "-", "+", remaining).Result()
	if err != nil {
		if err != redis.Nil {
			log.Printf("Warning: failed to read overflow stream: %v", err)
		}
		return
	}

	if len(entries) == 0 {
		return
	}

	invalidIDs := make([]string, 0, len(entries))
	for _, entry := range entries {
		raw, ok := entry.Values["payload"]
		if !ok {
			invalidIDs = append(invalidIDs, entry.ID)
			continue
		}

		payload, ok := raw.(string)
		if !ok {
			invalidIDs = append(invalidIDs, entry.ID)
			continue
		}

		var event domain.ClickEvent
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			log.Printf("Warning: failed to decode overflow click event: %v", err)
			invalidIDs = append(invalidIDs, entry.ID)
			continue
		}

		*batch = append(*batch, event)
		*overflowIDs = append(*overflowIDs, entry.ID)
	}

	if len(invalidIDs) == 0 {
		return
	}

	if err := w.redisClient.XDel(ctx, redisOverflowStreamKey, invalidIDs...).Err(); err != nil {
		log.Printf("Warning: failed to delete invalid overflow entries: %v", err)
	}
}

func (w *ClickWorker) ackOverflowEntries(ids []string) {
	if w.redisClient == nil || len(ids) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := w.redisClient.XDel(ctx, redisOverflowStreamKey, ids...).Err(); err != nil {
		log.Printf("Warning: failed to acknowledge overflow entries: %v", err)
	}
}

func calculateFlushBackoff(failures int) time.Duration {
	if failures <= 1 {
		return 200 * time.Millisecond
	}
	if failures == 2 {
		return 500 * time.Millisecond
	}
	if failures == 3 {
		return 1 * time.Second
	}
	if failures == 4 {
		return 2 * time.Second
	}
	return 5 * time.Second
}

func (w *ClickWorker) flush(batch []domain.ClickEvent) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := w.clickEventRepo.BatchInsert(ctx, batch); err != nil {
		log.Printf("Error flushing click events: %v", err)
		return false
	}

	log.Printf("click events flushed count=%d", len(batch))
	return true
}
