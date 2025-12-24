package main

import (
    "fmt"
    "io"
    "net/http"
    "os"
    "sync"
    "time"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage:")
        fmt.Println("  go run client/load_client.go <test_type>")
        fmt.Println("")
        fmt.Println("Test types:")
        fmt.Println("  stampede - Cache Stampede test (100 concurrent requests)")
        fmt.Println("  normal   - Normal load test (10 concurrent requests)")
        os.Exit(1)
    }

    testType := os.Args[1]
    serverURL := "http://localhost:80"
    if len(os.Args) >= 3 {
        serverURL = os.Args[2]
    }

    fmt.Printf("=== %s Test ===\n", testType)
    fmt.Printf("Server: %s\n", serverURL)
    fmt.Printf("Time: %s\n\n", time.Now().Format("15:04:05"))

    switch testType {
    case "stampede":
        runStampedeTest(serverURL)
    case "normal":
        runNormalTest(serverURL)
    default:
        fmt.Printf("Unknown test type: %s\n", testType)
        os.Exit(1)
    }
}

func runStampedeTest(serverURL string) {
    fmt.Println("🔥 Cache Stampede Test")
    fmt.Println("Waiting for cache to expire (11s)...")
    time.Sleep(11 * time.Second)
    
    result := runLoadTest(serverURL, 100)
    
    fmt.Printf("\n📊 Results:\n")
    fmt.Printf("  Total requests: %d\n", result.Total)
    fmt.Printf("  Successful: %d\n", result.Success)
    fmt.Printf("  Failed: %d\n", result.Failed)
    fmt.Printf("  Duration: %.3fs\n", result.Duration.Seconds())
    fmt.Printf("  Cache status:\n")
    fmt.Printf("    - HIT: %d\n", result.CacheHit)
    fmt.Printf("    - MISS: %d\n", result.CacheMiss)
    fmt.Printf("    - SHARED: %d\n", result.CacheShared)

    if result.CacheShared > 0 {
        fmt.Printf("\n✅ Singleflight working! %d requests shared the result.\n", result.CacheShared)
    }
}

func runNormalTest(serverURL string) {
    fmt.Println("📊 Normal Load Test")
    
    result := runLoadTest(serverURL, 10)
    
    fmt.Printf("\n📊 Results:\n")
    fmt.Printf("  Total requests: %d\n", result.Total)
    fmt.Printf("  Successful: %d\n", result.Success)
    fmt.Printf("  Failed: %d\n", result.Failed)
    fmt.Printf("  Duration: %.3fs\n", result.Duration.Seconds())
    fmt.Printf("  Cache status:\n")
    fmt.Printf("    - HIT: %d\n", result.CacheHit)
    fmt.Printf("    - MISS: %d\n", result.CacheMiss)
    fmt.Printf("    - SHARED: %d\n", result.CacheShared)

    if result.CacheShared > 0 {
        fmt.Printf("\n✅ Singleflight working! %d requests shared the result.\n", result.CacheShared)
    }
}

type TestResult struct {
    Total       int
    Success     int
    Failed      int
    Duration    time.Duration
    CacheHit    int
    CacheMiss   int
    CacheShared int
}

func runLoadTest(serverURL string, concurrent int) TestResult {
    var wg sync.WaitGroup
    var mu sync.Mutex
    
    result := TestResult{Total: concurrent}
    start := time.Now()
    
    for i := 0; i < concurrent; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            
            resp, err := http.Get(serverURL + "/ranking")
            
            mu.Lock()
            defer mu.Unlock()
            
            if err != nil {
                result.Failed++
                return
            }
            
            result.Success++
            
            cacheStatus := resp.Header.Get("X-Cache")
            switch cacheStatus {
            case "HIT":
                result.CacheHit++
            case "MISS":
                result.CacheMiss++
            case "SHARED":
                result.CacheShared++
            }
            
            io.Copy(io.Discard, resp.Body)
            resp.Body.Close()
        }()
    }
    
    wg.Wait()
    result.Duration = time.Since(start)
    return result
}
