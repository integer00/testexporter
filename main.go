/*

1) create basic listener on 8080 port with /metrics endpoint
2) do call to remote url and store result
3) do a periodical data retrieval
4) initialize a prometheus exporter that:
     *) nodeHealth gauge, which define status of node - 1 node is healthy, 0 node is not healthy
     *) nodeHealthTotalRequests counter, that shows total num of requests

EXAMPLE RESPONSE : {"data":{"ready":true,"time":"2020-05-01T12:13:44.202131883Z"}}

*/

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	nodeHealth = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "node_health",
		Help: "Current node health status",
	})
	nodeHealthTotalRequests = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "node_health_total",
		Help: "Sum of requests",
	})
)

type myJson struct {
	Data Data
}

type Data struct {
	Ready bool      `json:"ready"`
	Time  time.Time `json:"time"`
}

func doRequest(s string) []byte {
	fmt.Printf("requesting for %s\n", s)
	resp, err := http.Get(s)
	if err != nil {
		log.Printf("An error accured, %s", err.Error())
		return nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err.Error())
	}

	err = resp.Body.Close()
	if err != nil {
		panic(err.Error())
	}

	return body
}

func getRoutes() {
	http.HandleFunc("/", myHandler)
	http.Handle("/metrics", promhttp.Handler())
}

func myHandler(w http.ResponseWriter, req *http.Request) {
	_, err := fmt.Fprintf(w, "Default handler, hello there")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fprintf: %v\n", err)
	}
}

func backgroundTask(s string, t int) {
	ticker := time.NewTicker(time.Duration(t) * time.Second)

	for _ = range ticker.C {
		nodeHealthTotalRequests.Inc()

		jsonString := doRequest(s)
		if jsonString == nil {
			fmt.Println("failed to retrieve data")
			continue
		}

		result := new(myJson)

		//jsonString := `{"data":{"ready":true,"time":"2020-05-01T12:13:44.202131883Z"}}`
		err := json.Unmarshal([]byte(jsonString), &result)
		if err != nil {
			fmt.Println("failed to unmarshal data")
			continue
		}

		// check response
		if !result.Data.Ready {
			fmt.Println("(LOG) service is not ready")
			nodeHealth.Set(0)
		} else {
			log.Println("(LOG) service is OK")
			nodeHealth.Set(1)
		}
	}
}

func init() {
	prometheus.MustRegister(nodeHealthTotalRequests, nodeHealth)
}

func main() {
	var listen = flag.String("listen", "localhost:8080", "a webserver listen to address")
	var address = flag.String("address", "http://google.com", "an help for address flag")
	var requestInterval = flag.Int("interval", 15, "how often poll server")
	flag.Parse()

	getRoutes()

	//requestUrl := os.Getenv("TEST_URL")
	//if requestUrl == "" {
	//	requestUrl = *address
	//}

	//result := doRequest(requestUrl)
	//if result == nil {
	//	fmt.Println("failed to retrieve data")
	//}

	go backgroundTask(*address, *requestInterval)

	fmt.Printf("Starting web server at %s\n", *listen)
	log.Fatal(http.ListenAndServe(*listen, nil))
}
