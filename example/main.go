package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	// app "example/internal"
	"net/http"
	"time"
)

// https://github.com/ixugo/goddd

func main() {
	// app.Run()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Println("receiveing request")
		time.Sleep(5 * time.Second)
		fmt.Printf("slow response completed\n")
	})
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		http.Get("http://localhost:8080/")
	}()

	go func() {
		time.Sleep(time.Second)
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	}()

	log.Fatal(RunServer(context.Background(), server, 10*time.Second))
}

func RunServer(
	ctx context.Context,
	server *http.Server,
	shutdownTimeout time.Duration) error {

	serverErrCh := make(chan error, 1)

	go func() {
		defer close(serverErrCh)
		log.Println("starting server .....")
		if err := server.ListenAndServe(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				err = nil
			}
			serverErrCh <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err, ok := <-serverErrCh:
		if !ok {
			return nil
		}
		log.Printf("server error %v\n", err)
		return err
	case <-stop:
		log.Println("server interrupted")
	case <-ctx.Done():
		log.Println("server context cancelled")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		if closeErr := server.Close(); closeErr != nil {
			return errors.Join(err, closeErr)
		}
		return err
	}

	log.Println("server closed gracefully")

	return nil
}
