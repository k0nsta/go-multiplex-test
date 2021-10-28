package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/k0nsta/go-multiplex-test/internal/server"
	"github.com/k0nsta/go-multiplex-test/internal/server/handlers"
)

const (
	defaultAddress    = ":9000"
	defaultMaxConn    = 100
	defaultMaxOutConn = 4
	defaultMaxOutTask = 20
	defaultTimeout    = 1 // sec
)

func main() {
	var (
		bindAdr    string
		maxConn    uint
		maxOutConn uint
		maxOutTask uint
		timeout    uint
	)

	flag.StringVar(&bindAdr, "bind", defaultAddress, "TCP server address")
	flag.UintVar(&maxConn, "in", defaultMaxConn, "Maximum number of simultaneous client connections, 0 is unlimited")
	flag.UintVar(&maxOutConn, "out", defaultMaxOutConn, "Maximum number of simultaneous outcoming connections for each client request")
	flag.UintVar(&maxOutTask, "task", defaultMaxOutTask, "Maximum number of tasks (urls) for each client request")
	flag.UintVar(&timeout, "timeout", defaultTimeout, "Maximum timeout in seconds for outcoming connection")

	router := http.NewServeMux()
	router.Handle("/collector", handlers.NewCollector(
		maxConn,
		maxOutConn,
		maxOutTask,
		time.Second*time.Duration(timeout)),
	)

	srv := server.New(&server.Config{
		BindAddr: bindAdr,
		Handler:  router,
	})

	errChan := srv.Serve()
	if err := <-errChan; err != nil {
		log.Fatalf("failed while running server: %s\n", err)
	}
}
