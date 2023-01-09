package main

import (
	log "github.com/sirupsen/logrus"
	"os"
	"sc-tools/cmd"
	"time"
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: time.UnixDate,
		FullTimestamp:   true,
	})
}

func main() {
	_ = cmd.Execute()
}
