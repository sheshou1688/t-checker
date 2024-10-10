package main

import (
	"log"
	"os"
	"t-checker/checker"
	"time"

	"gopkg.in/ini.v1"
)

func main() {

	cur, _ := os.Getwd()
	cfg, err := ini.Load(cur + "/my.ini")
	if err != nil {
		log.Fatalf("Fail to read file: %v", err)
	}
	threads := cfg.Section("").Key("threads").MustInt(10)
	date := cfg.Section("").Key("date").MustString(time.Now().Format("2006-01-02"))

	cr := checker.NewChecker(date)
	cr.Init(threads)
	cr.Run()
}
