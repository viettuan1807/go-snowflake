package snowflake_test

import (
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/viettuan1807/go-snowflake"
)

// TestNewSnowflake tests the basic creation of a Snowflake instance
func TestNewSnowflake(t *testing.T) {
	sf := snowflake.NewSnowflake(1)
	if sf == nil {
		t.Fatal("NewSnowflake returned nil")
	}
}

// TestNewSnowflakeWithOptions tests the creation of Snowflake with various options
func TestNewSnowflakeWithOptions(t *testing.T) {
	customEpoch := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	
	tests := []struct {
		name    string
		opts    []snowflake.Option
		wantErr bool
	}{
		{
			name: "with custom epoch",
			opts: []snowflake.Option{snowflake.WithEpoch(customEpoch)},
		},
		{
			name: "with worker bits",
			opts: []snowflake.Option{snowflake.WithWorkerBits(10)},
		},
		{
			name: "with datacenter bits",
			opts: []snowflake.Option{
				snowflake.WithDatacenterBits(5),
				snowflake.WithWorkerBits(10),
				snowflake.WithSequenceBits(12),
				snowflake.WithTimestampBits(36),
			},
		},
		{
			name: "with datacenter ID",
			opts: []snowflake.Option{
				snowflake.WithDatacenterBits(5),
				snowflake.WithDatacenterID(1),
				snowflake.WithWorkerBits(10),
				snowflake.WithSequenceBits(12),
				snowflake.WithTimestampBits(36),
			},
		},
		{
			name: "with sequence bits",
			opts: []snowflake.Option{snowflake.WithSequenceBits(12)},
		},
		{
			name: "with timestamp bits",
			opts: []snowflake.Option{
				snowflake.WithTimestampBits(38),
				snowflake.WithSequenceBits(12),
			},
		},
		{
			name: "disable warn expiry",
			opts: []snowflake.Option{snowflake.WithWarnExpiry(false)},
		},
		{
			name: "with warn expiry before days",
			opts: []snowflake.Option{snowflake.WithWarnExpiryBeforeDays(30)},
		},
		{
			name: "with expiry callback",
			opts: []snowflake.Option{snowflake.WithExpiryCallback(func(expiry time.Time) {
				// Custom callback
			})},
		},
		{
			name: "multiple options",
			opts: []snowflake.Option{
				snowflake.WithEpoch(customEpoch),
				snowflake.WithWorkerBits(10),
				snowflake.WithDatacenterBits(5),
				snowflake.WithSequenceBits(12),
				snowflake.WithTimestampBits(36),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sf := snowflake.NewSnowflake(1, tt.opts...)
			if sf == nil {
				t.Fatal("NewSnowflake returned nil")
			}
		})
	}
}

// TestNewSnowflakePanicOnInvalidBits tests panic when total bits exceed 63
func TestNewSnowflakePanicOnInvalidBits(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic for invalid bit configuration")
		}
	}()
	
	// Total bits: 20 + 20 + 20 + 20 = 80 > 63
	snowflake.NewSnowflake(1,
		snowflake.WithWorkerBits(20),
		snowflake.WithDatacenterBits(20),
		snowflake.WithSequenceBits(20),
		snowflake.WithTimestampBits(20),
	)
}

// TestNewSnowflakePanicOnInvalidWorkerID tests panic for invalid worker ID
func TestNewSnowflakePanicOnInvalidWorkerID(t *testing.T) {
	tests := []struct {
		name     string
		workerID int64
		opts     []snowflake.Option
	}{
		{
			name:     "negative worker ID",
			workerID: -1,
		},
		{
			name:     "worker ID too large (default)",
			workerID: 2048, // max is 2047 with default 11 bits
		},
		{
			name:     "worker ID too large (custom bits)",
			workerID: 16, // max is 15 with 4 bits
			opts:     []snowflake.Option{snowflake.WithWorkerBits(4)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Expected panic for worker ID %d", tt.workerID)
				}
			}()
			snowflake.NewSnowflake(tt.workerID, tt.opts...)
		})
	}
}

