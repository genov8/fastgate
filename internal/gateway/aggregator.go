package gateway

import (
	"encoding/json"
	"fastgate/internal/config"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
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
		log.Printf("Error requesting %s (%s): %v", call.Name, url, err)
		resultChan <- map[string]interface{}{
			"__error__": map[string]interface{}{
				"service":  call.Name,
				"error":    "Service unavailable",
				"critical": call.Required,
			},
		}

		if !call.Required {
			resultChan <- map[string]interface{}{call.Name: nil}
		}
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response from %s: %v", call.Name, err)
		resultChan <- map[string]interface{}{
			"__error__": map[string]interface{}{
				"service":  call.Name,
				"error":    "Invalid response from service",
				"critical": call.Required,
			},
		}

		if !call.Required {
			resultChan <- map[string]interface{}{call.Name: nil}
		}
		return
	}

	var jsonData interface{}
	if err := json.Unmarshal(body, &jsonData); err != nil {
		log.Printf("Error parsing JSON from %s: %v", call.Name, err)
		resultChan <- map[string]interface{}{
			"__error__": map[string]interface{}{
				"service":  call.Name,
				"error":    "Invalid JSON response",
				"critical": call.Required,
			},
		}

		if !call.Required {
			resultChan <- map[string]interface{}{call.Name: nil}
		}
		return
	}

	log.Printf("Successful response from %s", call.Name)
	resultChan <- map[string]interface{}{call.Name: jsonData}
}

func (a *Aggregator) AggregateData(route config.Aggregation, pathParams map[string]string, req *http.Request) map[string]interface{} {
	var wg sync.WaitGroup
	resultChan := make(chan map[string]interface{}, len(route.Calls))

	queryParams := extractQueryParams(req)
	headerParams := extractHeaderParams(req)

	errors := make([]map[string]interface{}, 0)

	for _, call := range route.Calls {
		wg.Add(1)

		allParams := mergeParams(pathParams, queryParams, headerParams)
		resolvedParams := resolveParams(call, pathParams, queryParams, headerParams)
		for key, value := range resolvedParams {
			allParams[key] = value
		}

		url := call.Backend
		rePattern := regexp.MustCompile(`\{(\w+)\}`)
		missingParam := false
		missingParamName := ""

		url = rePattern.ReplaceAllStringFunc(url, func(match string) string {
			paramName := match[1 : len(match)-1]
			if val, exists := allParams[paramName]; exists && val != "" {
				return val
			}
			missingParam = true
			missingParamName = paramName
			return ""
		})

		if missingParam {
			log.Printf("⚠️ Missing parameter '%s' for service '%s'.", missingParamName, call.Name)
			wg.Done()

			errors = append(errors, map[string]interface{}{
				"service":  call.Name,
				"error":    fmt.Sprintf("Missing required parameter: %s", missingParamName),
				"critical": call.Required,
			})

			if !call.Required {
				resultChan <- map[string]interface{}{call.Name: nil}
			}
			continue
		}

		go a.fetchData(call, url, &wg, resultChan)
	}

	wg.Wait()
	close(resultChan)

	finalResponse := make(map[string]interface{})

	for result := range resultChan {
		if errData, exists := result["__error__"]; exists {
			errors = append(errors, errData.(map[string]interface{}))
			continue
		}

		for key, value := range result {
			if key != "__error__" {
				finalResponse[key] = value
			}
		}
	}

	if len(errors) > 0 {
		finalResponse["error"] = errors
	}

	return finalResponse
}

func mergeParams(pathParams, queryParams, headerParams map[string]string) map[string]string {
	merged := make(map[string]string)

	for key, value := range pathParams {
		merged[key] = value
	}

	for key, value := range queryParams {
		if _, exists := merged[key]; !exists {
			merged[key] = value
		}
	}

	for key, value := range headerParams {
		if _, exists := merged[key]; !exists {
			merged[key] = value
		}
	}

	return merged
}

func resolveParams(call config.Call, pathParams, queryParams, headerParams map[string]string) map[string]string {
	resolved := make(map[string]string)

	for key, source := range call.Params {
		parts := strings.Split(source, ".")
		if len(parts) != 2 {
			continue
		}

		prefix, param := parts[0], parts[1]

		switch prefix {
		case "$path":
			if val, exists := pathParams[param]; exists {
				resolved[key] = val
			}
		case "$query":
			if val, exists := queryParams[param]; exists {
				resolved[key] = val
			}
		case "$header":
			if val, exists := headerParams[param]; exists {
				resolved[key] = val
			}
		}
	}

	return resolved
}

func extractQueryParams(req *http.Request) map[string]string {
	params := make(map[string]string)
	query := req.URL.Query()

	for key, values := range query {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}
	return params
}

func extractHeaderParams(req *http.Request) map[string]string {
	params := make(map[string]string)
	for key, values := range req.Header {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}
	return params
}
