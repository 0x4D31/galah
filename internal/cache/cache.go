package cache

import (
	"database/sql"
	"errors"
	"net/http"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var mutex sync.Mutex

func InitializeCache(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
    CREATE TABLE IF NOT EXISTS cache (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		cachedAt DATETIME,
		key TEXT,
		response TEXT
	)	
`)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func CheckCache(client *sql.DB, r *http.Request, port string, cacheDuration int) ([]byte, error) {
	mutex.Lock()
	defer mutex.Unlock()

	cacheKey := GetCacheKey(r, port)
	var response []byte
	var cachedAt time.Time

	// Order by cachedAt DESC to get the most recent record.
	row := client.QueryRow("SELECT cachedAt, response FROM cache WHERE key = ? ORDER BY cachedAt DESC LIMIT 1", cacheKey)

	err := row.Scan(&cachedAt, &response)
	if err == sql.ErrNoRows {
		return nil, errors.New("not found in cache")
	}
	// TODO: Add an option to disable caching or set an indefinite caching (no expiration).
	if time.Since(cachedAt) > time.Duration(cacheDuration)*time.Hour {
		return nil, errors.New("cached record is too old")
	}

	return response, err
}

func GetCacheKey(r *http.Request, port string) string {
	return port + "_" + r.URL.String()
}

func StoreResponse(client *sql.DB, key string, resp []byte) error {
	mutex.Lock()
	defer mutex.Unlock()

	currentTime := time.Now()
	_, err := client.Exec("INSERT OR REPLACE INTO cache (cachedAt, key, response) VALUES (?, ?, ?)", currentTime, key, resp)
	if err != nil {
		return err
	}

	return nil
}
