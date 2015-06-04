package main

import (
	"net"
)

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
	ack := make([]byte, maxAckSize)

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