package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// the mock cachetHQ defines 1 component: "Component21"
func TestCachetListComponents(t *testing.T) {
	var ts *httptest.Server
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/api/v1/components" {
			w.Header().Set("Content-Type", "aplication/json")
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, `{
				    "meta": {
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
				            "name": "API",
				            "description": "This is the Cachet API.",
				            "link": "",
				            "status": 1,
				            "order": 0,
				            "group_id": 0,
				            "created_at": "2015-07-24 14:42:10",
				            "updated_at": "2015-07-24 14:42:10",
				            "deleted_at": null,
				            "status_name": "Operational",
				          	"tags": [
				            		{"slug-of-tag": "Tag Name"}
				            ]
				        }
				    ]
				}`)
		} else if r.Method == "GET" && r.URL.Path == "/api/v1/incidents" {
			w.Header().Set("Content-Type", "aplication/json")
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, `{
				"meta": {
					"pagination": {
						"total": 1,
						"count": 1,
						"per_page": "20",
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
						"id": 2,
						"component_id": 1,
						"name": "Incident Name",
						"status": 1,
						"visible": 1,
						"message": "Incident Message",
						"scheduled_at": "2015-08-01 12:00:00",
						"created_at": "2015-08-01 12:00:00",
						"updated_at": "2015-08-01 12:00:00",
						"deleted_at": null,
						"human_status": "Fixed"
					}
				]
			}`)
		} else if r.Method == "POST" && r.URL.Path == "/api/v1/incidents" {
			w.Header().Set("Content-Type", "aplication/json")
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, `{
				"data": {
					"id": 4,
					"component_id": 1,
					"name": "API down",
					"status": 1,
					"visible": 1,
					"message": "API down",
					"scheduled_at": "2015-08-01 12:00:00",
					"created_at": "2015-08-01 12:00:00",
					"updated_at": "2015-08-01 12:00:00",
					"deleted_at": null,
					"human_status": "Fixed"
				}
			}`)
		} else if r.Method == "PUT" && r.URL.Path == "/api/v1/incidents/4" {
			w.Header().Set("Content-Type", "aplication/json")
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, `{
					"data": {
						"id": 1,
						"component_id": 1,
						"name": "Incident Name",
						"status": 4,
						"visible": 1,
						"message": "Incident Message",
						"scheduled_at": "2015-08-01 12:00:00",
						"created_at": "2015-08-01 12:00:00",
						"updated_at": "2015-08-01 12:00:00",
						"deleted_at": null,
						"human_status": "Fatal"
					}
				}`)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, `{"status":"fail"}`)
		}
	}))
	defer ts.Close()

	cachet := NewCachetImpl(ts.URL, "undefined", ts.Client())

	// test list components
	listComponents, err := cachet.ListComponents()
	assert.Nil(t, err)
	assert.Equal(t, 1, len(listComponents))
	assert.Equal(t, 1, listComponents["API"])

	// test search component
	componentID, err := cachet.SearchComponent("API")
	assert.Nil(t, err)
	assert.Equal(t, 1, componentID)

	// test list incidents
	listIncidents, err := cachet.SearchIncidents(1)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(listIncidents))
	assert.Equal(t, 2, listIncidents[0].Id)
	assert.Equal(t, 1, listIncidents[0].Status)

	err = cachet.CreateIncident("API", 1, 1, 4)
	assert.Nil(t, err)

	err = cachet.UpdateIncident("API", 1, 4, 4)
	assert.Nil(t, err)
}
