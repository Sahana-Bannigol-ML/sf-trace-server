package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"fmt"
	"time"
	"os"
	"errors"
	"strconv"

	retryhttp "github.com/hashicorp/go-retryablehttp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// Response holds Pipeline metrics
type Response struct {
	Libbeat Pipeline `json:"libbeat"`
}

// Pipeline holds Events data
type Pipeline struct {
	Event EventData `json:"pipeline"`
}

// EventData holds map for event data metrics
type EventData struct {
	Metric map[string]interface{} `json:"events"`
}

const(
	traceEndPoint = "http://localhost:5066/stats"
	serverPort          = ":2112"
	serverMetricsPath = "/metrics"
)

var (
	// Gauge for adding custom metric value in prometheus
	gauge = prometheus.NewGauge(
			prometheus.GaugeOpts{
					Name: "sftrace_queued_events",
					Help: "Total number of queued events",
			})

	logger, _ = zap.NewProduction()
)

// HTTPClientWithRetry returns retryable http client
func HTTPClientWithRetry() *retryhttp.Client {
	client := retryhttp.NewClient()
	client.RetryWaitMin = 500 * time.Millisecond
	client.RetryMax = 3
	return client
}

// getActiveCount returns active queued count from trace server end point localhost:5066/stats
func getActiveCount() (float64, error) {
	var activeCount float64
	client := HTTPClientWithRetry()
	response, err := client.Get(traceEndPoint)
	if err != nil {
		logger.Error(err.Error())
		return activeCount, err
	}

	if response.StatusCode != 200 {
		errStr := fmt.Sprintf("Could not connect to end point %s", traceEndPoint)
		logger.Error(errStr)
		return activeCount, errors.New(errStr)
	}

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		logger.Error(err.Error())
		return activeCount, err
	}

	var responseObject Response
	err = json.Unmarshal(responseData, &responseObject)
	if err != nil {
		return activeCount, err
	}
	logger.Info(fmt.Sprintf("%v",responseObject.Libbeat.Event.Metric))
	if val, ok := responseObject.Libbeat.Event.Metric["active"].(float64); ok {
		return val, nil
	}
	return activeCount, errors.New("Key active count not found in Libbeat Metrics")
}

func main() {
	metricsInterval := int64(5)
    metricsInterval, _ = strconv.ParseInt(os.Getenv("Metrics_Interval"), 6, 12)
	// Starting go routine loop to capture Queued Events regularly at 1s interval.
	go func() {
		for {
			activeCount, err := getActiveCount()
			if err != nil {
				logger.Error(err.Error())
				continue
			}
			gauge.Set(activeCount)
			time.Sleep(time.Duration(metricsInterval) * time.Second)
		}
	}()
	http.Handle(serverMetricsPath, promhttp.Handler())
	prometheus.MustRegister(gauge)
	http.ListenAndServe(serverPort, nil)
}