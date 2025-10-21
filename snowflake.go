package snowflake

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Option is a function that configures a Snowflake instance
type Option func(*Snowflake)

func WithEpoch(epoch int64) Option {
	return func(s *Snowflake) { s.epoch = epoch }
}

func WithWorkerBits(bits uint8) Option {
	return func(s *Snowflake) { s.workerBits = bits }
}

func WithDatacenterBits(bits uint8) Option {
	return func(s *Snowflake) { s.datacenterBits = bits }
}

func WithDatacenterID(id int64) Option {
	return func(s *Snowflake) { s.datacenterID = id }
}

func WithSequenceBits(bits uint8) Option {
	return func(s *Snowflake) { s.sequenceBits = bits }
}

func WithTimestampBits(bits uint8) Option {
	return func(s *Snowflake) { s.timestampBits = bits }
}

func WithWarnExpiry(enabled bool) Option {
	return func(s *Snowflake) { s.warnExpiry = enabled }
}

func WithWarnExpiryBeforeDays(days int64) Option {
	return func(s *Snowflake) { s.warnExpiryBeforeDays = days }
}

func WithExpiryCallback(cb func(expiry time.Time)) Option {
	return func(s *Snowflake) { s.onWarn = cb }
}

type Snowflake struct {
	lastTimestamp atomic.Int64
	sequence      atomic.Int64
	workerID      int64
	datacenterID  int64
	mutex         sync.Mutex

	epoch           int64
	workerBits      uint8
	datacenterBits  uint8
	sequenceBits    uint8
	timestampBits   uint8
	maxWorkerID     int64
	maxDatacenterID int64
	maxSequence     int64
	workerShift     uint8
	datacenterShift uint8
	timestampShift  uint8

	warnExpiry           bool
	warnExpiryBeforeDays int64
	onWarn               func(expiry time.Time)
}

func NewSnowflake(workerID int64, opts ...Option) *Snowflake {
	// Default epoch: 1 Jan 2025 UTC (milliseconds since Unix epoch)
	epoch := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

	s := &Snowflake{
		epoch:                epoch,
		workerBits:           11, // 2048 workers
		datacenterBits:       0,  // default ignore datacenter
		datacenterID:         0,
		sequenceBits:         13, // 8192 IDs/ms
		timestampBits:        39, // ~17 years
		warnExpiry:           true,
		warnExpiryBeforeDays: 365,
		onWarn: func(expiry time.Time) {
			fmt.Printf("[WARN] Snowflake timestamp space will expire soon (~%s)\n", expiry.UTC().String())
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	// --- Design by Contract check ---
	totalBits := s.workerBits + s.datacenterBits + s.sequenceBits + s.timestampBits
	if totalBits > 63 {
		panic(
			fmt.Sprintf(
				"[FATAL] Invalid Snowflake configuration: total bits exceed 63.\n"+
					"workerBits (%d) + datacenterBits (%d) + sequenceBits (%d) + timestampBits (%d) = %d",
				s.workerBits, s.datacenterBits, s.sequenceBits, s.timestampBits, totalBits),
		)
	}

	// Derived constants
	s.maxWorkerID = (1 << s.workerBits) - 1
	s.maxDatacenterID = (1 << s.datacenterBits) - 1
	s.maxSequence = (1 << s.sequenceBits) - 1
	s.workerShift = s.sequenceBits
	s.datacenterShift = s.workerShift + s.workerBits
	s.timestampShift = s.datacenterShift + s.datacenterBits

	if workerID < 0 || workerID > s.maxWorkerID {
		panic("worker ID out of range")
	}
	if s.datacenterID < 0 || s.datacenterID > s.maxDatacenterID {
		panic("datacenter ID out of range")
	}

	s.workerID = workerID
	s.lastTimestamp.Store(-1)

	if s.warnExpiry {
		if s.warnExpiryBeforeDays <= 0 {
			panic("warnExpiryBeforeDays must be positive")
		}
		s.printExpiryWarning()
	}

	return s
}

func (s *Snowflake) printExpiryWarning() {
	maxAbs := s.epoch + s.maxTimestamp()
	expiry := time.UnixMilli(maxAbs).UTC()
	now := time.Now().UTC()
	daysLeft := int64(expiry.Sub(now).Hours() / 24)
	if daysLeft <= s.warnExpiryBeforeDays && s.onWarn != nil {
		s.onWarn(expiry)
		fmt.Printf("[WARN] Snowflake timestamp space will expire in %d days (~%s)\n",
			daysLeft, expiry.Format("2006-01-02"))
	}
}

func (s *Snowflake) maxTimestamp() int64 {
	return (1 << s.timestampBits) - 1
}

func (s *Snowflake) NextID() int64 {
	for {
		now := time.Now().UnixMilli()
		last := s.lastTimestamp.Load()

		if now < last {
			now = s.handleClockRollback(last, now)
		}

		if now == last {
			seq := s.sequence.Add(1)
			if seq <= s.maxSequence {
				return s.generateID(now, seq)
			}
			now = s.waitNextMillisOptimized(last)
			s.mutex.Lock()
			if s.lastTimestamp.Load() <= now {
				s.sequence.Store(0)
				s.lastTimestamp.Store(now)
			}
			s.mutex.Unlock()
			continue
		}

		if s.lastTimestamp.CompareAndSwap(last, now) {
			s.sequence.Store(0)
		}
	}
}

func (s *Snowflake) handleClockRollback(last, now int64) int64 {
	if last-now < 100 {
		time.Sleep(time.Duration(last-now) * time.Millisecond)
		return time.Now().UnixMilli()
	}
	panic("clock moved backwards significantly, potential ID collision risk")
}

func (s *Snowflake) generateID(timestamp, sequence int64) int64 {
	return ((timestamp - s.epoch) << s.timestampShift) |
		(s.datacenterID << s.datacenterShift) |
		(s.workerID << s.workerShift) |
		sequence
}

func (s *Snowflake) waitNextMillisOptimized(lastTimestamp int64) int64 {
	now := time.Now().UnixMilli()
	for i := 0; i < 3 && now <= lastTimestamp; i++ {
		runtime.Gosched()
		now = time.Now().UnixMilli()
	}
	backoff := 10 * time.Microsecond
	for now <= lastTimestamp {
		time.Sleep(backoff)
		if backoff < time.Millisecond {
			backoff *= 2
		}
		now = time.Now().UnixMilli()
	}
	return now
}

func (s *Snowflake) GetTimestamp(id int64) int64 {
	return (id >> s.timestampShift) + s.epoch
}

func (s *Snowflake) GetDatacenterID(id int64) int64 {
	if s.datacenterBits == 0 {
		return 0
	}
	return (id >> s.datacenterShift) & s.maxDatacenterID
}

func (s *Snowflake) GetWorkerID(id int64) int64 {
	return (id >> s.workerShift) & s.maxWorkerID
}

func (s *Snowflake) GetSequence(id int64) int64 {
	return id & s.maxSequence
}
