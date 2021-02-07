

`brokerd` is a hard fork of Philip O'Tooles [`hraftd`](http://github.com/otoolep/hraftd); it borrows the code structure -- which makes it extremely easy to study and understand the inner workings of a Raft cluster -- and extends it in order to demonstrate the use of SQLite3 as the persistent store behind the Finite State Machine, and to apply techniques such as automatic DB schema migration, automatic request redirection from Followers to the current Leader, and Raft cluster control via web APIs.  

For a background on `hraftd` check out Philip O'Tooles' [blog post](http://www.philipotoole.com/building-a-distributed-key-value-store-using-raft/).


`brokerd` makes use of the [Hashicorp Raft implementation v1.0](https://github.com/hashicorp/raft). [Raft](https://raft.github.io/) is a _distributed consensus protocol_, meaning its purpose is to ensure that a set of nodes -- a cluster -- agree on the state of some arbitrary finite state machine, even when nodes are vulnerable to failure and network partitions. Distributed consensus is a fundamental concept when it comes to building fault-tolerant distributed systems.

A simple example system like `brokerd` makes it easy to study the Raft consensus protocol in general, and Hashicorp's Raft implementation in particular. It can be run on Linux, OSX, and Windows.

## Reading and writing keys

`brokerd` uses an SQLite3 database for its key-value store. 

You can set a key by sending a request to the HTTP bind address (which defaults to `localhost:11000`):

```bash
$> curl -XPOST localhost:11000/key -d '{"foo": "bar"}'
```

You can read the value for a key like so:
```bash
$> curl -XGET localhost:11000/key/foo
```

## Running `brokerd`

_brokerd uses embed.FS; therefore it requires Go 1.16 or later._

Starting and running a `brokerd` cluster is easy. Download `brokerd` like so:
```bash
mkdir brokerd
cd brokerd/
export GOPATH=$PWD
GO111MODULE=on go get github.com/dihedron/brokerd
```

`brokerd` uses [`goreman`](https://github.com/mattn/goreman) to start a53-nodes cluster on the local machine. Once you have installed `goreman` on the local machine, open a terminal in the project root directory and start the `brokerd` cluster like this:

```bash
$> goreman start
```
Once the cluster has started up (it takes about 5 seconds to start up) you can set a key and read its value back:

```bash
$> curl -XPOST localhost:11000/key -d '{"foo": "bar"}'
$> curl -XGET localhost:11000/key/foo
```

### Bring up a cluster
_A walkthrough of setting up a more realistic cluster is [here](https://github.com/otoolep/hraftd/blob/master/CLUSTERING.md)._

Let's bring up 2 more nodes, so we have a 3-node cluster. That way we can tolerate the failure of 1 node:
```bash
$GOPATH/bin/hraftd -id node1 -haddr :11001 -raddr :12001 -join :11000 ~/node1
$GOPATH/bin/hraftd -id node2 -haddr :11002 -raddr :12002 -join :11000 ~/node2
```
_This example shows each hraftd node running on the same host, so each node must listen on different ports. This would not be necessary if each node ran on a different host._

This tells each new node to join the existing node. Once joined, each node now knows about the key:
```bash
curl -XGET localhost:11000/key/user1
curl -XGET localhost:11001/key/user1
curl -XGET localhost:11002/key/user1
```

Furthermore you can add a second key:
```bash
curl -XPOST localhost:11000/key -d '{"user2": "robin"}'
```

Confirm that the new key has been set like so:
```bash
curl -XGET localhost:11000/key/user2
curl -XGET localhost:11001/key/user2
curl -XGET localhost:11002/key/user2
```

#### Stale reads
Because any node will answer a GET request, and nodes may "fall behind" updates, stale reads are possible. Again, hraftd is a simple program, for the purpose of demonstrating a distributed key-value store. If you are particularly interested in learning more about issue, you should check out [rqlite](https://github.com/rqlite/rqlite). rqlite allows the client to control [read consistency](https://github.com/rqlite/rqlite/blob/master/DOC/CONSISTENCY.md), allowing the client to trade off read-responsiveness and correctness.

Read-consistency support could be ported to hraftd if necessary.

### Tolerating failure
Kill the leader process and watch one of the other nodes be elected leader. The keys are still available for query on the other nodes, and you can set keys on the new leader. Furthermore, when the first node is restarted, it will rejoin the cluster and learn about any updates that occurred while it was down.

A 3-node cluster can tolerate the failure of a single node, but a 5-node cluster can tolerate the failure of two nodes. But 5-node clusters require that the leader contact a larger number of nodes before any change e.g. setting a key's value, can be considered committed.

### Leader-forwarding
Automatically forwarding requests to set keys to the current leader is not implemented. The client must always send requests to change a key to the leader or an error will be returned.

## Production use of Raft
For a production-grade example of using Hashicorp's Raft implementation, to replicate a SQLite database, check out [rqlite](https://github.com/rqlite/rqlite).
