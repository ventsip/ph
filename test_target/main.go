package main

import (
	"log"
	"time"
)

func main() {
	for {
		log.Println("sleeping")
		time.Sleep(time.Second * 3)
	}
}
