package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

type CachetIncident struct {
	Id          int `json:"id"`
	ComponentId int `json:"component_id"`
	Status      int `json:"status"`
}

// Cachet is a facade to CachetHQ client calls
type Cachet interface {
	// List will fetch the different CachetHQ components (id/name) via a GET /api/v1/components
	// it will return a map[componentname]componentid
	ListComponents() (map[string]int, error)

	SearchComponent(name string) (int, error)

	// Returns all incidents for a given component, ASC sorted (i.e. the last incident, is the first in the list)
	SearchIncidents(componentId int) ([]*CachetIncident, error)

	// CreateIncident will create a new incident for the choosen CachetHQ components (id/name) via a POST /api/v1/incidents
	// component status: component status: https://docs.cachethq.io/docs/component-statuses
	// - status = 1 for alert resolved
	// - status = 4 for alert fatal
	CreateIncident(componentName string, componentID, status int, componentStatus int) error

	// UpdateIncident will create a new incident update for the choosen CachetHQ components (id/name) via a PUT /api/v1/incidents/<incidentid>
	// component status: component status: https://docs.cachethq.io/docs/component-statuses
	// - status = 1 for alert resolved
	// - status = 4 for alert fatal
	UpdateIncident(componentName string, componentID, incidentId, status int) error
}

// cf https://docs.cachethq.io/reference#update-a-component
// {
//    "data": {
//        "id": 1,
//        "name": "Component Name",
//        "description": "Description",
//        "link": "",
//        "status": 1,
//        "order": 0,
//        "group_id": 0,
//        "created_at": "2015-08-01 12:00:00",
//        "updated_at": "2015-08-01 12:00:00",
//        "deleted_at": null,
//        "status_name": "Operational",
//        "tags": [
//            "slug-of-tag": "Tag Name"
//        ]
//    }
//}
type cachetHqMessage struct {
	Status int `json:"status"`
}

// cf https://docs.cachethq.io/reference#get-components
// {
//    "meta": {
//        "pagination": {
//            "total": 1,
//            "count": 1,
//            "per_page": 20,
//            "current_page": 1,
//            "total_pages": 1,
//            "links": {
//                "next_page": null,
//                "previous_page": null
//            }
//        }
//    },
//    "data": [
//        {
//            "id": 1,
//            "name": "API",
//            "description": "This is the Cachet API.",
//            "link": "",
//            "status": 1,
//            "order": 0,
//            "group_id": 0,
//            "created_at": "2015-07-24 14:42:10",
//            "updated_at": "2015-07-24 14:42:10",
//            "deleted_at": null,
//            "status_name": "Operational",
//          	"tags": [
//            		"slug-of-tag": "Tag Name"
//            ]
//        }
//    ]
//}
type cachetHqComponentList struct {
	Meta struct {
		Pagination struct {
			CurrentPage int `json:"current_page"`
			TotalPages  int `json:"total_pages"`
		} `json:"pagination"`
	} `json:"meta"`
	Data []struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
	} `json:"data"`
}

type cachetHqIncidemntsList struct {
	Meta struct {
		Pagination struct {
			CurrentPage int `json:"current_page"`
			TotalPages  int `json:"total_pages"`
		} `json:"pagination"`
	} `json:"meta"`
	Data []CachetIncident `json:"data"`
}

// cf https://docs.cachethq.io/reference#incidents
type cachetHqIncident struct {
	Name            string `json:"name"`
	Message         string `json:"message"`
	Status          int    `json:"status"`
	Visible         int    `json:"visible"`
	ComponentID     int    `json:"component_id"`
	ComponentStatus int    `json:"component_status"`
}

type CachetImpl struct {
	apiURL string
	apiKey string
	client *http.Client
}

// NewCachetImpl creates a new Cachet interface implementation
func NewCachetImpl(apiURL, apiKey string, client *http.Client) *CachetImpl {
	// by precaution, remove the '/' at the end of apiURL
	apiURL = strings.TrimRight(apiURL, "/")

	return &CachetImpl{
		apiURL: apiURL,
		apiKey: apiKey,
		client: client,
	}
}

