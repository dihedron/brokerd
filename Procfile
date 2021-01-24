# Use goreman to run `go get github.com/mattn/goreman`
setup: mkdir -p _test/
brokerd1: ./brokerd --id=node1 --http=127.0.0.1:11000 --raft=127.0.0.1:12000 --dir="_test/node0" --storage="boltdb"
brokerd2: sleep 5 && ./brokerd --id=node2 --http=127.0.0.1:11001 --raft=127.0.0.1:12001 --join=127.0.0.1:11000 --dir="_test/node1" --storage="boltdb"
brokerd3: sleep 5 && ./brokerd --id=node3 --http=127.0.0.1:11002 --raft=127.0.0.1:12002 --join=127.0.0.1:11000 --dir="_test/node2" --storage="boltdb"