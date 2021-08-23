package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

var babelID string
var _db influxdb2.Client
var _api api.WriteAPI
var _xroute []string

func getClient(c *config) influxdb2.Client {
	_db = influxdb2.NewClient(c.InfluxDBAddr, c.InfluxDBToken)
	if ready, _ := _db.Ready(context.Background()); ready {
		log.Printf("connected to db")
	} else {
		log.Fatal("could not connect to db")
	}
	_api = _db.WriteAPI(c.InfluxDBOrg, c.InfluxDBBucket)
	return _db
}

type handler func(string)

func loopOnce(c *config, h handler) error {
	_xroute = make([]string, 0)
	conn, err := net.Dial("unix", c.BabelCtl)
	if err != nil {
		return err
	}
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		msg = strings.Trim(msg, "\n")
		if msg == "ok" {
			break
		}
		h(msg)
	}
	conn.Write([]byte("dump\n"))
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		msg = strings.Trim(msg, "\n")
		if msg == "ok" {
			break
		}
		h(msg)
	}
	p := influxdb2.NewPointWithMeasurement("xroute").
		AddTag("id", babelID).
		AddField("routes", strings.Join(_xroute, ",")).
		SetTime(time.Now())
	_api.WritePoint(p)
	_api.Flush()
	return nil
}

func loop(c *config, h handler) error {
	errChannel := _api.Errors()
	go func(c <-chan error) {
		for {
			err := <-c
			if !strings.Contains(err.Error(), "EOF") {
				log.Fatal(err)
			}
		}
	}(errChannel)

	for {
		if err := loopOnce(c, h); err != nil {
			log.Print(err)
		} else {
			log.Print("dump successed")
		}
		time.Sleep(10 * time.Second)
	}
}

func allHandler(message string) {
	if message == "" {
		return
	}
	m := strings.Split(message, " ")
	switch m[0] {
	case "my-id":
		babelID = m[1]
	case "add":
		lineHandler(m[1:])
	case "change":
		lineHandler(m[1:])
	}
}

func convertLinetoMap(message []string) map[string]string {
	result := make(map[string]string)
	if len(message)%2 != 0 {
		return result
	} else {
		for i := 0; i < len(message)/2; i++ {
			result[message[2*i]] = message[2*i+1]
		}
	}
	return result
}

func lineHandler(m []string) {
	data := convertLinetoMap(m)
	switch m[0] {
	case "neighbour":
		neighbourHandler(data)
	case "route":
		routeHandler(data)
	case "xroute":
		xrouteHandler(data)
	}
}

func neighbourHandler(m map[string]string) {
	reach, err := strconv.ParseUint(m["reach"], 16, 16)
	if err != nil {
		return
	}
	reachCount := 0
	for reach > 0 {
		reach = reach & (reach - 1)
		reachCount++
	}
	rtt, _ := strconv.ParseFloat(m["rtt"], 64)
	if rtt > 5000 {
		return
	}
	cost, _ := strconv.ParseUint(m["cost"], 10, 64)
	p := influxdb2.NewPointWithMeasurement("neighbour").
		AddTag("id", babelID).AddTag("to", m["address"]).AddTag("if", m["if"]).
		AddField("reach", reachCount).AddField("rtt", rtt).AddField("cost", cost).
		SetTime(time.Now())
	_api.WritePoint(p)
}

func routeHandler(m map[string]string) {
	i := "false"
	if m["installed"] == "yes" {
		i = "true"
	}
	metric, _ := strconv.ParseUint(m["metric"], 10, 64)
	rffmetric, _ := strconv.ParseUint(m["rffmetric"], 10, 64)
	if metric > 5000 {
		return
	}
	p := influxdb2.NewPointWithMeasurement("route").
		AddTag("id", babelID).AddTag("source", m["id"]).AddTag("prefix", m["prefix"]).AddTag("from", m["from"]).
		AddTag("installed", i).AddTag("via", m["via"]).AddTag("if", m["if"]).
		AddField("metric", metric).AddField("rffmetric", rffmetric).
		SetTime(time.Now())
	_api.WritePoint(p)
}

func xrouteHandler(m map[string]string) {
	_xroute = append(_xroute, fmt.Sprintf("%s@%s", m["prefix"], m["from"]))
}
