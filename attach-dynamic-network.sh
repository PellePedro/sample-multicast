#!/bin/bash

set -e

containerNames=(halo1 halo2 halo3)
pids=()

# Fetch pids from Docker Containers
for container in "${containerNames[@]}"
do
    pid=$(docker inspect -f '{{.State.Pid}}' ${container})
    pids+=($pid)
done

# Make Docker netns visible under /var/run/netns
for i in "${!containerNames[@]}"; do
  pidNS="/proc/${pids[$i]}/ns/net"
  nameNS="/var/run/netns/${containerNames[i]}"
  printf "Symlinking %s -> %s\n" "${nameNS}" "${pidNS}"
  sudo ln -sfT ${pidNS} ${nameNS}
done

# Configure virtual links
create_virtual_link() {
    ns1=$1
    veth1=$2
    addr1=$3
    ns2=$4
    veth2=$5
    addr2=$6

    ip link add ${veth1} netns ${ns1} type veth peer name ${veth2} netns ${ns2}
    ip net exec ${ns1} ip addr add ${addr1} dev ${veth1}
    ip net exec ${ns2} ip addr add ${addr2} dev ${veth2}
    ip net exec ${ns1} ip link set ${veth1} up
    ip net exec ${ns2} ip link set ${veth2} up
}

# Create and Provision links
create_virtual_link halo1 halo_12 10.10.1.1/24 halo2 halo_21 10.10.1.2/24
create_virtual_link halo1 halo_13 10.10.3.1/24 halo3 halo_31 10.10.3.3/24
create_virtual_link halo2 halo_23 10.10.2.2/24 halo2 halo_32 10.10.2.3/24

# Test Connectivity
sudo ip net exec halo1 ping 10.10.1.2 -c 1
sudo ip net exec halo1 ping 10.10.3.3 -c 1
sudo ip net exec halo2 ping 10.10.2.3 -c 1

echo "Completed"
