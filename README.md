# TSF HALO


## Building Container

```
make build-container
``

## Testing with docker compose

```
make start-compose
``

[Install Docker and Compose](./hack/install-docker-compose-ubuntu.sh)

The configuration file docker-compose.yml defines a network with four nodes connected with macvlans.
The .env file defines variables used in docker-compose.yml.

```
INTERFACE=enp0s31f6    # Interface name on host used for macvlan e.g. eth0
HALO_IMAGE=halo:latest # image name
HELLO_INTERVAL_MS=100  # interval for sending OSPF HELLO
```
