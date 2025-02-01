package gateway

import (
	"encoding/json"
	"fastgate/internal/config"
	"log"
	"net/http"
	"regexp"
)

type Router struct {
	Config *config.Config
}

func NewRouter(cfg *config.Config) *Router {
	return &Router{Config: cfg}
}

func matchRoute(path, pattern string) (bool, map[string]string) {
	rePattern := regexp.MustCompile(`\{(\w+)\}`)
	paramNames := rePattern.FindAllStringSubmatch(pattern, -1)

	regexStr := rePattern.ReplaceAllString(pattern, `([^/]+)`)
	regex := regexp.MustCompile("^" + regexStr + "$")

	matches := regex.FindStringSubmatch(path)
	if matches == nil {
		return false, nil
	}

	params := make(map[string]string)
	for i, param := range paramNames {
		params[param[1]] = matches[i+1]
	}
	return true, params
}

func (r *Router) HandleRequest(w http.ResponseWriter, req *http.Request) {
	log.Printf("Request received: %s %s", req.Method, req.URL.Path)

	for _, route := range r.Config.Aggregations {
		match, params := matchRoute(req.URL.Path, route.Path)
		if match {
			response := map[string]interface{}{
				"message": "🚀 API Gateway works!",
				"path":    route.Path,
				"params":  params,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	http.NotFound(w, req)
}
