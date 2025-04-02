package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// --- STRUCTURES ---

type Ticker struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Price  string `json:"price"`
	High24 string `json:"high_24"`
	Low24  string `json:"low_24"`
}

type KrakenTickerResponse struct {
	Error  []string                     `json:"error"`
	Result map[string]KrakenTickerData `json:"result"`
}

type KrakenTickerData struct {
	C []string `json:"c"`
	V []string `json:"v"`
	L []string `json:"l"`
	H []string `json:"h"`
	O string   `json:"o"`
}

type KrakenStatusResponse struct {
	Error  []string         `json:"error"`
	Result KrakenStatusData `json:"result"`
}

type KrakenStatusData struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

// --- DB SETUP ---

var mut sync.Mutex

func connectDb() *sql.DB {
	fmt.Println("🔌 Connexion à la base de données SQLite...")
	db, err := sql.Open("sqlite3", "./kraken.db")
	if err != nil {
		log.Fatal(err)
	}

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS Tickers (
		id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		name VARCHAR UNIQUE,
		price VARCHAR,
		high_24 VARCHAR,
		low_24 VARCHAR
	);`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("❌ Erreur création de la table: %s", err)
	}

	fmt.Println("✅ Table prête.")
	return db
}

func insertOrUpdateTicker(db *sql.DB, name, price, high, low string) {
	mut.Lock()
	defer mut.Unlock()
	fmt.Printf("🔁 Mise à jour : %s | Price=%s | High=%s | Low=%s\n", name, price, high, low)
	query := `
		INSERT INTO Tickers (name, price, high_24, low_24) 
		VALUES (?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET 
			price = excluded.price,
			high_24 = excluded.high_24,
			low_24 = excluded.low_24
	`
	_, err := db.Exec(query, name, price, high, low)
	if err != nil {
		log.Fatal(err)
	}
}

// --- API HANDLING ---

func getPair(pair string) []byte {
	resp, err := http.Get(fmt.Sprintf("https://api.kraken.com/0/public/Ticker?pair=%s", pair))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	return body
}

func parseResponse(response []byte) KrakenTickerResponse {
	var value KrakenTickerResponse
	err := json.Unmarshal(response, &value)
	if err != nil {
		log.Fatal(err)
	}
	return value
}

// --- CSV ARCHIVING ---

func writeCSV(pair, price, high, low string) {
	now := time.Now()

	if _, err := os.Stat("archives"); os.IsNotExist(err) {
		_ = os.Mkdir("archives", 0755)
	}

	filename := fmt.Sprintf("archives/%s_%02d_%02d_%d_%02d_%02d.csv",
		pair, now.Day(), now.Month(), now.Year(), now.Hour(), now.Minute())

	file, err := os.Create(filename)
	if err != nil {
		log.Printf("❌ Erreur création CSV : %v", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"Pair", "Price", "High 24h", "Low 24h"})
	writer.Write([]string{pair, price, high, low})

	fmt.Printf("📝 Fichier CSV archivé : %s\n", filename)
}

// --- TICKER JOB ---

func startTickerJob(db *sql.DB, pairs []string) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	runOnce(db, pairs)

	for range ticker.C {
		runOnce(db, pairs)
	}
}

func runOnce(db *sql.DB, pairs []string) {
	var wg sync.WaitGroup
	c := make(chan KrakenTickerResponse, len(pairs))

	wg.Add(len(pairs))
	for _, pair := range pairs {
		go func(p string) {
			defer wg.Done()
			body := getPair(p)
			data := parseResponse(body)
			c <- data
		}(pair)
	}

	wg.Wait()
	close(c)

	for resp := range c {
		if len(resp.Error) > 0 {
			log.Printf("❌ Erreur API Kraken: %v\n", resp.Error)
			continue
		}
		for name, data := range resp.Result {
			if len(data.C) == 0 || len(data.H) < 2 || len(data.L) < 2 {
				log.Printf("⚠️ Données insuffisantes pour %s", name)
				continue
			}
			insertOrUpdateTicker(db, name, data.C[0], data.H[1], data.L[1])
			writeCSV(name, data.C[0], data.H[1], data.L[1])
		}
	}
}

// --- HTTP HANDLERS ---

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	pair := strings.TrimPrefix(r.URL.Path, "/download/")
	if pair == "" {
		http.Error(w, "❌ Nom de pair manquant", http.StatusBadRequest)
		return
	}

	pattern := fmt.Sprintf("archives/%s_*.csv", pair)
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		http.Error(w, fmt.Sprintf("❌ Aucun fichier trouvé pour la pair %s", pair), http.StatusNotFound)
		return
	}

	latest := matches[0]
	for _, file := range matches {
		if file > latest {
			latest = file
		}
	}

	http.ServeFile(w, r, latest)
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	resp, err := http.Get("https://api.kraken.com/0/public/SystemStatus")
	if err != nil {
		http.Error(w, "❌ Erreur requête vers Kraken", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "❌ Erreur lecture réponse Kraken", http.StatusInternalServerError)
		return
	}

	var status KrakenStatusResponse
	if err := json.Unmarshal(body, &status); err != nil {
		http.Error(w, "❌ Erreur parsing JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func pairsHandler(w http.ResponseWriter, r *http.Request) {
	files, err := filepath.Glob("archives/*.csv")
	if err != nil {
		http.Error(w, "❌ Erreur lecture fichiers", http.StatusInternalServerError)
		return
	}

	unique := make(map[string]bool)
	for _, file := range files {
		base := filepath.Base(file)
		parts := strings.SplitN(base, "_", 2)
		if len(parts) >= 1 {
			unique[parts[0]] = true
		}
	}

	var result []string
	for pair := range unique {
		result = append(result, pair)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func pairHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/pair/")
		if name == "" {
			http.Error(w, "❌ Nom de la pair manquant", http.StatusBadRequest)
			return
		}

		row := db.QueryRow("SELECT id, name, price, high_24, low_24 FROM Tickers WHERE name = ?", name)

		var t Ticker
		err := row.Scan(&t.ID, &t.Name, &t.Price, &t.High24, &t.Low24)
		if err == sql.ErrNoRows {
			http.Error(w, fmt.Sprintf("❌ Aucune donnée trouvée pour %s", name), http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, "❌ Erreur lecture base", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(t)
	}
}

// --- SERVER ---

func listenServer(port uint, db *sql.DB) {
	http.HandleFunc("/download/", downloadHandler)
	http.HandleFunc("/status", statusHandler)
	http.HandleFunc("/pairs", pairsHandler)
	http.HandleFunc("/pair/", pairHandler(db))

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("🚀 Serveur en écoute sur http://localhost%s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("❌ Erreur serveur HTTP : %v", err)
	}
}

// --- MAIN ---

func main() {
	db := connectDb()
	defer db.Close()

	pairs := []string{"BTCEUR", "ETHEUR", "DOGEEUR", "PEPEEUR", "XLMEUR", "XRPEUR"}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		startTickerJob(db, pairs)
	}()

	go func() {
		defer wg.Done()
		listenServer(4242, db)
	}()

	wg.Wait()
}
