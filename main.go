package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

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
		name VARCHAR,
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

func insertTicker(db *sql.DB, name, price, high, low string) {
	mut.Lock()
	defer mut.Unlock()
	fmt.Printf("💾 Insertion : %s | Price=%s | High=%s | Low=%s\n", name, price, high, low)
	insertSQL := `INSERT INTO Tickers (name, price, high_24, low_24) VALUES (?, ?, ?, ?)`
	_, err := db.Exec(insertSQL, name, price, high, low)
	if err != nil {
		log.Fatal(err)
	}
}

// --- API STRUCTURES ---

type KrakenTickerResponse struct {
	Error  []string                     `json:"error"`
	Result map[string]KrakenTickerData `json:"result"`
}

type KrakenTickerData struct {
	A []string `json:"a"` // Ask
	B []string `json:"b"` // Bid
	C []string `json:"c"` // Last trade closed
	V []string `json:"v"` // Volume
	P []string `json:"p"` // VWAP
	T []int    `json:"t"` // Number of trades
	L []string `json:"l"` // Low prices
	H []string `json:"h"` // High prices
	O string   `json:"o"` // Opening price
}

// --- API HANDLING ---

func getAllPair() []byte {
	resp, err := http.Get("https://api.kraken.com/0/public/Ticker")
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

// --- TCP SERVER ---

func handleConnection(conn net.Conn) {
	conn.RemoteAddr()
	defer conn.Close()
	conn.Write([]byte("Bienvenue sur le serveur Kraken Tracker !\n"))
}

func listenServer(port uint) {
	address := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Println("❌ Erreur d'écoute :", err)
		return
	}
	fmt.Printf("🚀 Serveur TCP en écoute sur le port %d\n", port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("❌ Erreur d'acceptation :", err)
			continue
		}
		go handleConnection(conn)
	}
}

// --- MAIN ---

func main() {
	db := connectDb()
	defer db.Close()

	pairs := getAllPair()
	data := parseResponse(pairs)

	var wg sync.WaitGroup
	wg.Add(len(data.Result))

	fmt.Println("✅ Données récupérées depuis l'API Kraken :")

	

	// ✅ Insérer les données en parallèle
	for pairName, info := range data.Result {
		go func(name string, data KrakenTickerData) {
			defer wg.Done()
			insertTicker(db, name, data.C[0], data.H[1], data.L[1])
			//fmt.Printf("📊 %s : Dernier prix = %s | 24h High = %s | 24h Low = %s\n", name, data.C[0], data.H[1], data.L[1])
		}(pairName, info)
	}

	wg.Add(1)
	// ✅ Démarrer le serveur TCP sans le bloquer
	go func () {
		defer wg.Done()
		listenServer(8080)
	}()

	// ✅ Attendre que les insertions soient toutes terminées
	wg.Wait()

	
}

