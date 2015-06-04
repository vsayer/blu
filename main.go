package main

import (
	"flag"
	"fmt"
	"os"
	"os/user"
	"net"
	"os/signal"
	"log"
)

// Declare global variables
const MaxInt = int(^uint(0) >> 1)
var host string
var proto string
var udpForwardPort string
var originChannelSize int
var maxUdpSize int
var maxAckSize int
var logger *log.Logger

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
	buf := make([]byte, maxUdpSize)

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
			originChannel := make(chan *net.UDPAddr, originChannelSize)

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

	// tuning flags
	originChannelSizePtr := flag.Int("origin-channel-size", 1000, "set size of ACK listener's origin channel")
	maxUdpSizePtr := flag.Int("max-udp-size", 2000, "set maximum size of UDP packet")
	maxAckSizePtr := flag.Int("max-ack-size", 40, "set maximum size of ACK packet")

	// parse flags
	flag.Parse()

	// initialize global variables
	host = *hostPtr
	proto = GetProto(host)
	udpForwardPort = *udpForwardPortPtr
	originChannelSize = *originChannelSizePtr
	maxUdpSize = *maxUdpSizePtr
	maxAckSize = *maxAckSizePtr
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