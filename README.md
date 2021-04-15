# Multicast Test

## Description
A small project to test multicast with sample OSPF Hello.

## Configuration
The name of the container interface might be set with the CONTAINER_INTERFACE
environment variable (default is eth0). Local ip is retrived from the last entry in /etc/hosts

## Compilation
```
make build
```

## Run 3 nodes with docker compose
```
docker-compose up

```

