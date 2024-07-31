package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type JSONResponse struct {
	Data Data `json:"USDBRL"`
}

type Data struct {
	Value string `json:"bid"`
}

func main() {
	http.HandleFunc("/cotacao", handler)
	http.ListenAndServe(":8080", nil)
}

func connectDatabase() *sql.DB {
	file := "client-server.db"
	db, err := sql.Open("sqlite3", file)
	if err != nil {
		panic(err)
	}

	create := `
	  CREATE TABLE IF NOT EXISTS quotations (
	  id INTEGER NOT NULL PRIMARY KEY,
	  time DATETIME NOT NULL,
	  value DECIMAL(10,2)
	  );`

	_, err = db.Exec(create)
	if err != nil {
		panic(err)
	}

	return db
}

func handler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	select {
	case <-ctx.Done():
		panic(err)
	default:
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}

		var quotation JSONResponse
		err = json.Unmarshal(body, &quotation)
		if err != nil {
			panic(err)
		}

		saveToDatabase(ctx, &quotation, w)
	}
}

func saveToDatabase(ctx context.Context, quotation *JSONResponse, w http.ResponseWriter) {
	db := connectDatabase()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
	defer cancel()
	_, err := db.ExecContext(ctx, "INSERT INTO quotations VALUES (NULL, ?, ?);", time.Now(), quotation.Data.Value)

	select {
	case <-ctx.Done():
		panic(err)
	default:
		json.NewEncoder(w).Encode(quotation.Data.Value)
	}

}
