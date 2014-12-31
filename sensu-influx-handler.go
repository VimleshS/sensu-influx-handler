package main

import (
	"code.google.com/p/gcfg"
	"encoding/json"
	"fmt"
	"github.com/influxdb/influxdb/client"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	CONN_HOST = "localhost"
	CONN_PORT = "3333"
	CONN_TYPE = "tcp"
)

type Client struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}
type Check struct {
	Output string `json:"output"`
}
type Event struct {
	Client Client `json:"client"`
	Check  Check  `json:"check"`
}

type Config struct {
	Influx client.ClientConfig `gcfg:"influxdb"`
}

var Global Config

func init() {
	var err error
	var dir string
	var configFile string
	mode := os.Getenv("SENSU_INFLUX_MODE")

	dir, err = filepath.Abs(filepath.Dir(os.Args[0]))

	if mode == "production" {
		configFile = dir + "/sensu-influx.prod.conf"
	} else if mode == "staging" {
		configFile = dir + "/sensu-influx.stg.conf"
	} else {
		configFile = dir + "/sensu-influx.local.conf"
	}

	err = gcfg.ReadFileInto(&Global, configFile)

	if err != nil {
		panic("Unable to read configuration file from: " + configFile)
	}
}

func main() {
	// Listen for incoming connections.
	l, err := net.Listen(CONN_TYPE, CONN_HOST+":"+CONN_PORT)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	c, err := client.NewClient(&Global.Influx)
	if err != nil {
		panic(err)
	}

	// Close the listener when the application closes.
	defer l.Close()
	fmt.Println("Listening on " + CONN_HOST + ":" + CONN_PORT)
	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
		} else {
			// Handle connections in a new goroutine.
			go handleRequest(conn, c)
		}
	}
}

// Handles incoming requests.
func handleRequest(conn net.Conn, c *client.Client) {
	defer conn.Close()
	// Make a buffer to hold incoming data.
	buf := make([]byte, 102400)
	// Read the incoming connection into the buffer.
	numbytes, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading:", err.Error())
		return
	}

	var evt Event
	err = json.Unmarshal(buf[:numbytes], &evt)
	if err != nil {
		fmt.Printf("Error unmarshalling event: %s for input %s\n", err.Error(), string(buf[:numbytes]))
		return
	}

	outputlines := strings.Split(strings.TrimSpace(evt.Check.Output), "\n")

	seriesdata := make(map[string][][]interface{})

	for _, l := range outputlines {
		line := strings.TrimSpace(l)
		pieces := strings.Split(line, " ")
		if len(pieces) != 3 {
			continue
		}
		keys := strings.SplitN(pieces[0], ".", 2)
		if len(keys) != 2 {
			continue
		}
		keyraw := keys[1]
		key := strings.Replace(keyraw, ".", "_", -1)

		val, verr := strconv.ParseFloat(pieces[1], 64)
		if verr != nil {
			fmt.Printf("Error parsing value (%s): %s\n", pieces[1], verr.Error())
			continue
		}

		time, terr := strconv.ParseInt(pieces[2], 10, 64)
		if terr != nil {
			fmt.Printf("Error parsing time (%s): %s\n", pieces[2], terr.Error())
			continue
		}

		seriesdata[key] = append(seriesdata[key], []interface{}{time, evt.Client.Name, evt.Client.Address, val})
	}

	serieses := make([]*client.Series, 0)
	for key, points := range seriesdata {
		series := &client.Series{
			Name:    key,
			Columns: []string{"time", "host", "ip", "value"},
			Points:  points,
		}
		serieses = append(serieses, series)
	}

	if err := c.WriteSeriesWithTimePrecision(serieses, client.Second); err != nil {
		fmt.Printf("Error sending data to influx: %s, data: %+v\n", err.Error(), serieses)
	}

}
