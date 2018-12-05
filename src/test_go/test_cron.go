package main

import (
	"carmiddleware/log"
	"github.com/robfig/cron"
)

func main() {
	log.InitLogger("test_cron")
	carMiddlewareCon := cron.New()
	spec := "*/3 * * * * ?"
	carMiddlewareCon.AddFunc(spec, func() {
		log.Info("testsetsetstststst")
	})
	carMiddlewareCon.Start()
	defer carMiddlewareCon.Stop()

	select {}
}
