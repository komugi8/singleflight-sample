package main

import (
    "encoding/json"
    "fmt"
    "log"
    "math/rand"
    "net/http"
    "os"
    "strconv"
    "sync"
    "time"

    "golang.org/x/sync/singleflight"
)

// ランキングデータ
type Ranking struct {
    Items     []RankingItem `json:"items"`
    UpdatedAt time.Time     `json:"updated_at"`
}

type RankingItem struct {
    Rank  int    `json:"rank"`
    Name  string `json:"name"`
    Score int    `json:"score"`
}

// グローバル変数
var (
    group singleflight.Group
    cache = make(map[string]cacheItem)
    mu    sync.RWMutex
)

type cacheItem struct {
    data      string
    expiresAt time.Time
}

// モックキャッシュ
func getCache(key string) (string, bool) {
    mu.RLock()
    defer mu.RUnlock()
    
    item, exists := cache[key]
    if !exists || time.Now().After(item.expiresAt) {
        return "", false
    }
    return item.data, true
}

func setCache(key, value string, ttl time.Duration) {
    mu.Lock()
    defer mu.Unlock()
    cache[key] = cacheItem{
        data:      value,
        expiresAt: time.Now().Add(ttl),
    }
}

// 重いDB処理をモック
func getHeavyRanking() (*Ranking, error) {
    delay := 3 * time.Second
    if env := os.Getenv("DB_DELAY"); env != "" {
        if d, err := strconv.Atoi(env); err == nil {
            delay = time.Duration(d) * time.Millisecond
        }
    }
    
    time.Sleep(delay) // 重い処理をシミュレート
    
    // ダミーデータ生成
    items := make([]RankingItem, 10)
    for i := 0; i < 10; i++ {
        items[i] = RankingItem{
            Rank:  i + 1,
            Name:  fmt.Sprintf("User%d", rand.Intn(1000)),
            Score: rand.Intn(10000),
        }
    }
    
    return &Ranking{
        Items:     items,
        UpdatedAt: time.Now(),
    }, nil
}

func rankingHandler(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    log.Printf("Request started")
    
    // 1. キャッシュ確認
    if data, found := getCache("ranking"); found {
        log.Printf("Cache HIT (%.3fs)", time.Since(start).Seconds())
        w.Header().Set("X-Cache", "HIT")
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(data))
        return
    }
    
    log.Printf("Cache MISS - using singleflight")
    
    // 2. Singleflight で処理集約
    result, err, shared := group.Do("ranking", func() (interface{}, error) {
        log.Printf("DB query started (LEADER)")
        
        ranking, err := getHeavyRanking()
        if err != nil {
            return nil, err
        }
        
        data, _ := json.Marshal(ranking)
        setCache("ranking", string(data), 10*time.Second) // TTL 10秒
        
        log.Printf("DB query completed")
        return data, nil
    })
    
    if err != nil {
        log.Printf("Error: %v", err)
        http.Error(w, "Internal Server Error", 500)
        return
    }
    
    status := "MISS"
    if shared {
        status = "SHARED"
    }
    
    log.Printf("Response sent (%s, %.3fs)", status, time.Since(start).Seconds())
    
    w.Header().Set("X-Cache", status)
    w.Header().Set("Content-Type", "application/json")
    w.Write(result.([]byte))
}

func main() {
    port := "80"
    if p := os.Getenv("PORT"); p != "" {
        port = p
    }
    
    http.HandleFunc("/ranking", rankingHandler)
    
    log.Printf("Server starting on port %s", port)
    log.Printf("Try: go run client/load_client.go stampede")
    
    if err := http.ListenAndServe(":"+port, nil); err != nil {
        log.Fatal("Server failed:", err)
    }
}
