package main

import (
	"flag"
	"fmt"
	"os"
	"os/user"
	"net"
	"strings"
	"encoding/json"
	"io/ioutil"
	"os/signal"
	"log"
)

// Declare global variables
const MaxInt = int(^uint(0) >> 1)
const MaxUdpSize = 2000 // TODO: arbitrary, make configurable?
const MaxAckSize = 40 // TODO: arbitrary, make configurable?
const OriginChannelSize = 1000 // TODO: arbitrary, make configurable?
var host string
var proto string
var udpForwardPort string
var logger *log.Logger

// Route consists of a Terminus and a slice of Origins
type Route struct {
	Terminus string
	Origins []string
}

// RouteConfig consists of a slice of Routes
type RouteConfig struct {
	Routes []Route
}

// CheckError panics if any error is encountered
func CheckError(err error) {
	if err != nil {
		logger.Panicln("PANIC:", err.Error())
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

// HandleSIGINT traps SIGINT, saves routes, and exits (goroutine)
func HandleSIGINT(sigChan chan os.Signal, routesMap map[string][]string, routesJsonPath string) {
	<- sigChan
	_ = SaveRoutes(routesMap, routesJsonPath)
	logger.Println("----- BLU EXITED -----")
	os.Exit(0)
}

// ForwardUDP forwards a buffer to a terminus over UDP (goroutine)
func ForwardUDP(terminus string, buf []byte) {
	// set terminus
	terminusAddr, err := net.ResolveUDPAddr(proto, terminus)
	CheckError(err)

	// set local
	localAddr, err := net.ResolveUDPAddr(proto, host + ":" + udpForwardPort)
	CheckError(err)

	// dial connection
	relay, err := net.DialUDP(proto, localAddr, terminusAddr)
	CheckError(err)
	defer relay.Close()

	// forward udp
	_, err = relay.Write(buf)
	CheckError(err)
}

// BlockingForwardUDP forwards a buffer to a terminus over UDP after waiting for other worker to finish (goroutine)
func BlockingForwardUDP(terminus string, buf []byte, done chan bool) {
	<- done
	ForwardUDP(terminus, buf)
	done <- true
}

// ForwardACK listens for ACKs sent back from terminus and forwards to origins from originChannel (goroutine)
func ForwardACK(ackPort string, originConn *net.UDPConn, originChannel chan *net.UDPAddr) {
	// start ACK listener
	addr, err := net.ResolveUDPAddr(proto, host + ":" + ackPort)
	CheckError(err)
	conn, err := net.ListenUDP(proto, addr)
	CheckError(err)
	defer conn.Close()

	// set maximum payload per ACK packet
	ack := make([]byte, MaxAckSize)

	// start event loop
	for {
		n, terminus, err := conn.ReadFromUDP(ack)
		CheckError(err)
		if terminus != nil { // TODO: does terminus come from termini?
			// dequeue a origin address from the originChannel
			originAddr := <- originChannel

			// send ack to originAddron originConn
			_, err = originConn.WriteTo(ack[0:n], originAddr)
			CheckError(err)
		}
	}
}

// BalanceLoadUDP initializes routesMaps and starts a UDP flow
func BalanceLoadUDP(termini, routesJsonPath, port, ackPort string, ackForward bool) {
	// exit if termini not set
	if len(termini) == 0 {
		logger.Fatalln("FATAL: termini must be set! exiting!")
	}

	// initialize and load routesMap
	routesMap := InitRoutes(termini)
	LoadRoutes(routesMap, routesJsonPath)

	// handle SIGINT to save routes on exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go HandleSIGINT(sigChan, routesMap, routesJsonPath)

	// start UDP listener
	addr, err := net.ResolveUDPAddr(proto, host + ":" + port)
	CheckError(err)
	conn, err := net.ListenUDP(proto, addr)
	CheckError(err)
	defer conn.Close()

	// set maximum payload per UDP packet
	buf := make([]byte, MaxUdpSize)

	// start event loop
	if ackForward == false {
		if ackPort != "0" {
			logger.Fatalln("FATAL: ack-forward and ack-port must both be set! exiting!")
		} else { // plain forward case
			if udpForwardPort != "0" { // contention subcase
				// create and initialize done channel
				done := make(chan bool, 1)
				done <- true

				// call BlockingForwardUDP to handle contention
				for {
					n, origin, err := conn.ReadFromUDP(buf)
					CheckError(err)
					terminus := DetermineTerminus(routesMap, origin.String())
					go BlockingForwardUDP(terminus, buf[0:n], done)
				}
			} else { // no contention subcase
				for {
					n, origin, err := conn.ReadFromUDP(buf)
					CheckError(err)
					terminus := DetermineTerminus(routesMap, origin.String())
					go ForwardUDP(terminus, buf[0:n])
				}
			}
		}
	} else {
		if ackPort == "0" { // TODO: forward UDP, forward ACK
			logger.Fatalln("FATAL: unimplemented! exiting!")
		} else { // forward udp, forward ack + dedicated listener
			originChannel := make(chan *net.UDPAddr, OriginChannelSize)

			// start ACK listener
			go ForwardACK(ackPort, conn, originChannel)

			if udpForwardPort != "0" { // contention subcase
				// create and initialize done channel
				done := make(chan bool, 1)
				done <- true

				// call BlockingForwardUDP to handle contention
				for {
					n, origin, err := conn.ReadFromUDP(buf)
					CheckError(err)

					// push connection early to ensure ACK will get forwarded
					originChannel <- origin

					terminus := DetermineTerminus(routesMap, origin.String())
					go BlockingForwardUDP(terminus, buf[0:n], done)
				} 
			} else { // no contention case
				for {
					n, origin, err := conn.ReadFromUDP(buf)
					CheckError(err)

					// push connection early to ensure ACK will get forwarded
					originChannel <- origin

					terminus := DetermineTerminus(routesMap, origin.String())
					go ForwardUDP(terminus, buf[0:n])
				}
			}
		}
	}
}

// main starts program
func main() {
	// core flags
	hostPtr := flag.String("host", "127.0.0.1", "UDP listener host")
	portPtr := flag.String("port", "8080", "UDP listener port")
	terminiPtr := flag.String("termini", "", "comma-separated list of termini to route packets to")
	resetPtr := flag.Bool("reset", false, "clear all origins in routes json")

	// default path flags
	usr, _ := user.Current()
	defaultRoutesJsonPath := usr.HomeDir + "/.bluroutes.json"
	defaultLogFilePath := usr.HomeDir + "/.blu.log"
	routesJsonPtr := flag.String("routes-json", defaultRoutesJsonPath, "file to persist routes json")
	logFilePtr := flag.String("log-file", defaultLogFilePath, "file to log prints, fatals, and panics")

	// optional flags
	udpForwardPortPtr := flag.String("udp-forward-port", "0", "forward UDP to terminus using specified outgoing port (same host)")
	ackForwardPtr := flag.Bool("ack-forward", false, "forward ACK back to origin")
	ackPortPtr := flag.String("ack-port", "0", "activate ACK listener to listen on specified port (same host)")

	// parse flags
	flag.Parse()

	// initialize global variables
	host = *hostPtr
	proto = GetProto(host)
	udpForwardPort = *udpForwardPortPtr
	logFile, err := os.OpenFile(*logFilePtr, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0644)
	if err != nil {
		fmt.Println("could not open log file for writing!")
		os.Exit(1)
	}
	defer logFile.Close()
	logger = log.New(logFile, "", log.LstdFlags)
	logger.Println("----- BLU STARTED -----")

	// start flows
	if *resetPtr {
		// stop user if other flags provided; logFile and routesJsonPtr are okay
		if udpForwardPort != "0" || *ackForwardPtr || *ackPortPtr != "0" || *terminiPtr != "" || *portPtr != "8080" || host != "127.0.0.1" {
			logger.Fatalln("FATAL: multiple modes specified! exiting!")
		}
		ResetRoutes(*routesJsonPtr)
	} else {
		BalanceLoadUDP(*terminiPtr, *routesJsonPtr, *portPtr, *ackPortPtr, *ackForwardPtr)
	}
}