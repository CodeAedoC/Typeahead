package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	_ "modernc.org/sqlite" 
)

var db *sql.DB
var searchQueue = make(chan string, 10000)

// --- CACHE & CONSISTENT HASHING ---
type Suggestion struct {
	Query string `json:"query"`
}

type CacheNode struct {
	Data map[string][]Suggestion
	mu   sync.RWMutex
}

var cacheRing = []string{"NodeA", "NodeB", "NodeC"}
var cacheNodes = map[string]*CacheNode{
	"NodeA": {Data: make(map[string][]Suggestion)},
	"NodeB": {Data: make(map[string][]Suggestion)},
	"NodeC": {Data: make(map[string][]Suggestion)},
}

func getTargetNode(key string) string {
	h := fnv.New32a()
	h.Write([]byte(key))
	return cacheRing[h.Sum32()%3]
}

func invalidateCache(query string) {
	for i := 1; i <= len(query); i++ {
		sub := query[:i]
		node := cacheNodes[getTargetNode(sub)]
		node.mu.Lock()
		delete(node.Data, sub)
		node.mu.Unlock()
	}
}

// --- DATABASE SETUP & INGESTION ---
func initDB() {
	var err error
	db, err = sql.Open("sqlite", "./typeahead.db")
	if err != nil {
		log.Fatal(err)
	}

	createTable := `
	CREATE TABLE IF NOT EXISTS search_queries (
		query TEXT PRIMARY KEY,
		all_time INTEGER DEFAULT 0,
		recent INTEGER DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_query ON search_queries(query);
	`
	if _, err := db.Exec(createTable); err != nil {
		log.Fatal(err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM search_queries").Scan(&count)
	if count > 0 {
		log.Printf("Database ready! Found %d existing queries.\n", count)
		return
	}

	log.Println("Database is empty. Downloading Google dataset (333,333 queries)...")
	resp, err := http.Get("http://norvig.com/ngrams/count_1w.txt")
	if err != nil {
		log.Fatal("Failed to download dataset:", err)
	}
	defer resp.Body.Close()

	tx, _ := db.Begin()
	stmt, _ := tx.Prepare("INSERT INTO search_queries (query, all_time, recent) VALUES (?, ?, 0)")
	defer stmt.Close()

	scanner := bufio.NewScanner(resp.Body)
	loaded := 0
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), "\t")
		if len(parts) == 2 {
			word := parts[0]
			freq, _ := strconv.Atoi(parts[1])
			
			scaledFreq := freq / 100000
			if scaledFreq == 0 {
				scaledFreq = 1
			}
			stmt.Exec(word, scaledFreq)
			loaded++
		}
	}
	tx.Commit()
	log.Printf("Successfully saved %d queries to SQLite!\n", loaded)
}

func main() {
	initDB()

	// --- BATCH WRITER WORKER ---
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		batch := make(map[string]int)

		upsertQuery := `
			INSERT INTO search_queries (query, all_time, recent) 
			VALUES (?, ?, ?)
			ON CONFLICT(query) DO UPDATE SET 
				all_time = all_time + excluded.all_time,
				recent = recent + excluded.recent;
		`

		for {
			select {
			case q := <-searchQueue:
				batch[q]++
			case <-ticker.C:
				if len(batch) > 0 {
					tx, _ := db.Begin()
					stmt, _ := tx.Prepare(upsertQuery)
					
					for query, count := range batch {
						stmt.Exec(query, 0, count)
						invalidateCache(query)
					}
					
					stmt.Close()
					tx.Commit()
					batch = make(map[string]int)
				}
			}
		}
	}()

	// --- API ROUTES ---

	// 1. SUGGEST API
	http.HandleFunc("/suggest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		
		prefix := r.URL.Query().Get("q")
		if prefix == "" {
			json.NewEncoder(w).Encode([]Suggestion{})
			return
		}

		node := cacheNodes[getTargetNode(prefix)]

		node.mu.RLock()
		if val, ok := node.Data[prefix]; ok {
			node.mu.RUnlock()
			json.NewEncoder(w).Encode(val)
			return
		}
		node.mu.RUnlock()

		searchPattern := prefix + "%"
		rows, err := db.Query(`
			SELECT query FROM search_queries 
			WHERE query LIKE ? 
			ORDER BY (all_time * 0.1) + (recent * 10000.0) DESC 
			LIMIT 10
		`, searchPattern)
		
		var results []Suggestion
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var s Suggestion
				rows.Scan(&s.Query)
				results = append(results, s)
			}
		}
		if results == nil {
			results = []Suggestion{}
		}

		node.mu.Lock()
		node.Data[prefix] = results
		node.mu.Unlock()

		json.NewEncoder(w).Encode(results)
	})

	// 2. SEARCH API
	http.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		var req struct{ Query string `json:"query"` }
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil && req.Query != "" {
			select {
			case searchQueue <- req.Query:
			default:
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "Searched"})
	})

	// 3. TRENDING API
	http.HandleFunc("/trending", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		
		type TrendItem struct {
			Query string `json:"query"`
			Count int    `json:"count"`
		}
		
		rows, err := db.Query(`
			SELECT query, all_time 
			FROM search_queries 
			ORDER BY (all_time * 0.1) + (recent * 10000.0) DESC 
			LIMIT 5
		`)
		
		var trends []TrendItem
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var item TrendItem
				rows.Scan(&item.Query, &item.Count)
				if item.Query != "" {
					trends = append(trends, item)
				}
			}
		}

		if len(trends) == 0 {
			trends = []TrendItem{}
		}
		json.NewEncoder(w).Encode(trends)
	})

	// 4. DEBUG CACHE ROUTING API
	http.HandleFunc("/cache/debug", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		prefix := r.URL.Query().Get("prefix")
		if prefix == "" {
			http.Error(w, `{"error": "Prefix required"}`, http.StatusBadRequest)
			return
		}

		nodeId := getTargetNode(prefix)
		node := cacheNodes[nodeId]

		node.mu.RLock()
		_, isHit := node.Data[prefix]
		node.mu.RUnlock()

		json.NewEncoder(w).Encode(map[string]interface{}{
			"prefix":        prefix,
			"hashed_value":  fnv.New32a().Sum32(),
			"assigned_node": nodeId,
			"cache_hit":     isHit,
		})
	})

	fmt.Println("SQLite Typeahead Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}