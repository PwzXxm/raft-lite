# raft-lite

[![build](https://github.com/PwzXxm/raft-lite/workflows/build/badge.svg)](https://github.com/PwzXxm/raft-lite/actions?query=workflow%3Abuild+event%3Apush+branch%3Amaster)

## Group 2

| Student Name | Student ID |
| ------------ | ---------- |
| Minjian Chen | 813534     |   
| Shijie Liu   | 813277     |   
| Weizhi Xu    | 752454     |   
| Wenqing Xue  | 813044     |   
| Zijun Chen   | 813190     |

## Project Details
The focus of this project is to explore the detailed implementation of the Raft algorithm, which is based on [In Search of an Understandable Consensus Algorithm](https://raft.github.io/raft.pdf) by Diego Ongaro and John Ousterhout.

On top of that, we build a transaction system that supports several operations, including
- Set account amount
- Increment/decrement the account balance
- Transfer deposits from account A to account B
- Query the current account balance

By using the raft consensus algorithm as the backend, we can tolerate at most half of the machines (exclusive) disconnected from the cluster, because of network issues or shutting down unexpectedly.
Therefore, it secures the transaction system to be available most of the time and ensures its
consistency at the same time.

### Raft Features
- Leader election
- Log replication
- Log compaction by taking snapshots

### Folder Structure
```bash
/
├── client/         # Client config, command parse and request
├── functests/      # Functional test cases
├── pstorage/       # Persistent storage
├── raft/           # Raft algorithm
├── rpccore/        # RPC implementation - channel and tcp version
├── sample/         # Sample configurations
├── simulation/     # Simulation for testing locally
├── sm/             # State machine
└── utils/          # Utils
main.go             # Main entrance
```

## Example Config
Exmaple configuration files are under [`sample`](https://github.com/PwzXxm/raft-lite/tree/master/sample) folder. The `NodeAddrMap` should be the same in all peers and client.

## Usage

Note that you can always use `./raft-lite --help`.

### Start Raft Peer

#### Docker-compose
To build docker image
```bash
docker build -t raft-lite .
```

To start peers (with the sample config)
```bash
cd sample/docker-compose
docker-compose up
```

#### Manually

Build executable
```bash
go build .
```

On different computer or separate terminal on the same computer, run chosen configuration.
To start with a size of five, you need to start five peers using different configuration files.
```bash
./raft-lite peer -c sample/config/sample-config1.json
./raft-lite peer -c sample/config/sample-config2.json
./raft-lite peer -c sample/config/sample-config3.json
./raft-lite peer -c sample/config/sample-config4.json
./raft-lite peer -c sample/config/sample-config5.json
```

### Start Raft Client

Build executable for client
```bash
go build .
```

To start client
```bash
# for sample docker-compose version
./raft-lite client -c sample/docker-compose/client-A.json
# for sample docker-compose version
./raft-lite client -c sample/docker-compose/client-B.json
# for sample manual version
./raft-lite client -c sample/config/client_config.json
```
Note that you can't use multiple client instances with the same config at the same time, the clientID should be different.

You can start entering command once you see prompt like this:
```
=============================================
______          __  _      _  _  _
| ___ \        / _|| |    | |(_)| |
| |_/ /  __ _ | |_ | |_   | | _ | |_   ___
|    /  / _` ||  _|| __|  | || || __| / _ \
| |\ \ | (_| || |  | |_   | || || |_ |  __/
\_| \_| \__,_||_|   \__|  |_||_| \__| \___|


 Welcome to Raft Lite Transaction System

=============================================
>
```
The supported commands are:
```
Usage: <cmd> <args> ...

Commands:
	increment   <key> <value>
	loggerLevel <level> (warn, info, debug, error)
	move        <source> <target> <value>
	query       <key>
	set         <key> <value>
```

## Testing

Raft is a system that can be very hard to test and verify. We provide 3 type of tests (unit, functional and integration) and a simulation tool.
We run all unit and functional tests in our continue integration so all changes we made are verified.

### Unit test

The code under `./raft/` only contains the core logic of raft and we use dependency injection to let it uses other components like persistent storage (`./pstorage/`), network (`./rpccore/`) and state machine (`./sm/`). Each component has multiple implementations, eg. rpccore has a tcp version and a go channel version (for testing). We use unit test to make sure they work as we want.

```bash
go test -v ./...
go test -v ./... -bench=.
```

### Functional tests
Functional tests treat the raft system as a black box and check if the outputs meet the requirements. We design those cases manually.

To list all functional test cases
```bash
./raft-lite functionaltest list
```

To run single functional test case 10
```bash
./raft-lite functionaltest run 10
```

To run all functional test cases in parallel
```bash
python3 functional_tests.py
python3 functional_tests.py --parallel 4 --timeout 120 --times 10
```

### Integration test
Integration test mimics different network actions and client actions, such as
- network packets loss rate
- network partition
- disconnect nodes from the network
- restart peer
- client request to set account balance
- client request to transfer account balance
- client request to increment/decrement account balance
- client query data

All these events are randomly occurred according to given weights.
A local state machine is maintained to check the correctness of the queries.
Also, snapshot and log entries are checked periodically.

```bash
./raft-lite integrationtest -t <minutes>
```

### Simulation
The purpose of simulation is to test the core of the raft algorithm on one local machine.
Events such as network glitches(delay, packet loss, etc.), network partition, peer shutdown and restart, are emulated.
```bash
# Local simulation with 5 peers
./raft-lite simulation local -n 5
```
