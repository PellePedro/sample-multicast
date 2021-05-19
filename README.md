# Multicast Test

## Description
A small project to test a 3 node docker-compose network with PWOSPF multicast.
The nodes in the nework is simulated as docker-conntaioners with multiple network interfaces.
Each node also connects to a simulates metric server ocer grps to fetch link flow statistic to
simulate network jitter,latency and badwith constrains.

## Usage
The provided Makefile supports the following commands

```
build-grpc-server              - Building mock metrics (GRPC) Server
build-halo                     - Building Halo Container
generate-proto-stubs           - Generate Protonuff Stubs
help                           - Show help message
purge-simulation               - Purge Simulation
run-simulation                 - Run Simulation (i.e run 3 containers and the link metric server)
```


## Configuration
See docker compose file for recommended environment variables and arguments.

## Simulation
The PWOSPF simulation requires configuration of multipple networks in each container.
To attach PWOSPF interfaces (and network) run the script  'sudo ./attach-dynamic-network.sh'

The new interfaces will be auto detected and used for PWOSPF broadcast. 