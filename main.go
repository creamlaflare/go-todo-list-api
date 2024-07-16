package main

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	_ "github.com/mattn/go-sqlite3" // import the sqlite3 driver
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func NextDate(now time.Time, date string, repeat string) (string, error) {
	parsedDate, err := time.Parse("20060102", date)
	if err != nil {
		return "", errors.New("недопустимый формат date")
	}

	repeatParts := strings.SplitN(repeat, " ", 2)
	repeatType := ""
	repeatRule := ""

	if len(repeatParts) > 0 {
		repeatType = repeatParts[0]
	}
	if len(repeatParts) > 1 {
		repeatRule = repeatParts[1]
	}

	if repeatType == "d" {
		if repeatRule == "" {
			return "", errors.New("не указан интервал в днях")
		} else {
			numberOfDays, err := strconv.Atoi(repeatRule)
			if err != nil {
				return "", errors.New("некорректно указано правило repeat")
			}
			if numberOfDays > 400 {
				return "", errors.New("превышен максимально допустимый интервал")
			}
			parsedDate = parsedDate.AddDate(0, 0, numberOfDays)
			for now.After(parsedDate) {
				parsedDate = parsedDate.AddDate(0, 0, numberOfDays)
			}
		}
	} else if repeatType == "y" {
		parsedDate = parsedDate.AddDate(1, 0, 0)
		for now.After(parsedDate) {
			parsedDate = parsedDate.AddDate(1, 0, 0)
		}
	} else if repeatType == "w" {
		substrings := strings.Split(repeatRule, ",")

		resultMap := make(map[string]bool)

		for _, value := range substrings {
			resultMap[value] = true
		}

		for {
			weekdayStr := fmt.Sprintf("%d", parsedDate.Weekday())
			if _, exists := resultMap[weekdayStr]; exists {
				break
			}
			parsedDate = parsedDate.AddDate(0, 0, 1)
		}
	} else {
		return "", errors.New("недопустимый символ")
	}

	return parsedDate.Format("20060102"), nil
}

func handleNextDate(res http.ResponseWriter, req *http.Request) {
	nowParam := req.URL.Query().Get("now")
	date := req.URL.Query().Get("date")
	repeat := req.URL.Query().Get("repeat")

	if date == "20250701" {
		fmt.Println(date)
	}

	now, err := time.Parse("20060102", nowParam)

	if err != nil {
		http.Error(res, "Неправильный формат парамeтра now", http.StatusBadRequest)
		return
	}

	newDate, err := NextDate(now, date, repeat)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	// TODO: узнать что имеется ввиду под удалением из дб, у меня же нет никакого айди таска

	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusOK)
	_, err = res.Write([]byte(newDate))
	if err != nil {
		fmt.Printf("Error in writing a response for /api/nextdate GET request,\n %v", err)
		return
	}
}

func main() {
	r := chi.NewRouter()

	TODO_DBFILE := os.Getenv("TODO_DBFILE")
	if TODO_DBFILE == "" {
		TODO_DBFILE = "scheduler.db"
	}

	db, err := sql.Open("sqlite3", TODO_DBFILE)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Always attempt to create the table and index if they do not exist
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS scheduler (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		date TEXT NOT NULL,
		title TEXT NOT NULL,
		comment TEXT NOT NULL,
		repeat TEXT NOT NULL
	)`)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_scheduler_date ON scheduler (date)`)
	if err != nil {
		log.Fatalf("Failed to create index: %v", err)
	}

	PORT := os.Getenv("TODO_PORT")
	if PORT == "" {
		PORT = "7540"
	}

	fileServer := http.FileServer(http.Dir("./web"))
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		if filepath.Ext(r.URL.Path) == ".css" {
			w.Header().Set("Content-Type", "text/css")
		}
		fileServer.ServeHTTP(w, r)
	})

	r.Get("/api/nextdate", handleNextDate)

	if err := http.ListenAndServe(fmt.Sprintf(":%s", PORT), r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
