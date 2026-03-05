package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/generate", handleGenerate)
	http.HandleFunc("/", handleStatic)

	fmt.Println("🚀 Сервер запущен: http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}