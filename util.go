package main

import (
	"strings"
	"os"
)

// CheckError panics if any error is encountered
func CheckError(err error) {
	if err != nil {
		logger.Panicln("PANIC:", err.Error())
	}
}

// GetProto determines the protocol for UDP traffic and returns it as a string
func GetProto(host string) string {
	var proto string
	if strings.Contains(host, ".") && strings.Contains(host, ":") {
		logger.Fatalln("FATAL: invalid host provided! exiting!")
	}
	if strings.Contains(host, ".") {
		proto = "udp4"
	}
	if strings.Contains(host, ":") {
		proto = "udp6"
	}
	return proto
}

// HandleSIGINT traps SIGINT, saves routes, and exits (goroutine)
func HandleSIGINT(sigChan chan os.Signal, routesMap map[string][]string, routesJsonPath string) {
	<- sigChan
	_ = SaveRoutes(routesMap, routesJsonPath)
	logger.Println("----- BLU EXITED -----")
	os.Exit(0)
}