func (c *CachetImpl) ListComponents() (map[string]int, error) {
	componentsID := make(map[string]int)
	var message cachetHqComponentList

	// we loop "only" on the max first 100 pages
	for page := 1; page < 100; page++ {
		nextPage := fmt.Sprintf("%s/api/v1/components?page=%d", c.apiURL, page)

		req, err := http.NewRequest(http.MethodGet, nextPage, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Cachet-Token", c.apiKey)

		resp, err := c.client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if resp.StatusCode != 200 {
			if err != nil {
				return nil, err
			}
			log.Println(string(body))
		}

		if err := json.Unmarshal(body, &message); err != nil {
			return nil, err
		}

		for _, data := range message.Data {
			componentsID[data.Name] = data.Id
		}

		// is there a next page?
		if message.Meta.Pagination.CurrentPage >= message.Meta.Pagination.TotalPages {
			// nope
			return componentsID, nil
		}
	}
	return componentsID, nil
}

func (c *CachetImpl) SearchComponent(name string) (int, error) {
	var message cachetHqComponentList

	page := fmt.Sprintf("%s/api/v1/components?name=%s&page=1", c.apiURL, name)

	req, err := http.NewRequest(http.MethodGet, page, nil)
	if err != nil {
		return -1, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cachet-Token", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		if err != nil {
			return -1, err
		}
		log.Println(string(body))
	}

	if err := json.Unmarshal(body, &message); err != nil {
		return -1, err
	}

	if len(message.Data) == 1 {
		return message.Data[0].Id, nil
	}

	return -1, fmt.Errorf("no component found")
}

func (c *CachetImpl) CreateIncident(componentName string, componentID, status int, componentStatus int) error {
	incidentName := fmt.Sprintf("%s down", componentName)
	incidentMessage := fmt.Sprintf("Prometheus flagged service %s as down", componentName)
	incidentStatus := 2 // "Identified"

	// if we are in status = 1 (alert resolved)
	if status == 1 {
		incidentName = fmt.Sprintf("%s up", componentName)
		incidentMessage = fmt.Sprintf("Prometheus flagged service %s as recovered", componentName)
		incidentStatus = 4 // "Fixed"
	}

	incident := &cachetHqIncident{
		Name:            incidentName,
		Message:         incidentMessage,
		Status:          incidentStatus,
		ComponentID:     componentID,
		Visible:         1,
		ComponentStatus: componentStatus,
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(incident); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/incidents", c.apiURL), &buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cachet-Token", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		log.Println(string(b))
	}
	//body, _ := ioutil.ReadAll(resp.Body)
	//log.Println("response from CachetHQ when sending alert: ", string(body))

	return nil
}

func (c *CachetImpl) UpdateIncident(componentName string, componentID, incidentId, status int) error {
	incidentName := fmt.Sprintf("%s down", componentName)
	incidentMessage := fmt.Sprintf("Prometheus flagged service %s as down", componentName)
	incidentStatus := 2 // "Identified"

	// if we are in status = 1 (alert resolved)
	if status == 1 {
		incidentName = fmt.Sprintf("%s up", componentName)
		incidentMessage = fmt.Sprintf("Prometheus flagged service %s as recovered", componentName)
		incidentStatus = 4 // "Fixed"
	}

	incident := &cachetHqIncident{
		Name:            incidentName,
		Message:         incidentMessage,
		Status:          incidentStatus,
		ComponentID:     componentID,
		Visible:         1,
		ComponentStatus: incidentStatus,
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(incident); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/api/v1/incidents/%d", c.apiURL, incidentId), &buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cachet-Token", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		log.Println(string(b))
	}
	return nil
}

func (c *CachetImpl) SearchIncidents(componentId int) ([]*CachetIncident, error) {
	incidents := make([]*CachetIncident, 0)
	var message cachetHqIncidemntsList

	// we loop "only" on the max first 100 pages
	for page := 1; page < 100; page++ {
		nextPage := fmt.Sprintf("%s/api/v1/incidents?component_id=%d&sort=id&order=desc", c.apiURL, componentId)

		req, err := http.NewRequest(http.MethodGet, nextPage, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Cachet-Token", c.apiKey)

		resp, err := c.client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if resp.StatusCode != 200 {
			if err != nil {
				return nil, err
			}
			log.Println(string(body))
		}

		if err := json.Unmarshal(body, &message); err != nil {
			return nil, err
		}

		for _, data := range message.Data {
			copydata := data
			incidents = append(incidents, &copydata)
		}

		// is there a next page?
		if message.Meta.Pagination.CurrentPage >= message.Meta.Pagination.TotalPages {
			// nope
			return incidents, nil
		}
	}
	return incidents, nil
}
