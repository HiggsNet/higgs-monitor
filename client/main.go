package main

import (
	"flag"
	"log"
)

func main() {
	configFile := flag.String("c", "./config.json", "config file")
	flag.Parse()
	c := (&config{}).init(*configFile)
	getClient(c)
	if err := loop(c, allHandler); err != nil {
		log.Fatal(err)
	}
}
