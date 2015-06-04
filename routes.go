package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
)

// Route consists of a Terminus and a slice of Origins
type Route struct {
	Terminus string
	Origins []string
}

// RouteConfig consists of a slice of Routes
type RouteConfig struct {
	Routes []Route
}

// LoadRoutes loads a json file into a routesMap and returns routesMap
func LoadRoutes(routesMap map[string][]string, routesJsonPath string) {
	var routeConfig RouteConfig
	if _, err := os.Stat(routesJsonPath); err == nil {
		routesJson, _ := ioutil.ReadFile(routesJsonPath)
		_ = json.Unmarshal(routesJson, &routeConfig)
		for _, route := range routeConfig.Routes {
			if _, ok := routesMap[route.Terminus]; ok {
				routesMap[route.Terminus] = route.Origins
			}
		}
	}
}

// InitRoutes initializes routes and returns routesMap
func InitRoutes(termini string) map[string][]string {
	routesMap := make(map[string][]string)
	for _, n := range strings.Split(termini, ",") {
		routesMap[n] = make([]string, 0)
	}
	return routesMap
}

// SaveRoutes saves a routesMap into a json file
func SaveRoutes(routesMap map[string][]string, routesJsonPath string) error {
	// convert map to RouteConfig
	var routeConfig RouteConfig
	routeConfig.Routes = make([]Route, 0)
	for terminus, origins := range routesMap {
		route := Route{Terminus: terminus, Origins: origins}
		routeConfig.Routes = append(routeConfig.Routes, route)
	}
	data, _ := json.MarshalIndent(routeConfig, "", " ")
	return ioutil.WriteFile(routesJsonPath, data, 0644)
}

// ResetRoutes loads a json file if it exists and clears all origins
func ResetRoutes(routesJsonPath string) {
	var routeConfig RouteConfig
	if _, err := os.Stat(routesJsonPath); err == nil {
		// load routes
		routesJson, _ := ioutil.ReadFile(routesJsonPath)
		_ = json.Unmarshal(routesJson, &routeConfig)

		// clear origins
		for i := range routeConfig.Routes {
			routeConfig.Routes[i].Origins = make([]string, 0)
		}

		// save routes
		data, _ := json.MarshalIndent(routeConfig, "", " ")
		_ = ioutil.WriteFile(routesJsonPath, data, 0644)
	} else {
		logger.Fatalln("FATAL: no json to reset! exiting!")	
	}
}

// DetermineTerminus determines best terminus to send traffic to
func DetermineTerminus(routesMap map[string][]string, origin string) string {
	// ASSUMPTION: all termini are reachable

	var leastLoaded string
	minLength := MaxInt

	for terminus, origins := range routesMap {
		// set least-loaded terminus
		length := len(origins)
		if length < minLength {
			minLength = length
			leastLoaded = terminus
		}

		for _, i := range origins {
			// if origin found, return that terminus
			if i == origin {
				return terminus
			}
		}
	}

	// origin not found so append origin to leastLoaded terminus
	routesMap[leastLoaded] = append(routesMap[leastLoaded], origin)

	// return least-loaded
	return leastLoaded
}