#!/usr/bin/env bash 

set -e
cat <<EOF | sudo bash
echo "Installing Docker and Compose"

apt update
apt install -y apt-transport-https ca-certificates curl software-properties-common

# Installing Docker Compose
mkdir -p "$HOME/.docker/cli-plugins"
curl -L https://github.com/docker/compose/releases/download/v2.0.0-rc.3/docker-compose-linux-amd64 \
  -o $HOME/.docker/cli-plugins/docker-compose
chmod +x $HOME/.docker/cli-plugins/docker-compose

curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu focal stable"
apt update
apt-cache policy docker-ce
apt-get -y install docker-ce docker-ce-cli containerd.io

usermod -aG docker ${USER}
systemctl enable docker
systemctl start docker
systemctl status docker

EOF