// TestNewSnowflakePanicOnInvalidDatacenterID tests panic for invalid datacenter ID
func TestNewSnowflakePanicOnInvalidDatacenterID(t *testing.T) {
	tests := []struct {
		name         string
		datacenterID int64
		bits         uint8
	}{
		{
			name:         "negative datacenter ID",
			datacenterID: -1,
			bits:         5,
		},
		{
			name:         "datacenter ID too large",
			datacenterID: 32, // max is 31 with 5 bits
			bits:         5,
		},
		{
			name:         "datacenter ID > 0 with 0 bits",
			datacenterID: 1, // max is 0 with 0 bits
			bits:         0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Expected panic for datacenter ID %d with %d bits", tt.datacenterID, tt.bits)
				}
			}()
			
			opts := []snowflake.Option{
				snowflake.WithDatacenterBits(tt.bits),
				snowflake.WithDatacenterID(tt.datacenterID),
			}
			
			// Adjust other bits if needed
			if tt.bits > 0 {
				opts = append(opts,
					snowflake.WithWorkerBits(10),
					snowflake.WithSequenceBits(12),
					snowflake.WithTimestampBits(36),
				)
			}
			
			snowflake.NewSnowflake(1, opts...)
		})
	}
}

// TestNewSnowflakePanicOnInvalidWarnExpiryBeforeDays tests panic for invalid warnExpiryBeforeDays
func TestNewSnowflakePanicOnInvalidWarnExpiryBeforeDays(t *testing.T) {
	tests := []struct {
		name string
		days int64
	}{
		{
			name: "zero days",
			days: 0,
		},
		{
			name: "negative days",
			days: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Expected panic for warnExpiryBeforeDays %d", tt.days)
				}
			}()
			snowflake.NewSnowflake(1, snowflake.WithWarnExpiryBeforeDays(tt.days))
		})
	}
}

// TestNextID tests basic ID generation
func TestNextID(t *testing.T) {
	sf := snowflake.NewSnowflake(1, snowflake.WithWarnExpiry(false))
	
	id1 := sf.NextID()
	id2 := sf.NextID()
	
	if id1 == 0 {
		t.Error("Generated ID should not be zero")
	}
	
	if id2 <= id1 {
		t.Error("IDs should be increasing")
	}
}

// TestNextIDUniqueness tests that generated IDs are unique
func TestNextIDUniqueness(t *testing.T) {
	sf := snowflake.NewSnowflake(1, snowflake.WithWarnExpiry(false))
	
	seen := make(map[int64]bool)
	count := 10000
	
	for i := 0; i < count; i++ {
		id := sf.NextID()
		if seen[id] {
			t.Errorf("Duplicate ID generated: %d", id)
		}
		seen[id] = true
	}
	
	if len(seen) != count {
		t.Errorf("Expected %d unique IDs, got %d", count, len(seen))
	}
}

// TestNextIDConcurrency tests concurrent ID generation for uniqueness
func TestNextIDConcurrency(t *testing.T) {
	sf := snowflake.NewSnowflake(1, snowflake.WithWarnExpiry(false))
	
	numGoroutines := 10
	idsPerGoroutine := 1000
	
	var mu sync.Mutex
	seen := make(map[int64]bool)
	var wg sync.WaitGroup
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < idsPerGoroutine; j++ {
				id := sf.NextID()
				mu.Lock()
				if seen[id] {
					t.Errorf("Duplicate ID generated in concurrent test: %d", id)
				}
				seen[id] = true
				mu.Unlock()
			}
		}()
	}
	
	wg.Wait()
	
	expectedCount := numGoroutines * idsPerGoroutine
	if len(seen) != expectedCount {
		t.Errorf("Expected %d unique IDs, got %d", expectedCount, len(seen))
	}
}

// TestGetTimestamp tests timestamp extraction from ID
func TestGetTimestamp(t *testing.T) {
	sf := snowflake.NewSnowflake(1, snowflake.WithWarnExpiry(false))
	
	beforeGen := time.Now().UnixMilli()
	id := sf.NextID()
	afterGen := time.Now().UnixMilli()
	
	timestamp := sf.GetTimestamp(id)
	
	if timestamp < beforeGen || timestamp > afterGen {
		t.Errorf("Timestamp %d not in expected range [%d, %d]", timestamp, beforeGen, afterGen)
	}
}

