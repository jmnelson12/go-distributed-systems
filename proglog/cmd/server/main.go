package main

import (
	"log"

	"github.com/jmnelson12/distributed-systems/proglog/internal/server"
)

func main() {
	l := server.NewLog()
	srv := server.NewHTTPServer(":8080", l)

	log.Fatal(srv.ListenAndServe())
}
