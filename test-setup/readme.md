# Project 1 Test Setup
1. In `node`, students must provide a Dockerfile that creates a container in which `/app/node` is the node application.
  The `/app/data` directory in the container persists across contiainer restarts (it is mapped to, eg, `node1_data` on the host machine).
  - An example Dockerfile is provided where a Go program is compiled.
2. To test, run `start_tests.sh`.
   The output can be found after around 30 seconds in `tests/results.log`.

## What the tests do
- The compose file spins up `n=5` containers all running the node application but with different config files (see `configs`).
- The config files specify which node should initially connect to which other node (by using their hardcoded hostnames, which are automatically resolved to the right IP addresses in the compose environment).

The topology is:
```
node1---node2
    \   /
    node3
      |
    node4
      |
    node5
```

- The compose file launches two more containers which concurrently send broadcast RPCs to randomly chosen nodes.
- After a few seconds, `node3`--which is at the center of the topology--is killed.
  The tester containers then issue more broadcasts to the other nodes.
- Then, nodes 1, 2, 4 and 5 are restarted and we log which one received which messages.
  Ideally, everyone (except for node 3) should have received every message because even when `node3` fails, the gossip layer should deliver all messages.
  
