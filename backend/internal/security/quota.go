package security

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type QuotaManager struct {
	limiters map[string]*quotaEntry
	mu       sync.Mutex
	limit    rate.Limit
	burst    int
}

type quotaEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func NewQuotaManager(limit rate.Limit, burst int) *QuotaManager {
	return &QuotaManager{
		limiters: make(map[string]*quotaEntry),
		limit:    limit,
		burst:    burst,
	}
}

func (m *QuotaManager) Allow(subject string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.limiters[subject]
	if !ok {
		entry = &quotaEntry{
			limiter:  rate.NewLimiter(m.limit, m.burst),
			lastSeen: time.Now(),
		}
		m.limiters[subject] = entry
	}
	entry.lastSeen = time.Now()
	return entry.limiter.Allow()
}
