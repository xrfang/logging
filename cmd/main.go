package main

import (
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/xrfang/logging"
)

func task() {
	log := logging.Open("task.log")
	defer func() {
		log.Flush()
		fmt.Println("flushed:", log.Path())
	}()
	defer log.Catch(func(l logging.Logger, e interface{}) {
		if e != nil {
			fmt.Println("catched something, which we've already logged.")
		}
		l.Print("program quitting...")
	})
	buf := make([]byte, 512)
	rand.Read(buf)
	log.Dump(buf, "%d bytes of random data", len(buf))
	panic(errors.New("something went wrong"))
}

func main() {
	logging.Init("", logging.LevelTrace, &logging.Options{Split: 10240, Keep: 3})
	defer logging.Finish()
	fmt.Println("logs are stored at:", logging.Path())
	log := logging.Open("app.log")
	log.Print("Application launched")
	task()
	log.Print("Task finished")
}
