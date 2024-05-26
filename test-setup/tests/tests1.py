#!/usr/bin/env python3
# Tests if all nodes receive all messages.
import json
import random
import requests
import sys
import time
import logging
logging.basicConfig(filename='results.log', format='%(asctime)s %(message)s', datefmt='%Y/%m/%d %I:%M:%S', level=logging.INFO)

n = 5
rpc_port = 7654


def rpc(url, method, arg):
    headers = {'content-type': 'application/json'}
    payload = {
        "method": method,
        "params": arg,
        "jsonrpc": "2.0",
        "id": 0,
    }
    print(url)
    print(json.dumps(payload))
    print(headers)
    response = requests.post(url, data=json.dumps(payload), headers=headers)
    print("Response Text:", response.text)
    return response.json()  # Ensure response is valid JSON



def send_messages1():
    for _ in range(15):
        url = f"http://node{random.choice(range(1, n+1))}:{rpc_port}/rpc"
        rpc(url, "Node.Broadcast", [f"user {random.choice(range(1, 15))} says hello"])


def send_messages2():
    for _ in range(15):
        not3 = list(set(range(1, n+1))-{3})
        url = f"http://node{random.choice(not3)}:{rpc_port}/rpc"
        rpc(url, "Node.Broadcast", [f"user {random.choice(range(1, 15))} says hello"])


def check_outputs():
    print("")
    for i in range(1, n+1):
        if i == 3:
            continue
        url = f"http://node{i}:{rpc_port}/rpc"
        output = rpc(url, "Node.QueryAll", [])['result']
        logging.info('\n'.join(
            [f'node{i}:']+[f'    {message}' for message in output]
        ))


if __name__ == "__main__":
    time.sleep(5)

    if sys.argv[1] == "phase1":
        send_messages1()
        while True:
            time.sleep(5)

    elif sys.argv[1] == "phase2":
        send_messages2()
        while True:
            time.sleep(5)

    elif sys.argv[1] == "phase3":
        check_outputs()
