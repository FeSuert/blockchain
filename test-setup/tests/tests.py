#!/usr/bin/env python3
# Tests if all nodes would be able to process all messages *in the same order*.
import json
import requests
import random
import sys
import time
import logging
logging.basicConfig(filename='results.log', format='%(asctime)s %(message)s', datefmt='%Y/%m/%d %I:%M:%S', level=logging.INFO)

n = 5
rpc_port = 7654


def rpc(url, method, arg):
    headers = {'content-type': 'application/json'}
    payload = {
        'method': method,
        'params': arg,
        'jsonrpc': '2.0',
        'id': 0,
    }
    response = requests.post(url, data=json.dumps(payload), headers=headers)
    return response.json() 


def send_messages():
    people = [ 'Alice', 'Bob', 'Charlie', 'David' ]
    for _ in range(20):
        url = f'http://node{random.choice(range(1, n+1))}:{rpc_port}/rpc'
        payer = random.choice(people)
        message = f'[nonce: {random.randint(0, 1000000)}] {payer} wants to pay {random.choice(list(set(people)-{payer}))}'
        # The pseudorandom nonce should make messages unique.
        # For simplicity, the nonce has no meaning here (unlike, eg, mainnet Ethereum).
        rpc(url, 'Node.Broadcast', [message])


def check_outputs():
    for i in range(1, n+1):
        url = f'http://node{i}:{rpc_port}/rpc'
        output = rpc(url, 'Node.QueryAll', [])['result']
        logging.info('\n'.join([f'node{i}']+[f'    {block}' for block in output]))


if __name__ == '__main__':
    time.sleep(5)
    send_messages()
    time.sleep(10)
    if sys.argv[1] == 'tester1':
        check_outputs()
    else:
        # make sure tester1 exits first
        time.sleep(60)