// TestGetWorkerID tests worker ID extraction from ID
func TestGetWorkerID(t *testing.T) {
	workerID := int64(42)
	sf := snowflake.NewSnowflake(workerID, snowflake.WithWarnExpiry(false))
	
	id := sf.NextID()
	extractedWorkerID := sf.GetWorkerID(id)
	
	if extractedWorkerID != workerID {
		t.Errorf("Expected worker ID %d, got %d", workerID, extractedWorkerID)
	}
}

// TestGetDatacenterID tests datacenter ID extraction from ID
func TestGetDatacenterID(t *testing.T) {
	tests := []struct {
		name         string
		datacenterID int64
		bits         uint8
	}{
		{
			name:         "datacenter ID 0 with 0 bits",
			datacenterID: 0,
			bits:         0,
		},
		{
			name:         "datacenter ID 1 with 5 bits",
			datacenterID: 1,
			bits:         5,
		},
		{
			name:         "datacenter ID 15 with 5 bits",
			datacenterID: 15,
			bits:         5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := []snowflake.Option{
				snowflake.WithDatacenterBits(tt.bits),
				snowflake.WithDatacenterID(tt.datacenterID),
				snowflake.WithWarnExpiry(false),
			}
			
			// Adjust other bits to ensure total doesn't exceed 63
			if tt.bits > 0 {
				opts = append(opts,
					snowflake.WithWorkerBits(10),
					snowflake.WithSequenceBits(12),
					snowflake.WithTimestampBits(36),
				)
			}
			
			sf := snowflake.NewSnowflake(1, opts...)
			
			id := sf.NextID()
			extractedDatacenterID := sf.GetDatacenterID(id)
			
			if extractedDatacenterID != tt.datacenterID {
				t.Errorf("Expected datacenter ID %d, got %d", tt.datacenterID, extractedDatacenterID)
			}
		})
	}
}

// TestGetSequence tests sequence extraction from ID
func TestGetSequence(t *testing.T) {
	sf := snowflake.NewSnowflake(1, snowflake.WithWarnExpiry(false))
	
	// Generate multiple IDs in the same millisecond
	ids := make([]int64, 10)
	for i := 0; i < 10; i++ {
		ids[i] = sf.NextID()
	}
	
	// At least some should have different sequences
	sequences := make(map[int64]bool)
	for _, id := range ids {
		seq := sf.GetSequence(id)
		sequences[seq] = true
	}
	
	// We should have at least one sequence number
	if len(sequences) == 0 {
		t.Error("Expected at least one sequence number")
	}
}

// TestExpiryWarning tests the expiry warning functionality
func TestExpiryWarning(t *testing.T) {
	var callbackCalled bool
	var callbackExpiry time.Time
	
	// Use a very short timestamp space that will expire soon
	_ = snowflake.NewSnowflake(1,
		snowflake.WithTimestampBits(10), // Very small timestamp space
		snowflake.WithWarnExpiry(true),
		snowflake.WithWarnExpiryBeforeDays(9999999), // Very large value to ensure warning
		snowflake.WithExpiryCallback(func(expiry time.Time) {
			callbackCalled = true
			callbackExpiry = expiry
		}),
	)
	
	if !callbackCalled {
		t.Error("Expected expiry callback to be called")
	}
	
	if callbackExpiry.IsZero() {
		t.Error("Expected expiry time to be set")
	}
}

// TestExpiryWarningDisabled tests that expiry warning can be disabled
func TestExpiryWarningDisabled(t *testing.T) {
	var callbackCalled bool
	
	_ = snowflake.NewSnowflake(1,
		snowflake.WithWarnExpiry(false),
		snowflake.WithExpiryCallback(func(expiry time.Time) {
			callbackCalled = true
		}),
	)
	
	// Callback should not be called when warnings are disabled
	if callbackCalled {
		t.Error("Expected expiry callback not to be called when warnings are disabled")
	}
}

