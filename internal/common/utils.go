package common

import (
	"log"
	"os"
)

func InitLog(logFile string) (logger *log.Logger) {
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("error opening file %s for log dump. error %v", logFile, err)
		logger = log.New(os.Stdout, "Client Log: ", log.LstdFlags)
	} else {
		logger = log.New(file, "Client Log: ", log.LstdFlags)
	}
	defer file.Close()
	return
}
