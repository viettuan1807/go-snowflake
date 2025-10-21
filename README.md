# go-snowflake

A high-performance Snowflake ID generator for Go, providing distributed unique ID generation with timestamp ordering.

## Overview

`go-snowflake` is a Go implementation of Twitter's Snowflake algorithm for generating distributed unique IDs. Snowflake IDs are 64-bit integers composed of:

- **Timestamp** (39 bits): Milliseconds since a custom epoch (default: 2025-01-01)
- **Worker ID** (11 bits): Supports up to 2048 unique workers
- **Datacenter ID** (0 bits by default): Optional datacenter identification
- **Sequence** (13 bits): Up to 8192 IDs per millisecond per worker

This design provides:
- **Monotonically increasing IDs** within a single worker
- **Roughly time-ordered IDs** across workers
- **High throughput**: Up to 8192 IDs per millisecond per worker
- **~17 years** of timestamp space (with default configuration)
- **Collision-free** ID generation in distributed systems

## Features

- 🚀 **High Performance**: Optimized for concurrent ID generation
- 🔒 **Thread-Safe**: Uses atomic operations for thread safety
- ⚙️ **Configurable**: Customizable epoch, worker bits, datacenter bits, and sequence bits
- ⏰ **Clock Skew Handling**: Handles small clock rollbacks gracefully
- ⚠️ **Expiry Warnings**: Automatic warnings when timestamp space is nearing expiration
- 📊 **ID Decomposition**: Extract timestamp, worker ID, datacenter ID, and sequence from generated IDs

## Installation

To install the package, use `go get`:

```bash
go get github.com/viettuan1807/go-snowflake
```

Make sure your project is using Go modules:

```bash
go mod init your-project-name
go mod tidy
```

## Quick Start

Here's a basic example to get you started:

```go
package main

import (
    "fmt"
    "github.com/viettuan1807/go-snowflake"
)

func main() {
    // Create a new Snowflake generator with worker ID 1
    sf := snowflake.NewSnowflake(1)
    
    // Generate unique IDs
    id1 := sf.NextID()
    id2 := sf.NextID()
    id3 := sf.NextID()
    
    fmt.Printf("Generated ID 1: %d\n", id1)
    fmt.Printf("Generated ID 2: %d\n", id2)
    fmt.Printf("Generated ID 3: %d\n", id3)
}
```

## Usage Examples

### Basic ID Generation

```go
package main

import (
    "fmt"
    "github.com/viettuan1807/go-snowflake"
)

func main() {
    // Initialize with worker ID 42
    sf := snowflake.NewSnowflake(42)
    
    // Generate a single ID
    id := sf.NextID()
    fmt.Printf("Generated ID: %d\n", id)
}
```

### Concurrent ID Generation

The package is designed for high-concurrency scenarios:

```go
package main

import (
    "fmt"
    "sync"
    "github.com/viettuan1807/go-snowflake"
)

func main() {
    sf := snowflake.NewSnowflake(1)
    
    var wg sync.WaitGroup
    ids := make(chan int64, 100)
    
    // Generate IDs concurrently
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for j := 0; j < 10; j++ {
                ids <- sf.NextID()
            }
        }()
    }
    
    go func() {
        wg.Wait()
        close(ids)
    }()
    
    // Collect and print IDs
    for id := range ids {
        fmt.Printf("ID: %d\n", id)
    }
}
```

### Extracting Information from IDs

You can decompose a Snowflake ID to extract its components:

```go
package main

import (
    "fmt"
    "time"
    "github.com/viettuan1807/go-snowflake"
)

func main() {
    sf := snowflake.NewSnowflake(42)
    
    // Generate an ID
    id := sf.NextID()
    
    // Extract information
    timestamp := sf.GetTimestamp(id)
    workerID := sf.GetWorkerID(id)
    datacenterID := sf.GetDatacenterID(id)
    sequence := sf.GetSequence(id)
    
    fmt.Printf("ID: %d\n", id)
    fmt.Printf("Timestamp: %s\n", time.UnixMilli(timestamp))
    fmt.Printf("Worker ID: %d\n", workerID)
    fmt.Printf("Datacenter ID: %d\n", datacenterID)
    fmt.Printf("Sequence: %d\n", sequence)
}
```

## Configuration

### Worker ID

Each Snowflake instance must be initialized with a unique worker ID:

```go
// Worker IDs range from 0 to 2047 (with default configuration)
sf := snowflake.NewSnowflake(100)
```

**Important**: Ensure each worker in your distributed system has a unique worker ID to prevent ID collisions.

### Custom Configuration (Advanced)

The package supports configuration options through the `Option` pattern (note: option functions need to be defined based on your specific needs):

- **Custom Epoch**: Change the reference timestamp
- **Bit Allocation**: Customize worker bits, datacenter bits, sequence bits, and timestamp bits
- **Datacenter ID**: Set a datacenter identifier for multi-datacenter deployments
- **Expiry Warnings**: Configure or disable timestamp expiry warnings

**Note**: The total of all bits (worker + datacenter + sequence + timestamp) must not exceed 63 bits.

## Environment Setup

### Requirements

- **Go 1.24.2 or later**: This package uses Go's atomic operations and modern concurrency features

### Worker ID Management

In a distributed system, you need to ensure each worker has a unique ID. Common approaches:

1. **Static Configuration**: Assign worker IDs through environment variables or config files
2. **Service Discovery**: Use a service like etcd, Consul, or ZooKeeper to coordinate worker IDs
3. **Database Assignment**: Store and retrieve worker IDs from a central database

