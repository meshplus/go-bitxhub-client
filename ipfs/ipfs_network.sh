#!/usr/bin/env sh
# start a ipfs local test network with 4 nodes dockerly
OPT=$1
source ipfs_config.sh

function printHelp() {
  echo "Usage:  "
  echo "  ipfs_network.sh <OPT>"
  echo "    <OPT> - one of 'up', 'down' "
  echo "      - 'up' - bring up the ipfs network with 4 nodes locally"
  echo "      - 'down' - shut down the ipfs network with 4 nodes"
  echo "  ipfs_network.sh -h (print this message)"
  echo "Default ports used: "
  echo "  ---------------------------------------------------"
  echo "  |  ******  | API_PORT | GATEWAY_PORT | SWARM_PORT |"
  echo "  |  node_1  |  15001   |     18080    |    14001   |"
  echo "  |  node_2  |  25001   |     28080    |    24001   |"
  echo "  |  node_3  |  35001   |     38080    |    34001   |"
  echo "  |  node_4  |  45001   |     48080    |    44001   |"
  echo "  ---------------------------------------------------"
}

function ipfs_network_up() {
    if ! type ipfs-swarm-key-gen >/dev/null 2>&1; then
        echo "===> Install go-ipfs-swarm-key-gen"
        go get github.com/Kubuxu/go-ipfs-swarm-key-gen/ipfs-swarm-key-gen
    fi
    if [ ! -d "$HOME/.ipfs" ]; then
        mkdir ~/.ipfs
    fi
    ipfs-swarm-key-gen > ~/.ipfs/swarm.key
    # cp ipfs_config.sh ~/.ipfs

    for i in {1..4}  
    do  
        echo "start node_$i: "
        docker run -d --name ipfs-node$i -v ~/.ipfs/swarm.key:/data/ipfs/swarm.key -v ~/.ipfs/ipfs_config.sh:/data/ipfs/ipfs_config.sh -p 127.0.0.1:${i}5001:${i}5001 -p ${i}8080:${i}8080 -p ${i}4001:${i}4001 ipfs/go-ipfs:latest
        if [ $? -ne 0 ]; then
            echo "failed."
            exit $?
        fi
    done

    for i in {1..4}
    do
        echo "config node_$i ... : "
        
        sleep 10s
        config ${i} ${i}5001 ${i}8080 ${i}4001
        if [ $? -ne 0 ]; then
            echo "failed at config.sh."
            exit $?
        fi

        if [ ${i} == 1 ]; then
            docker exec ipfs-node1 ipfs config Identity.PeerID
            if [ $? -ne 0 ]; then
                echo "failed at get PeerID."
                exit $?
            fi
            PeerID1=$(docker exec ipfs-node1 ipfs config Identity.PeerID)
        else
            docker exec ipfs-node${i} ipfs bootstrap add /ip4/127.0.0.1/tcp/14001/p2p/${PeerID1}
            if [ $? -ne 0 ]; then
                echo "failed at bootstrap add."
                exit $?
            fi
        fi
        print_green "succeed."
    done

    for i in {1..4}
    do
        docker restart ipfs-node${i}
        if [ $? -ne 0 ]; then
            echo "restart failed."
            exit $?
        else
            print_green "restart succeed!"
        fi
    done
}

function ipfs_network_down() {
    for i in {1..4}
    do
        docker stop ipfs-node${i}
        docker rm ipfs-node${i}
    done
}

if [ "$OPT" == "up" ]; then
  ipfs_network_up
elif [ "$OPT" == "down" ]; then
  ipfs_network_down
else
  printHelp
  exit 1
fi