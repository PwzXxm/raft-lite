# raft-lite

[![build](https://github.com/PwzXxm/raft-lite/workflows/build/badge.svg)](https://github.com/PwzXxm/raft-lite/actions?query=workflow%3Abuild+event%3Apush+branch%3Amaster)

## Group 2
Minjian Chen 813534  
Shijie Liu   813277  
Weizhi Xu    752454  
Wenqing Xue  813044  
Zijun Chen   813190

## Project Details
The focus of this project is to explore the detailed implementation of the Raft algorithm, which is based on [In Search of an Understandable Consensus Algorithm](https://raft.github.io/raft.pdf) by Diego Ongaro and John Ousterhout.

### Features
- Leader election
- Log replication
- Log compaction
- Unit test
- Functional test
- Integration test

### Folder Structure
```bash
/
├── client/         # Client
├── functests/      # Functional test cases
├── pstorage/       # Persistent storage
├── raft/           # Raft algorithm
├── rpccore/        # RPC
├── sample_config/  # Configuration files
├── simulation/     # Simulation
├── sm/             # State machine
└── utils/          # Utils
```

## Usage
### Start
```bash
#  Start peer with configuration 1 JSON file
go run . peer -c sample-config/sample-config1.json
#  Start client with client configuration JSON file
go run . client -c sample_config/client_config.json
```

### Simulation
```bash
# Local simulation with 5 peers
go run ./ simulation local -n 5
```

### Unit test
```bash
go test -v ./...
```

### Functional test
```bash
# List all functional test cases
go run ./ functionaltest list
# Run single functional test case 10
go run ./ functionaltest run 10
# Run all functional test cases
python3 functional_tests.py
#  Sample: 4 tests running in parallel, 120s timeout, 10 times for each test case
python3 functional_tests.py --parallel 4 --timeout 120 --times 10
```

### Integration test
```bash
go run ./ integrationtest
```
