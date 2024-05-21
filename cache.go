package main

import (
	"database/sql"
	"errors"
	"net/http"
	"sync"
	"time"
)

var mutex sync.Mutex

func initializeCache(path string) (*sql.DB, error) {
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

func (app *App) checkCache(r *http.Request, port string) ([]byte, error) {
	mutex.Lock()
	defer mutex.Unlock()

	cacheKey := getCacheKey(r, port)
	var response []byte
	var cachedAt time.Time

	// Order by cachedAt DESC to get the most recent record.
	row := app.Cache.QueryRow("SELECT cachedAt, response FROM cache WHERE key = ? ORDER BY cachedAt DESC LIMIT 1", cacheKey)

	err := row.Scan(&cachedAt, &response)
	if err == sql.ErrNoRows {
		return nil, errors.New("not found in cache")
	}
	// TODO: Add an option to disable caching or set an indefinite caching (no expiration).
	if time.Since(cachedAt) > time.Duration(app.Config.CacheDuration)*time.Hour {
		return nil, errors.New("cached record is too old")
	}

	return response, err
}

func getCacheKey(r *http.Request, port string) string {
	return port + "_" + r.URL.String()
}

func (app *App) storeResponse(key string, resp []byte) error {
	mutex.Lock()
	defer mutex.Unlock()

	currentTime := time.Now()
	_, err := app.Cache.Exec("INSERT OR REPLACE INTO cache (cachedAt, key, response) VALUES (?, ?, ?)", currentTime, key, resp)
	if err != nil {
		return err
	}

	return nil
}
