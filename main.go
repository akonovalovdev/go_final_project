package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/akonovalovdev/go_final_project/db"
	"github.com/akonovalovdev/go_final_project/handlers"
)

func main() {
	dbFilePath := db.GetDBFilePath()
	db.InitDB(dbFilePath)

	port := os.Getenv("TODO_PORT")
	if port == "" {
		fmt.Println("TODO_PORT not set, using default port 7540")
		port = "7540"
	}

	http.Handle("/", http.FileServer(http.Dir("./web")))

	http.HandleFunc("/api/nextdate", handlers.NextDateHandler)
	http.HandleFunc("/api/task", handlers.TaskHandler)
	http.HandleFunc("/api/task/done", handlers.TaskDoneHandler)
	http.HandleFunc("/api/tasks", handlers.TasksListHandler)

	log.Printf("Server starting on port %s\n", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
