package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func waitForSignal(logger *log.Logger) {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	s := <-ch
	if logger != nil {
		logger.Printf("Received signal \"%v\"", s)
	}
}
