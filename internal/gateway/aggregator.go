package gateway

import (
	"encoding/json"
	"fastgate/internal/config"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
)

type Aggregator struct {
	Config *config.Config
}

func NewAggregator(cfg *config.Config) *Aggregator {
	return &Aggregator{Config: cfg}
}

func (a *Aggregator) fetchData(call config.Call, url string, wg *sync.WaitGroup, resultChan chan<- map[string]interface{}) {
	defer wg.Done()

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Request error to %s: %v", call.Name, err)
		if call.Required {
			resultChan <- map[string]interface{}{"__error__": call.Name}
		} else {
			resultChan <- map[string]interface{}{call.Name: nil}
		}
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response from %s: %v", call.Name, err)
		if call.Required {
			resultChan <- map[string]interface{}{"__error__": call.Name}
		} else {
			resultChan <- map[string]interface{}{call.Name: nil}
		}
		return
	}

	var jsonData interface{}
	if err := json.Unmarshal(body, &jsonData); err != nil {
		log.Printf("Error parsing JSON from %s: %v", call.Name, err)
		if call.Required {
			resultChan <- map[string]interface{}{"__error__": call.Name}
		} else {
			resultChan <- map[string]interface{}{call.Name: nil}
		}
		return
	}

	log.Printf("Successful response from %s", call.Name)
	resultChan <- map[string]interface{}{call.Name: jsonData}
}

func (a *Aggregator) AggregateData(route config.Aggregation, params map[string]string) map[string]interface{} {
	var wg sync.WaitGroup
	resultChan := make(chan map[string]interface{}, len(route.Calls))

	for _, call := range route.Calls {
		wg.Add(1)

		url := call.Backend
		for key, value := range params {
			url = strings.ReplaceAll(url, "{"+key+"}", value)
		}

		go a.fetchData(call, url, &wg, resultChan)
	}

	wg.Wait()
	close(resultChan)

	finalResponse := make(map[string]interface{})
	for result := range resultChan {
		if errService, exists := result["__error__"]; exists {
			log.Printf("Critical error from %s", errService)
			return map[string]interface{}{
				"error": "Critical service '" + errService.(string) + "' is unavailable",
			}
		}

		for key, value := range result {
			finalResponse[key] = value
		}
	}

	return finalResponse
}
