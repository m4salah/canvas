package integrationtest

import (
	"net/http"
	"testing"
	"time"

	"canvas/server"
)

// CreateServer for testing on port 8081, returning a cleanup function that stops the server.
// Usage:
//
//	cleanup := CreateServer()
//	defer cleanup()
func CreateServer() func() {
	db, cleanupDB := CreateDatabase()
	queue, cleanupQueue := CreateQueue()
	s := server.New(server.Options{
		Database: db,
		Host:     "localhost",
		Port:     8081,
		Queue:    queue,
	})

	go func() {
		if err := s.Start(); err != nil {
			panic(err)

		}
	}()

	for {
		_, err := http.Get("http://localhost:8081/")

		if err == nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	return func() {
		if err := s.Stop(); err != nil {
			panic(err)
		}
		cleanupDB()
		cleanupQueue()
	}
}

// SkipIfShort skips t if the "-short" flag is passed to "go test".
func SkipIfShort(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
}
