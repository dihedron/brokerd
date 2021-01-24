# Use goreman to run `go get github.com/mattn/goreman`
setup: mkdir -p _test/
brokerd1: ./brokerd -id node1 -haddr 127.0.0.1:11000 -raddr 127.0.0.1:12000 _test/node0
brokerd2: sleep 10 && ./brokerd -id node2 -haddr 127.0.0.1:11001 -raddr 127.0.0.1:12001 -join 127.0.0.1:11000 _test/node1
brokerd3: sleep 10 && ./brokerd -id node3 -haddr 127.0.0.1:11002 -raddr 127.0.0.1:12002 -join 127.0.0.1:11000 _test/node2