Example using environment variables:

```go
package main

import (
    "os"
    "strconv"
    "github.com/viettuan1807/go-snowflake"
)

func main() {
    workerIDStr := os.Getenv("WORKER_ID")
    if workerIDStr == "" {
        panic("WORKER_ID environment variable not set")
    }
    
    workerID, err := strconv.ParseInt(workerIDStr, 10, 64)
    if err != nil {
        panic("Invalid WORKER_ID: " + err.Error())
    }
    
    sf := snowflake.NewSnowflake(workerID)
    // Use sf to generate IDs
}
```

## Important Considerations

### Clock Synchronization

- **Keep system clocks synchronized** across all workers using NTP or similar
- The package handles small clock rollbacks (< 100ms) automatically
- Significant clock rollbacks will cause a panic to prevent ID collisions

### Worker ID Uniqueness

- **Each worker must have a unique ID** in your distributed system
- Worker ID collisions will result in duplicate IDs
- Plan your worker ID allocation strategy before deployment

### Timestamp Expiry

- With default settings, the timestamp space expires after ~17 years (from 2025-01-01)
- The package will warn you 365 days before expiry
- Consider your epoch and bit allocation for long-term projects

## Performance

The package is optimized for high-throughput scenarios:

- Thread-safe using atomic operations
- Optimized wait strategies for sequence exhaustion
- Minimal lock contention
- Capable of generating millions of IDs per second

Run the included benchmark:

```bash
go test -bench=. -benchmem
```

## Benchmarks

Below are the benchmark results showing the performance characteristics of the Snowflake ID generator:

```
goos: linux
goarch: amd64
pkg: github.com/viettuan1807/go-snowflake
cpu: AMD EPYC 7763 64-Core Processor                
BenchmarkSnowflake-4              	 8949064	       133.6 ns/op	       0 B/op	       0 allocs/op
BenchmarkSnowflakeParallel-4      	 8546304	       143.1 ns/op	       0 B/op	       0 allocs/op
BenchmarkGetTimestamp-4           	1000000000	         0.3285 ns/op	       0 B/op	       0 allocs/op
BenchmarkGetWorkerID-4            	1000000000	         0.3115 ns/op	       0 B/op	       0 allocs/op
BenchmarkGetDatacenterID-4        	1000000000	         0.3115 ns/op	       0 B/op	       0 allocs/op
BenchmarkGetSequence-4            	1000000000	         0.3112 ns/op	       0 B/op	       0 allocs/op
BenchmarkConcurrentGeneration-4   	24127252	        46.35 ns/op	       0 B/op	       0 allocs/op
PASS
ok  	github.com/viettuan1807/go-snowflake	5.267s
```

**Key Performance Metrics:**

### ID Generation
- **BenchmarkSnowflake**: 133.6 ns/op (~7.5 million IDs/sec)
  - Single-threaded ID generation benchmark
  - Zero memory allocations per operation
  
- **BenchmarkSnowflakeParallel**: 143.1 ns/op (~7.0 million IDs/sec)
  - Parallel ID generation with multiple goroutines
  - Demonstrates thread-safe concurrent performance
  
- **BenchmarkConcurrentGeneration**: 46.35 ns/op (~21.6 million IDs/sec aggregate)
  - Simulates 10 concurrent workers generating IDs
  - Shows excellent scalability under concurrent load

### ID Extraction (Decomposition)
All extraction methods are extremely fast with sub-nanosecond execution times:
- **GetTimestamp**: 0.33 ns/op - Extract timestamp from ID
- **GetWorkerID**: 0.31 ns/op - Extract worker ID from ID
- **GetDatacenterID**: 0.31 ns/op - Extract datacenter ID from ID
- **GetSequence**: 0.31 ns/op - Extract sequence number from ID

**Summary:**
The benchmarks demonstrate excellent performance with zero heap allocations across all operations, making this implementation suitable for high-throughput, low-latency applications. The ID extraction methods are particularly fast due to simple bitwise operations.

## API Reference

### Types

#### `Snowflake`

The main generator type.

### Functions

#### `NewSnowflake(workerID int64, opts ...Option) *Snowflake`

Creates a new Snowflake ID generator.

- `workerID`: Unique identifier for this worker (0-2047 with default config)
- `opts`: Optional configuration options
- Returns: A new `*Snowflake` instance

#### `(*Snowflake) NextID() int64`

Generates the next unique ID.

- Returns: A unique 64-bit integer ID

#### `(*Snowflake) GetTimestamp(id int64) int64`

Extracts the timestamp (in milliseconds since Unix epoch) from an ID.

#### `(*Snowflake) GetWorkerID(id int64) int64`

Extracts the worker ID from an ID.

#### `(*Snowflake) GetDatacenterID(id int64) int64`

Extracts the datacenter ID from an ID (returns 0 if datacenter bits are not configured).

#### `(*Snowflake) GetSequence(id int64) int64`

Extracts the sequence number from an ID.

## Additional Resources

- [Twitter Snowflake](https://github.com/twitter-archive/snowflake) - Original Snowflake implementation
- [Snowflake ID Algorithm Explanation](https://en.wikipedia.org/wiki/Snowflake_ID) - Wikipedia article on Snowflake IDs
- [Go Documentation](https://pkg.go.dev/github.com/viettuan1807/go-snowflake) - Package documentation on pkg.go.dev

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is open source. Please check the repository for license information.

## Support

For issues, questions, or contributions, please visit the [GitHub repository](https://github.com/viettuan1807/go-snowflake).
