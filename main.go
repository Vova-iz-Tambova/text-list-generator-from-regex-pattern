package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	// Флаги командной строки
	useHTTPS := flag.Bool("https", false, "Запустить HTTPS сервер")
	port := flag.String("port", "8080", "Порт для сервера")
	certFile := flag.String("cert", "cert.pem", "Путь к сертификату")
	keyFile := flag.String("key", "key.pem", "Путь к ключу")
	flag.Parse()

	http.HandleFunc("/generate", handleGenerate)
	http.HandleFunc("/cancel", handleCancel)
	http.HandleFunc("/", handleStatic)

	if *useHTTPS {
		// Проверка наличия сертификатов
		if _, err := os.Stat(*certFile); os.IsNotExist(err) {
			fmt.Printf("❌ Файл %s не найден\n", *certFile)
			fmt.Println("💡 Создай сертификаты:")
			fmt.Println("   openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes -subj \"/CN=localhost\"")
			fmt.Println("   Или запусти без -https для HTTP режима")
			return
		}

		addr := ":" + *port
		if *port == "8080" {
			addr = ":8443" // Default HTTPS port
		}

		fmt.Printf("🔒 HTTPS сервер запущен на https://localhost%s\n", addr)
		fmt.Println("   (прими самоподписанный сертификат в браузере)")

		server := &http.Server{
			Addr: addr,
			TLSConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		}

		if err := server.ListenAndServeTLS(*certFile, *keyFile); err != nil {
			log.Fatal(err)
		}
	} else {
		addr := ":" + *port
		fmt.Printf("🚀 HTTP сервер запущен на http://localhost%s\n", addr)

		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Fatal(err)
		}
	}
}