package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Task struct {
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

func main() {
	dbFilePath := os.Getenv("TODO_DBFILE")
	if dbFilePath == "" {
		currentDir, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		dbFilePath = filepath.Join(currentDir, "scheduler.db")
	}

	db, err := sql.Open("sqlite", dbFilePath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS scheduler (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		date TEXT NOT NULL,
		title TEXT NOT NULL,
		comment TEXT,
		repeat TEXT CHECK(length(repeat) <= 128)
	);`)

	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_scheduler_date ON scheduler(date);`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = os.Stat(dbFilePath)
	if os.IsNotExist(err) {
		log.Fatalf("File %s was not created", dbFilePath)
	} else if err != nil {
		log.Fatal(err)
	} else {
		log.Printf("Database file %s has been created successfully", dbFilePath)
	}

	port := os.Getenv("TODO_PORT")
	if port == "" {
		fmt.Println("TODO_PORT not set, using default port 7540")
		port = "7540"
	}

	http.Handle("/", http.FileServer(http.Dir("./web")))

	http.HandleFunc("/api/nextdate", func(w http.ResponseWriter, r *http.Request) {
		nowStr := r.FormValue("now")
		dateStr := r.FormValue("date")
		repeatStr := r.FormValue("repeat")

		now, err := time.Parse("20060102", nowStr)
		if err != nil {
			http.Error(w, "Invalid now parameter", http.StatusBadRequest)
			return
		}

		nextDate, err := NextDate(now, dateStr, repeatStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Write([]byte(nextDate))
	})

	http.HandleFunc("/api/task", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var task Task
			err := json.NewDecoder(r.Body).Decode(&task)
			if err != nil {
				http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
				return
			}

			if task.Title == "" {
				http.Error(w, `{"error":"Title is required"}`, http.StatusBadRequest)
				return
			}

			if task.Date != "" {
				_, err = time.Parse("20060102", task.Date)
				if err != nil {
					http.Error(w, `{"error":"Invalid date format"}`, http.StatusBadRequest)
					return
				}
			}

			if task.Date == "" || task.Date < time.Now().Format("20060102") {
				task.Date = time.Now().Format("20060102")
			}

			if task.Repeat == "d 1" {
				task.Date = time.Now().Format("20060102")
			} else if task.Repeat != "" {
				task.Date, err = NextDate(time.Now(), task.Date, task.Repeat)
				if err != nil {
					http.Error(w, `{"error":"Invalid repeat rule"}`, http.StatusBadRequest)
					return
				}
			}

			res, err := db.Exec(`INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)`,
				task.Date, task.Title, task.Comment, task.Repeat)
			if err != nil {
				http.Error(w, `{"error":"Failed to insert task"}`, http.StatusInternalServerError)
				return
			}

			id, err := res.LastInsertId()
			if err != nil {
				http.Error(w, `{"error":"Failed to get task ID"}`, http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			json.NewEncoder(w).Encode(map[string]string{"id": strconv.FormatInt(id, 10)})
		default:
			http.Error(w, `{"error":"Invalid method"}`, http.StatusMethodNotAllowed)
		}
	})

	log.Printf("Server starting on port %s\n", port)
	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func NextDate(now time.Time, date string, repeat string) (string, error) {
	startDate, err := time.Parse("20060102", date)
	if err != nil {
		return "", err
	}

	if repeat == "" {
		return "", errors.New("repeat rule is empty")
	}

	repeatParts := strings.Split(repeat, " ")

	switch repeatParts[0] {
	case "d":
		if len(repeatParts) != 2 {
			return "", errors.New("invalid repeat rule format")
		}

		days, err := strconv.Atoi(repeatParts[1])
		if err != nil || days > 400 {
			return "", errors.New("invalid number of days")
		}

		for {
			startDate = startDate.AddDate(0, 0, days)

			if !startDate.Before(now) && !startDate.Equal(now) {
				break
			}
		}

	case "y":
		for {
			startDate = startDate.AddDate(1, 0, 0)

			if !startDate.Before(now) && !startDate.Equal(now) {
				break
			}
		}

	default:
		return "", errors.New("unsupported repeat rule format")
	}

	return startDate.Format("20060102"), nil
}
