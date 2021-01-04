#!/usr/bin/env sh
# config a single ipfs node

function config() {
    NUM=$1
    API_PORT=$2
    GATEWAY_PORT=$3
    SWARM_PORT=$4
    while [ -f /data/ipfs/repo.lock ]
    do
        echo "waiting for repo unlock..."
        cat /data/ipfs/repo.lock
        sleep 1s
    done
    docker exec ipfs-node${NUM} ipfs config Addresses.API /ip4/0.0.0.0/tcp/${API_PORT}
    docker exec ipfs-node${NUM} ipfs config Addresses.Gateway /ip4/0.0.0.0/tcp/${GATEWAY_PORT}
    docker exec ipfs-node${NUM} ipfs config Addresses.Swarm "[\"/ip4/0.0.0.0/tcp/$SWARM_PORT\", \"/ip6/::/tcp/$SWARM_PORT\", \"/ip4/0.0.0.0/udp/$SWARM_PORT/quic\", \"/ip6/::/udp/$SWARM_PORT/quic\"]" --json
    # delete public bootstrap node:
    docker exec ipfs-node${NUM} ipfs bootstrap rm --all
}

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

function print_blue() {
  printf "${BLUE}%s${NC}\n" "$1"
}

function print_green() {
  printf "${GREEN}%s${NC}\n" "$1"
}

function print_red() {
  printf "${RED}%s${NC}\n" "$1"
}