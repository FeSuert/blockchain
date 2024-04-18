#!/bin/bash
# docker compose build
docker compose up &
sleep 10
docker compose kill node3
docker compose exec --detach --no-TTY tester1 python3 tests.py phase2
docker compose exec --detach --no-TTY tester2 python3 tests.py phase2
sleep 10
docker compose restart node1
docker compose restart node2
docker compose restart node4
docker compose restart node5
docker compose exec --detach --no-TTY tester1 python3 tests.py phase3
sleep 5
docker compose down
