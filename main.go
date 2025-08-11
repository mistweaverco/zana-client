package main

import (
	"io"
	"log"
	"os"

	"github.com/mistweaverco/zana-client/cmd/zana"
)

func main() {
	f, err := os.OpenFile("/tmp/zana.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			log.Printf("Warning: failed to close log file: %v", closeErr)
		}
	}()
	wrt := io.Writer(f)
	log.SetOutput(wrt)
	log.Println("Zana client started")
	zana.Execute()
}
