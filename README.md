# blu 
*Balance the Load for UDP*

Blu balances a UDP traffic load coming from a set of origins destined for a set of termini. In short, it is a **UDP Load Balancer**.

## Features
* supports IPv4 and IPv6
* routes are preserved on exit
* routes can be reset

## Usage
* origins do not require ack
```shell
./blu -host=<host> -port=<port> -termini=<comma-separated IP:port list>
```

* origins do not require ack **AND** forwarder requires fixed outgoing port
```shell
./blu -host=<host> -port=<port> -termini=<comma-separated IP:port list> -udp-forward-port=<port>
```

* origins require ack
```shell
./blu -host=<host> -port=<port> -termini=<comma-separated IP:port list> -ack-forward -ack-port=<port>
```

* origins require ack **AND** forwarder requires fixed outgoing port
```shell
./blu -host=<host> -port=<port> -termini=<comma-separated IP:port list> -ack-forward -ack-port=<port> -udp-forward-port=<port>
```

* reset routing table
```shell
./blu -reset
```

## Build
```shell
make
```

## Roadmap
* origins require ack (send back on forwarder)
```shell
./blu -host=<host> -port=<port> -termini=<comma-separated IP:port list> -ack-forward
```
* origins require ack (send back on forwarder) **AND** forwarder outgoing port is fixed
```shell
./blu -host=<host> -port=<port> -termini=<comma-separated IP:port list> -ack-forward -udp-forward-port=<port>
```
* refactor
* unit tests courtesy of [GoConvey](http://goconvey.co/)
* logo
* yml configuration
* init service
* [dockerization](https://docs.docker.com/userguide/dockerizing/)
* adhoc mode: add or delete termini on-the-fly
* auto-rebalancing

### Less Certain
* deep learning
* clustering for failover
* alerting

## License
BSD 3-Clause
