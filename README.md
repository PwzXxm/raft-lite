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
├── sample          # Sample configurations
├── simulation/     # Simulation for testing locally
├── sm/             # State machine
└── utils/          # Utils
main.go             # main entrance
```

## Example Config
Exmaple configuration files are under [`sample`](https://github.com/PwzXxm/raft-lite/tree/master/sample) folder. The `NodeAddrMap` should be the same in all peers and client.

## Usage

### Start

#### Docker-compose
To build docker image
```bash
docker build -t raft-lite .
```

To start peers
```bash
docker-compose up
```

To start clientA and clientB
```bash
./raft-lite peer -c sample/docker-compose/client-A.json
./raft-lite peer -c sample/docker-compose/client-B.json
```

#### Manually
On different computer or separate terminal on the same computer, run chosen configuration.
To start with a size of five, you need to start five peers using different configuration files.
```bash
./raft-lite peer -c sample/config/sample-config1.json
```

Start client with client configuration JSON file to send requests
```bash
./raft-lite client -c sample/config/client_config.json
```

### Simulation
The purpose of simulation is to test the core of the raft algorithm on one local machine.
Events such as network glitches(delay, packet loss, etc.), network partition, peer shutdown and restart, are emulated.
```bash
# Local simulation with 5 peers
./raft-lite simulation local -n 5
```

### Unit tests
All core components - persistent storage and RPC have unit tests and RPC has benchmark tests to illustrate the performance.

```bash
go test -v ./...
go test -v ./... -bench=.
```

### Functional tests
Functional tests treat the raft system as a black box and check if the outputs meet the requirements.

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
