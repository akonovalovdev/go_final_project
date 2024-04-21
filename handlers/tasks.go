package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/akonovalovdev/go_final_project/db"
	"github.com/akonovalovdev/go_final_project/models"
	"github.com/akonovalovdev/go_final_project/utils"
)

func TaskHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var task models.Task
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
			task.Date, err = utils.NextDate(time.Now(), task.Date, task.Repeat)
			if err != nil {
				http.Error(w, `{"error":"Invalid repeat rule"}`, http.StatusBadRequest)
				return
			}
		}

		res, err := db.DB.Exec(`INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)`,
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
}
