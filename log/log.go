package log

import (
	"fmt"
	"log"
	"os"
	"time"
)

type Logger interface {
	Println(string)
	Printf(string, ...interface{})
	Close() error
}

func New(sku string) Logger {
	f, err := os.Create(fmt.Sprintf("bot-%s.log", sku))
	if err != nil {
		log.Fatalf("%T, %v\n", err, err)
	}
	return &logger{sku: sku, logFile: f}
}

type logger struct {
	sku     string
	logFile *os.File
}

func (l *logger) Println(msg string) {
	fmt.Fprintf(l.logFile, "%s: %s\n", time.Now().Format(time.Stamp), msg)
	log.Printf("%s: %s\n", l.sku, msg)
}

func (l *logger) Printf(msg string, params ...interface{}) {
	fmtMsg := fmt.Sprintf(msg, params...)
	fmt.Fprintf(l.logFile, "%s: %s", time.Now().Format(time.Stamp), fmtMsg)
	log.Printf("%s: %s", l.sku, fmtMsg)
}

func (l *logger) Close() error {
	return l.logFile.Close()
}
