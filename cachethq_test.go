package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	// mux is the HTTP request multiplexer used with the test server.
	mux *http.ServeMux

	// server is a test HTTP server used to provide mock API responses.
	mockServer *httptest.Server

	// status updated by cachetHQ bridge
	finalStatus int
)

// setup sets up a test HTTP server. Tests should register handlers on
// mux which provide mock responses for the API method being tested.
func setupMockCachetHQ(t *testing.T) {
	// fake CachetHQ server
	mux = http.NewServeMux()
	mockServer = httptest.NewServer(mux)

	finalStatus = 0

	mux.HandleFunc("/api/v1/components",
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Println("mock GET /api/v1/components")
			assert.Equal(t, "1", r.FormValue("page"))
			assert.Equal(t, "GET", r.Method)
			fmt.Fprint(w, `
					{"meta": {
				        "pagination": {
				            "total": 1,
				            "count": 1,
				            "per_page": 20,
				            "current_page": 1,
				            "total_pages": 1,
				            "links": {
				                "next_page": null,
				                "previous_page": null
				            }
				        }
				    },
				    "data": [
				        {
				            "id": 1,
				            "name": "component21",
				            "description": "This is a component",
				            "link": "",
				            "status": 1,
				            "order": 0,
				            "group_id": 0,
				            "created_at": "2015-07-24 14:42:10",
				            "updated_at": "2015-07-24 14:42:10",
				            "deleted_at": null,
				            "status_name": "Operational"
				        }
				    ]}`)
		})

	mux.HandleFunc("/api/v1/incidents",
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Println("mock POST /api/v1/incidents")
			assert.Equal(t, "POST", r.Method)
			type incidentStruct struct {
				Name            string `json:"name"`
				Message         string `json:"message"`
				Status          int    `json:"status"`
				ComponentID     int    `json:"component_id"`
				ComponentStatus int    `json:"component_status"`
			}
			decoder := json.NewDecoder(r.Body)
			var incident incidentStruct
			err := decoder.Decode(&incident)
			if err == nil {
				finalStatus = incident.Status
			}
		})
}

// teardown closes the test HTTP server.
func teardown() {
	mockServer.Close()
}

// the mock cachetHQ defines 1 component: "Component21"
func TestCachetHqComponent21(t *testing.T) {
	setupMockCachetHQ(t)
	defer teardown()

	config := PrometheusCachetConfig{
		LabelName:       "alertname",
		PrometheusToken: "promToken",
		CachetURL:       mockServer.URL,
		CachetToken:     "1234567890abcdef",
		LogLevel:        LOG_DEBUG,
		HttpClient:      &http.Client{},
	}

	router := PrepareGinRouter(&config)

	server := &http.Server{
		Addr:           fmt.Sprintf(":9999"),
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go server.ListenAndServe()
	defer server.Close()

	// send an alert
	url := "http://localhost:9999/alert"

	var jsonStr = []byte(`{"receiver":"cachethq-receiver","status":"firing","alerts":[{"status":"firing","labels":{"alertname":"component21"},"annotations":{},"startsAt":"2018-05-22T20:00:32.729840058-04:00","endsAt":"0001-01-01T00:00: 00Z","generatorURL":""}],"groupLabels":{"alertname":"component21"},"commonLabels":{"alertname":"component21"},"commonAnnotations":{},"externalURL":"http://localhost.localdomain:9093","version":"4","groupKey":"{}:{alertname=\"component21\"}"}`)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	req.Header.Set("Authorization", "Bearer "+config.PrometheusToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.Nil(t, err, "Not able to send POST alert request")
	defer resp.Body.Close()

	// the status HAS been updated
	assert.Equal(t, 2, finalStatus)
}

func TestCachetHqComponent22(t *testing.T) {
	setupMockCachetHQ(t)
	defer teardown()

	config := PrometheusCachetConfig{
		PrometheusToken: "promToken",
		CachetURL:       mockServer.URL,
		CachetToken:     "1234567890abcdef",
		LogLevel:        LOG_DEBUG,
		HttpClient:      &http.Client{},
	}

	router := PrepareGinRouter(&config)

	server := &http.Server{
		Addr:           fmt.Sprintf(":9998"),
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go server.ListenAndServe()
	defer server.Close()

	// send an alert
	url := "http://localhost:9998/alert"

	var jsonStr = []byte(`{"receiver":"cachethq-receiver","status":"firing","alerts":[{"status":"firing","labels":{"alertname":"component22"},"annotations":{},"startsAt":"2018-05-22T20:00:32.729840058-04:00","endsAt":"0001-01-01T00:00: 00Z","generatorURL":""}],"groupLabels":{"alertname":"component22"},"commonLabels":{"alertname":"component22"},"commonAnnotations":{},"externalURL":"http://localhost.localdomain:9093","version":"4","groupKey":"{}:{alertname=\"component22\"}"}`)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	req.Header.Set("Authorization", "Bearer "+config.PrometheusToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.Nil(t, err, "Not able to send POST alert request")
	defer resp.Body.Close()

	// the status has NOT been updated because "component22" does not exist
	assert.Equal(t, 0, finalStatus)
}
