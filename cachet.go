package prometheuscachethq

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

// Cachet is a facade to CachetHQ client calls
type Cachet interface {
	// List will fetch the different CachetHQ components (id/name) via a GET /api/v1/components
	// it will return a map[componentname]componentid
	ListAlerts() (map[string]int, error)

	SearchAlert(name string) (int, error)

	// Alert will update the choosen CachetHQ components (id/name) via a PUT /api/v1/components/<componentid>
	// component status: component status: https://docs.cachethq.io/docs/component-statuses
	// - status = 1 for alert resolved
	// - status = 4 for alert fatal
	Alert(componentName string, componentID, componentStatus int) error
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
type cachetHqMessageList struct {
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

// cf https://docs.cachethq.io/reference#incidents
type cachetHqIncident struct {
	Name            string `json:"name"`
	Message         string `json:"message"`
	Status          int    `json:"status"`
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

func (c *CachetImpl) ListAlerts() (map[string]int, error) {
	componentsID := make(map[string]int)
	var message cachetHqMessageList

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

		body, _ := ioutil.ReadAll(resp.Body)
		//		log.Println("response from CachetHQ when listing component's pages: ", string(body))

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

func (c *CachetImpl) SearchAlert(name string) (int, error) {
	var message cachetHqMessageList

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

	body, _ := ioutil.ReadAll(resp.Body)
	//		log.Println("response from CachetHQ when listing component's pages: ", string(body))

	if err := json.Unmarshal(body, &message); err != nil {
		return -1, err
	}

	if len(message.Data) == 1 {
		return message.Data[0].Id, nil
	}

	return -1, fmt.Errorf("no component found")
}

func (c *CachetImpl) Alert(componentName string, componentID, componentStatus int) error {
	incidentName := fmt.Sprintf("%s down", componentName)
	incidentMessage := fmt.Sprintf("Prometheus flagged service %s as down", componentName)
	incidentStatus := 2 // "Identified"

	// if we are in status = 1 (alert resolved)
	if componentStatus == 1 {
		incidentName = fmt.Sprintf("%s up", componentName)
		incidentMessage = fmt.Sprintf("Prometheus flagged service %s as recovered", componentName)
		incidentStatus = 4 // "Fixed"
	}

	incident := &cachetHqIncident{
		Name:            incidentName,
		Message:         incidentMessage,
		Status:          incidentStatus,
		ComponentID:     componentID,
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

	//body, _ := ioutil.ReadAll(resp.Body)
	//log.Println("response from CachetHQ when sending alert: ", string(body))

	return nil
}
