package main

import (
	"fmt"
	"net/http"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/handlers"
)

// main регистрирует HTTP-обработчики и запускает сервер на порту 8080.
func main() {
	http.HandleFunc("/", handlers.MainPage)
	http.HandleFunc("/health", handlers.Health)

	fmt.Println("сервер запускается на http://localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