// TestSequenceOverflow tests behavior when sequence overflows
func TestSequenceOverflow(t *testing.T) {
	// Use very small sequence bits to trigger overflow quickly
	sf := snowflake.NewSnowflake(1,
		snowflake.WithSequenceBits(2), // Only 4 sequences per millisecond
		snowflake.WithWarnExpiry(false),
	)
	
	// Generate enough IDs to potentially overflow the sequence
	for i := 0; i < 100; i++ {
		id := sf.NextID()
		if id == 0 {
			t.Error("Generated ID should not be zero")
		}
	}
}

// TestDifferentWorkerIDs tests that different workers generate different IDs
func TestDifferentWorkerIDs(t *testing.T) {
	sf1 := snowflake.NewSnowflake(1, snowflake.WithWarnExpiry(false))
	sf2 := snowflake.NewSnowflake(2, snowflake.WithWarnExpiry(false))
	
	id1 := sf1.NextID()
	id2 := sf2.NextID()
	
	worker1 := sf1.GetWorkerID(id1)
	worker2 := sf2.GetWorkerID(id2)
	
	if worker1 == worker2 {
		t.Error("Different Snowflake instances should have different worker IDs")
	}
	
	if worker1 != 1 {
		t.Errorf("Expected worker ID 1, got %d", worker1)
	}
	
	if worker2 != 2 {
		t.Errorf("Expected worker ID 2, got %d", worker2)
	}
}

// TestIDComponentsRoundTrip tests that we can extract all components correctly
func TestIDComponentsRoundTrip(t *testing.T) {
	workerID := int64(100)
	datacenterID := int64(5)
	
	sf := snowflake.NewSnowflake(workerID,
		snowflake.WithDatacenterBits(5),
		snowflake.WithDatacenterID(datacenterID),
		snowflake.WithWorkerBits(10),
		snowflake.WithSequenceBits(12),
		snowflake.WithTimestampBits(36),
		snowflake.WithWarnExpiry(false),
	)
	
	id := sf.NextID()
	
	// Extract components
	extractedTimestamp := sf.GetTimestamp(id)
	extractedWorkerID := sf.GetWorkerID(id)
	extractedDatacenterID := sf.GetDatacenterID(id)
	extractedSequence := sf.GetSequence(id)
	
	// Verify worker ID
	if extractedWorkerID != workerID {
		t.Errorf("Worker ID mismatch: expected %d, got %d", workerID, extractedWorkerID)
	}
	
	// Verify datacenter ID
	if extractedDatacenterID != datacenterID {
		t.Errorf("Datacenter ID mismatch: expected %d, got %d", datacenterID, extractedDatacenterID)
	}
	
	// Verify timestamp is reasonable (within last few seconds)
	now := time.Now().UnixMilli()
	if extractedTimestamp < now-5000 || extractedTimestamp > now {
		t.Errorf("Timestamp out of expected range: %d (now: %d)", extractedTimestamp, now)
	}
	
	// Verify sequence is valid
	if extractedSequence < 0 {
		t.Errorf("Sequence should be non-negative, got %d", extractedSequence)
	}
}

// TestCustomEpoch tests using a custom epoch
func TestCustomEpoch(t *testing.T) {
	customEpoch := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	
	sf := snowflake.NewSnowflake(1,
		snowflake.WithEpoch(customEpoch),
		snowflake.WithWarnExpiry(false),
	)
	
	id := sf.NextID()
	timestamp := sf.GetTimestamp(id)
	
	// Timestamp should be after custom epoch
	if timestamp < customEpoch {
		t.Errorf("Timestamp %d should be after custom epoch %d", timestamp, customEpoch)
	}
	
	// Timestamp should be close to current time
	now := time.Now().UnixMilli()
	if timestamp < now-5000 || timestamp > now {
		t.Errorf("Timestamp %d out of expected range (now: %d)", timestamp, now)
	}
}

// TestMaxWorkerID tests the maximum worker ID
func TestMaxWorkerID(t *testing.T) {
	// With default 11 bits, max worker ID is 2047
	sf := snowflake.NewSnowflake(2047, snowflake.WithWarnExpiry(false))
	
	id := sf.NextID()
	workerID := sf.GetWorkerID(id)
	
	if workerID != 2047 {
		t.Errorf("Expected worker ID 2047, got %d", workerID)
	}
}

