package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

type config struct {
	BabelCtl       string
	InfluxDBAddr   string
	InfluxDBToken  string
	InfluxDBOrg    string
	InfluxDBBucket string
}

func (s *config) init(path string) *config {
	s.BabelCtl = "/run/higgs.ctl"
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	if err = json.Unmarshal(data, s); err != nil {
		log.Fatal(err)
	}
	if s.BabelCtl == "" || s.InfluxDBAddr == "" || s.InfluxDBToken == "" || s.InfluxDBOrg == "" || s.InfluxDBBucket == "" {
		log.Fatal("missing value in config file")
	}
	return s
}
