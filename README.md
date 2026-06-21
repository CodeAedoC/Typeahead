# 🔍 Search Typeahead System

A high-performance, low-latency search typeahead and suggestion system. This project implements advanced backend data-system designs including asynchronous batch writing, in-memory distributed caching via consistent hashing, and recency-aware trending search algorithms.

**[👉 Demo Video: https://github.com/user-attachments/assets/1f0dc4ca-5aa2-44b0-89bf-c96caae1c628]**

## ✨ Core Features
* **Sub-millisecond Search Suggestions:** Uses an indexed SQLite database for lightning-fast prefix matching.
* **Debounced React UI:** Prevents backend API spamming while delivering a smooth user experience.
* **Trending Searches:** Calculates real-time trending queries using a weighted recency algorithm `(all_time * 0.1) + (recent * 10000.0)`.
* **Asynchronous Batch Writes:** Uses Go channels to queue massive search spikes and flush them to the database in a single transaction, reducing write pressure by up to 99%.
* **Distributed Caching Layer:** Implements Consistent Hashing to reliably route and cache prefix requests across multiple in-memory nodes.

## 🛠️ Tech Stack
* **Frontend:** React, Vite
* **Backend:** Go (Golang) Standard Library
* **Database:** SQLite (Pure Go Driver, zero external dependencies)
* **Dataset:** Peter Norvig’s Google Web Trillion Word Corpu

s (333,333 queries)

## 🚀 Quick Start (Local Setup)

This project requires **zero manual database installation**. Everything runs natively.

### 1. Start the Backend
```bash
# Clone the repository and navigate to the backend
cd backend
go mod tidy

# Run the server. On the very first run, it will automatically download 
# the 333k dataset and build the SQLite database in about 3 seconds.
go run main.go