// TestMinWorkerID tests the minimum worker ID
func TestMinWorkerID(t *testing.T) {
	sf := snowflake.NewSnowflake(0, snowflake.WithWarnExpiry(false))
	
	id := sf.NextID()
	workerID := sf.GetWorkerID(id)
	
	if workerID != 0 {
		t.Errorf("Expected worker ID 0, got %d", workerID)
	}
}

// TestClockRollbackSmall tests handling of small clock rollbacks by manipulating state
func TestClockRollbackSmall(t *testing.T) {
	sf := snowflake.NewSnowflake(1, snowflake.WithWarnExpiry(false))
	
	// Generate an ID to initialize the state
	_ = sf.NextID()
	
	// Use unsafe reflection to set lastTimestamp to a future value
	// This will cause the next call to NextID to detect a clock rollback
	v := reflect.ValueOf(sf).Elem()
	lastTimestampField := v.FieldByName("lastTimestamp")
	
	// Set the atomic.Int64 to a future timestamp using unsafe
	futureTime := time.Now().UnixMilli() + 50 // 50ms in the future
	ptr := unsafe.Pointer(lastTimestampField.UnsafeAddr())
	atomicPtr := (*atomic.Int64)(ptr)
	atomicPtr.Store(futureTime)
	
	// Now call NextID - it should detect the clock rollback and handle it
	// Since the difference is < 100ms, it should sleep and recover
	id := sf.NextID()
	if id == 0 {
		t.Error("Generated ID should not be zero after clock rollback")
	}
}

// TestClockRollbackLarge tests handling of large clock rollbacks (should panic)
func TestClockRollbackLarge(t *testing.T) {
	sf := snowflake.NewSnowflake(1, snowflake.WithWarnExpiry(false))
	
	// Generate an ID to initialize the state
	_ = sf.NextID()
	
	// Use unsafe reflection to set lastTimestamp to a far future value
	v := reflect.ValueOf(sf).Elem()
	lastTimestampField := v.FieldByName("lastTimestamp")
	
	// Set the atomic.Int64 to a far future timestamp (200ms in the future)
	farFutureTime := time.Now().UnixMilli() + 200 // 200ms in the future
	ptr := unsafe.Pointer(lastTimestampField.UnsafeAddr())
	atomicPtr := (*atomic.Int64)(ptr)
	atomicPtr.Store(farFutureTime)
	
	// Now call NextID - it should panic due to significant clock rollback
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for large clock rollback")
		}
	}()
	
	sf.NextID()
}

func BenchmarkSnowflake(b *testing.B) {
	sf := snowflake.NewSnowflake(12)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sf.NextID()
	}
}

func BenchmarkSnowflakeParallel(b *testing.B) {
	sf := snowflake.NewSnowflake(12)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = sf.NextID()
		}
	})
}

func BenchmarkGetTimestamp(b *testing.B) {
	sf := snowflake.NewSnowflake(12)
	id := sf.NextID()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sf.GetTimestamp(id)
	}
}

func BenchmarkGetWorkerID(b *testing.B) {
	sf := snowflake.NewSnowflake(12)
	id := sf.NextID()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sf.GetWorkerID(id)
	}
}

func BenchmarkGetDatacenterID(b *testing.B) {
	sf := snowflake.NewSnowflake(12)
	id := sf.NextID()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sf.GetDatacenterID(id)
	}
}

func BenchmarkGetSequence(b *testing.B) {
	sf := snowflake.NewSnowflake(12)
	id := sf.NextID()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sf.GetSequence(id)
	}
}

func BenchmarkConcurrentGeneration(b *testing.B) {
	sf := snowflake.NewSnowflake(12)
	numWorkers := 10

	b.ResetTimer()
	var wg sync.WaitGroup
	idsPerWorker := b.N / numWorkers

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < idsPerWorker; j++ {
				_ = sf.NextID()
			}
		}()
	}
	wg.Wait()
}
