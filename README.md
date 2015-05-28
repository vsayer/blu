# blu
*Balance the Load for UDP*

Blu balances a UDP traffic load coming from a set of origins destined for a set of termini. In short, it is a **UDP Load Balancer**.

## Install
Install blu as you would any other Go program:
```shell
go get github.com/vsayer/blu
```

## Usage
#### origins do not require ack
```shell
blu -host=<host> -port=<port> -termini=<comma-separated host:port list>
```

#### origins do not require ack AND forwarder requires fixed outgoing port
```shell
blu -host=<host> -port=<port> -termini=<comma-separated host:port list> \
-udp-forward-port=<port>
```

#### origins require ack
```shell
blu -host=<host> -port=<port> -termini=<comma-separated host:port list> \
-ack-forward -ack-port=<port>
```

#### origins require ack **AND** forwarder requires fixed outgoing port
```shell
blu -host=<host> -port=<port> -termini=<comma-separated host:port list> \
-ack-forward -ack-port=<port> -udp-forward-port=<port>
```

#### reset routing table
```shell
blu -reset
```

## Features
- [x] supports IPv4 and IPv6
- [x] supports origin expecting ack for reliability
- [x] ability to balance by finding least-loaded terminus
- [x] ability to fix outgoing port on forwarder
- [x] routes are preserved on exit
- [x] routes can be reset
- [ ] origins require ack (send back on forwarder)
```shell
blu -host=<host> -port=<port> -termini=<comma-separated host:port list> \
-ack-forward
```
- [ ] origins require ack (send back on forwarder) **AND** forwarder outgoing port is fixed
```shell
blu -host=<host> -port=<port> -termini=<comma-separated host:port list> \
-ack-forward -udp-forward-port=<port>
```
- [ ] unit tests courtesy of [GoConvey](http://goconvey.co/)
- [ ] travis-ci integration
- [ ] godoc integration
- [ ] peformance profiling
- [ ] refactor for more idiomatic code
- [ ] logo

## Build
If you want to develop blu, a Makefile is included and building is straightforward.
```shell
make
```

## Roadmap for v1.0
* adhoc mode: add or delete termini on-the-fly
* auto-rebalancing: for when termini go offline
* yml configuration
* init service
* [dockerization](https://docs.docker.com/userguide/dockerizing/)

## Ideation for beyond v1.0
* companion REPL client
* balancing algorithms other than least-loaded
* deep learning 
* clustering to handle restricted routes
* alerting
* DHT?

## License
[BSD 3-Clause](LICENSE)
