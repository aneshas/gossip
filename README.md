# Gossip

Self-hosted nats streaming / redis / golang chat

## Prerequisites
To run gossip locally, you need to have `docker`, `docker-compose` and `go` installed and on your path.

## Run
- Clone this repo, eg. `git clone https://github.com/tonto/gossip.git ~/go/src/github.com/tonto/gossip`
- Change to `~/go/src/github.com/tonto/gossip`
- Run `./up` script which will compile the binary and run `docker-compose up` with gossip, nats-streaming and redis
- If everything went fine, you should now have gossip running on `localhost` (port 80)


