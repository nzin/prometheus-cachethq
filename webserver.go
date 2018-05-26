package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

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

// cachetList will fetch the different CachetHQ components (id/name) via a GET /api/v1/components
// it will return a map[componentname]componentid
func cachetList(apiURL, apiKEY string) (map[string]int, error) {
	componentsId := make(map[string]int)
	var message cachetHqMessageList

	// by precaution, remove the '/' at the end of apiURL
	apiURL = strings.TrimRight(apiURL, "/")

	// we loop "only" on the max first 100 pages
	for page := 1; page < 100; page++ {
		nextPage := fmt.Sprintf("%s/api/v1/components?page=%d", apiURL, page)

		req, err := http.NewRequest(http.MethodGet, nextPage, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Cachet-Token", apiKEY)

		client := &http.Client{}
		resp, err := client.Do(req)
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
			componentsId[data.Name] = data.Id
		}

		// is there a next page?
		if message.Meta.Pagination.CurrentPage >= message.Meta.Pagination.TotalPages {
			// nope
			return componentsId, nil
		}
	}
	return componentsId, nil
}

// alert will update the choosen CachetHQ components (id/name) via a PUT /api/v1/components/<componentid>
// component status: component status: https://docs.cachethq.io/docs/component-statuses
// - status = 1 for alert resolved
// - status = 4 for alert fatal
func cachetAlert(componentid, status int, apiURL, apiKEY string) error {
	msg := &cachetHqMessage{
		Status: status,
	}

	// by precaution, remove the '/' at the end of apiURL
	apiURL = strings.TrimRight(apiURL, "/")

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(msg); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/api/v1/components/%d", apiURL, componentid), &buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cachet-Token", apiKEY)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	//body, _ := ioutil.ReadAll(resp.Body)
	//log.Println("response from CachetHQ when sending alert: ", string(body))

	return nil
}

/*
cf https://prometheus.io/docs/alerting/configuration/#webhook_config
{
	"version": "4",
	"groupKey": <string>,    // key identifying the group of alerts (e.g. to deduplicate)
	"status": "<resolved|firing>",
	"receiver": <string>,
	"groupLabels": <object>,
	"commonLabels": <object>,
	"commonAnnotations": <object>,
	"externalURL": <string>,  // backlink to the Alertmanager.
	"alerts": [
	  {
		"labels": <object>,
		"annotations": <object>,
		"startsAt": "<rfc3339>",
		"endsAt": "<rfc3339>"
	  },
	  ...
	]
  }
*/
type PrometheusAlertDetail struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartAt     string            `json:"startsAt"`
	EndsAt      string            `json:"endsAt"`
}

type PrometheusAlert struct {
	Version           string                  `json:"version" binding:"required"`
	GroupKey          string                  `json:"groupKey"`
	Status            string                  `json:"status" binding:"required"`
	Receiver          string                  `json:"receiver"`
	GroupLabels       map[string]string       `json:"groupLabels"`
	CommonLabels      map[string]string       `json:"commonLabels"`
	CommonAnnotations map[string]string       `json:"commonAnnotations"`
	ExternalURL       string                  `json:"externalURL"`
	Alerts            []PrometheusAlertDetail `json:"alerts"`
}

// SubmitAlert receive an alert from Prometheus, and try to forward it to CachetHQ
func SubmitAlert(c *gin.Context, config *PrometheusCachetConfig) {
	// check the Bearer
	bearer := c.GetHeader("Authorization")
	if bearer != fmt.Sprintf("Bearer %s", config.PrometheusToken) {
		if config.LogLevel == LOG_DEBUG {
			log.Println("wrong Authorization header:", bearer)
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "wrong Authorization header"})
		return
	}

	// read the payload
	var alerts PrometheusAlert
	if err := c.ShouldBindJSON(&alerts); err == nil {
		// talk to CachetHQ
		status := 1 // "resolved"
		if alerts.Status == "firing" {
			status = 4
		}

		list, err := cachetList(config.CachetURL, config.CachetToken)
		if err != nil {
			if config.LogLevel == LOG_DEBUG {
				log.Println(err)
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		for _, alert := range alerts.Alerts {
			// fire something
			if componentID, ok := list[alert.Labels["alertname"]]; ok {
				if err := cachetAlert(componentID, status, config.CachetURL, config.CachetToken); err != nil {
					if config.LogLevel == LOG_DEBUG {
						log.Println(err)
					}
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
			}
		}

	} else {
		if config.LogLevel == LOG_DEBUG {
			log.Println(err)
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "OK"})
}

func PrepareGinRouter(config *PrometheusCachetConfig) *gin.Engine {
	router := gin.New()
	router.Use(gin.LoggerWithWriter(gin.DefaultWriter, "/health"))
	router.Use(gin.Recovery())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "OK"})
	})

	router.POST("/alert", func(c *gin.Context) {
		SubmitAlert(c, config)
	})

	return router
}
