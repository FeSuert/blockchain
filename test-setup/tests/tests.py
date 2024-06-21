#!/usr/bin/env python3
# Tests if we can deploy and use a simple smart contract.
import os
import json
import math
import requests
import random
import sys
import time
import logging
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey
logging.basicConfig(filename='results.log', format='%(asctime)s %(message)s', datefmt='%Y/%m/%d %I:%M:%S', level=logging.INFO)

n = 5
rpc_port = 7654
alice = {'address': '0x261a32aaccd1359ad0e27e366aafcd3d7e0f17b2',
         'public_key': '0x6638fc42a8d310bd5067add39837ea149295cbed56fe862c0e2ed25c21c4d16e',
         'private_key': '0x6a7f7710372da6e30e4907767a409d914b3b3110d7deff04b113e4d26cbbac3b',}
bob = {'address': '0x74137ab12a1876fedb216133186f3b6367bd7ece',
       'public_key': '0x5381c34c209a6041df71bb763e601610ff268b2335964c96ed8743d43654cbaf',
       'private_key': '0xc470c1cc89307e5790ac1b8a7f2fd11cb05d97a361353e465dba5797d8a1fdf1',}
charlie = {'address': '0xaf2470495aa63898bf9cbb0096c866b306d6179e',
           'public_key': '0xce0f148298ea9640e6d717deb43fa406ee99bff5442df28740a453a58b6bc55d',
           'private_key': '0x377ff0de7afdedb6e7b8ac8d01fa4335f39dabf80fc77fd4e05de3e0e50d2406',}
david = {'address': '0x0f39b60ed96569f44f67348faa8744052382dd85',
         'public_key': '0x5f84a5c7668982de00e4f2088ee1540744f352d106fc7773927f0fd2d701e1c6',
         'private_key': '0xf0dffc78bb58adc98df1c9a2cc84f8f02f451f4b88a820e1fd9e68fad11f7c2d',}
erc20token = '6101606040523480156200001257600080fd5b506040518060400160405280600c81526020017f4578616d706c65546f6b656e0000000000000000000000000000000000000000815250806040518060400160405280600181526020017f31000000000000000000000000000000000000000000000000000000000000008152506040518060400160405280600c81526020017f4578616d706c65546f6b656e00000000000000000000000000000000000000008152506040518060400160405280600781526020017f4558414d504c45000000000000000000000000000000000000000000000000008152508160039081620000fd91906200065d565b5080600490816200010f91906200065d565b5050506200013262000126620001e960201b60201c565b620001f160201b60201c565b62000148600683620002b760201b90919060201c565b610120818152505062000166600782620002b760201b90919060201c565b6101408181525050818051906020012060e08181525050808051906020012061010081815250504660a08181525050620001a56200030f60201b60201c565b608081815250503073ffffffffffffffffffffffffffffffffffffffff1660c08173ffffffffffffffffffffffffffffffffffffffff168152505050505062000967565b600033905090565b6000600560009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905081600560006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055508173ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a35050565b6000602083511015620002dd57620002d5836200036c60201b60201c565b905062000309565b82620002ef83620003d960201b60201c565b60000190816200030091906200065d565b5060ff60001b90505b92915050565b60007f8b73c3c69bb8fe3d512ecc4cf759cc79239f7b179b0ffacaa9a75d522b39400f60e05161010051463060405160200162000351959493929190620007b5565b60405160208183030381529060405280519060200120905090565b600080829050601f81511115620003bc57826040517f305a27a9000000000000000000000000000000000000000000000000000000008152600401620003b39190620008a1565b60405180910390fd5b805181620003ca90620008f7565b60001c1760001b915050919050565b6000819050919050565b600081519050919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b600060028204905060018216806200046557607f821691505b6020821081036200047b576200047a6200041d565b5b50919050565b60008190508160005260206000209050919050565b60006020601f8301049050919050565b600082821b905092915050565b600060088302620004e57fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82620004a6565b620004f18683620004a6565b95508019841693508086168417925050509392505050565b6000819050919050565b6000819050919050565b60006200053e62000538620005328462000509565b62000513565b62000509565b9050919050565b6000819050919050565b6200055a836200051d565b62000572620005698262000545565b848454620004b3565b825550505050565b600090565b620005896200057a565b620005968184846200054f565b505050565b5b81811015620005be57620005b26000826200057f565b6001810190506200059c565b5050565b601f8211156200060d57620005d78162000481565b620005e28462000496565b81016020851015620005f2578190505b6200060a620006018562000496565b8301826200059b565b50505b505050565b600082821c905092915050565b6000620006326000198460080262000612565b1980831691505092915050565b60006200064d83836200061f565b9150826002028217905092915050565b6200066882620003e3565b67ffffffffffffffff811115620006845762000683620003ee565b5b6200069082546200044c565b6200069d828285620005c2565b600060209050601f831160018114620006d55760008415620006c0578287015190505b620006cc85826200063f565b8655506200073c565b601f198416620006e58662000481565b60005b828110156200070f57848901518255600182019150602085019450602081019050620006e8565b868310156200072f57848901516200072b601f8916826200061f565b8355505b6001600288020188555050505b505050505050565b6000819050919050565b620007598162000744565b82525050565b6200076a8162000509565b82525050565b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b60006200079d8262000770565b9050919050565b620007af8162000790565b82525050565b600060a082019050620007cc60008301886200074e565b620007db60208301876200074e565b620007ea60408301866200074e565b620007f960608301856200075f565b620008086080830184620007a4565b9695505050505050565b600082825260208201905092915050565b60005b838110156200084357808201518184015260208101905062000826565b60008484015250505050565b6000601f19601f8301169050919050565b60006200086d82620003e3565b62000879818562000812565b93506200088b81856020860162000823565b62000896816200084f565b840191505092915050565b60006020820190508181036000830152620008bd818462000860565b905092915050565b600081519050919050565b6000819050602082019050919050565b6000620008ee825162000744565b80915050919050565b60006200090482620008c5565b826200091084620008d0565b90506200091d81620008e0565b9250602082101562000960576200095b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff83602003600802620004a6565b831692505b5050919050565b60805160a05160c05160e051610100516101205161014051612792620009c26000396000610622015260006105ee015260006114540152600061143301526000610f5601526000610fac01526000610fd501526127926000f3fe608060405234801561001057600080fd5b50600436106101215760003560e01c8063715018a6116100ad578063a457c2d711610071578063a457c2d714610314578063a9059cbb14610344578063d505accf14610374578063dd62ed3e14610390578063f2fde38b146103c057610121565b8063715018a61461027a5780637ecebe001461028457806384b0196e146102b45780638da5cb5b146102d857806395d89b41146102f657610121565b8063313ce567116100f4578063313ce567146101c25780633644e515146101e057806339509351146101fe57806340c10f191461022e57806370a082311461024a57610121565b806306fdde0314610126578063095ea7b31461014457806318160ddd1461017457806323b872dd14610192575b600080fd5b61012e6103dc565b60405161013b9190611897565b60405180910390f35b61015e60048036038101906101599190611952565b61046e565b60405161016b91906119ad565b60405180910390f35b61017c610491565b60405161018991906119d7565b60405180910390f35b6101ac60048036038101906101a791906119f2565b61049b565b6040516101b991906119ad565b60405180910390f35b6101ca6104ca565b6040516101d79190611a61565b60405180910390f35b6101e86104d3565b6040516101f59190611a95565b60405180910390f35b61021860048036038101906102139190611952565b6104e2565b60405161022591906119ad565b60405180910390f35b61024860048036038101906102439190611952565b610519565b005b610264600480360381019061025f9190611ab0565b61052f565b60405161027191906119d7565b60405180910390f35b610282610577565b005b61029e60048036038101906102999190611ab0565b61058b565b6040516102ab91906119d7565b60405180910390f35b6102bc6105db565b6040516102cf9796959493929190611be5565b60405180910390f35b6102e06106dd565b6040516102ed9190611c69565b60405180910390f35b6102fe610707565b60405161030b9190611897565b60405180910390f35b61032e60048036038101906103299190611952565b610799565b60405161033b91906119ad565b60405180910390f35b61035e60048036038101906103599190611952565b610810565b60405161036b91906119ad565b60405180910390f35b61038e60048036038101906103899190611cdc565b610833565b005b6103aa60048036038101906103a59190611d7e565b610975565b6040516103b791906119d7565b60405180910390f35b6103da60048036038101906103d59190611ab0565b6109fc565b005b6060600380546103eb90611ded565b80601f016020809104026020016040519081016040528092919081815260200182805461041790611ded565b80156104645780601f1061043957610100808354040283529160200191610464565b820191906000526020600020905b81548152906001019060200180831161044757829003601f168201915b5050505050905090565b600080610479610a7f565b9050610486818585610a87565b600191505092915050565b6000600254905090565b6000806104a6610a7f565b90506104b3858285610c50565b6104be858585610cdc565b60019150509392505050565b60006012905090565b60006104dd610f52565b905090565b6000806104ed610a7f565b905061050e8185856104ff8589610975565b6105099190611e4d565b610a87565b600191505092915050565b610521611009565b61052b8282611087565b5050565b60008060008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020549050919050565b61057f611009565b61058960006111dd565b565b60006105d4600860008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000206112a3565b9050919050565b60006060806000806000606061061b60067f00000000000000000000000000000000000000000000000000000000000000006112b190919063ffffffff16565b61064f60077f00000000000000000000000000000000000000000000000000000000000000006112b190919063ffffffff16565b46306000801b600067ffffffffffffffff8111156106705761066f611e81565b5b60405190808252806020026020018201604052801561069e5781602001602082028036833780820191505090505b507f0f00000000000000000000000000000000000000000000000000000000000000959493929190965096509650965096509650965090919293949596565b6000600560009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905090565b60606004805461071690611ded565b80601f016020809104026020016040519081016040528092919081815260200182805461074290611ded565b801561078f5780601f106107645761010080835404028352916020019161078f565b820191906000526020600020905b81548152906001019060200180831161077257829003601f168201915b5050505050905090565b6000806107a4610a7f565b905060006107b28286610975565b9050838110156107f7576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016107ee90611f22565b60405180910390fd5b6108048286868403610a87565b60019250505092915050565b60008061081b610a7f565b9050610828818585610cdc565b600191505092915050565b83421115610876576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161086d90611f8e565b60405180910390fd5b60007f6e71edae12b1b97f4d1f60370fef10105fa2faae0126114a169c64845d6126c98888886108a58c611361565b896040516020016108bb96959493929190611fae565b60405160208183030381529060405280519060200120905060006108de826113bf565b905060006108ee828787876113d9565b90508973ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff161461095e576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016109559061205b565b60405180910390fd5b6109698a8a8a610a87565b50505050505050505050565b6000600160008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054905092915050565b610a04611009565b600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff1603610a73576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610a6a906120ed565b60405180910390fd5b610a7c816111dd565b50565b600033905090565b600073ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff1603610af6576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610aed9061217f565b60405180910390fd5b600073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff1603610b65576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610b5c90612211565b60405180910390fd5b80600160008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508173ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff167f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b92583604051610c4391906119d7565b60405180910390a3505050565b6000610c5c8484610975565b90507fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8114610cd65781811015610cc8576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610cbf9061227d565b60405180910390fd5b610cd58484848403610a87565b5b50505050565b600073ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff1603610d4b576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610d429061230f565b60405180910390fd5b600073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff1603610dba576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610db1906123a1565b60405180910390fd5b610dc5838383611404565b60008060008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054905081811015610e4b576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610e4290612433565b60405180910390fd5b8181036000808673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002081905550816000808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020600082825401925050819055508273ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef84604051610f3991906119d7565b60405180910390a3610f4c848484611409565b50505050565b60007f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff163073ffffffffffffffffffffffffffffffffffffffff16148015610fce57507f000000000000000000000000000000000000000000000000000000000000000046145b15610ffb577f00000000000000000000000000000000000000000000000000000000000000009050611006565b61100361140e565b90505b90565b611011610a7f565b73ffffffffffffffffffffffffffffffffffffffff1661102f6106dd565b73ffffffffffffffffffffffffffffffffffffffff1614611085576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161107c9061249f565b60405180910390fd5b565b600073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff16036110f6576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016110ed9061250b565b60405180910390fd5b61110260008383611404565b80600260008282546111149190611e4d565b92505081905550806000808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020600082825401925050819055508173ffffffffffffffffffffffffffffffffffffffff16600073ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef836040516111c591906119d7565b60405180910390a36111d960008383611409565b5050565b6000600560009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905081600560006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055508173ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a35050565b600081600001549050919050565b606060ff60001b83146112ce576112c7836114a4565b905061135b565b8180546112da90611ded565b80601f016020809104026020016040519081016040528092919081815260200182805461130690611ded565b80156113535780601f1061132857610100808354040283529160200191611353565b820191906000526020600020905b81548152906001019060200180831161133657829003601f168201915b505050505090505b92915050565b600080600860008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002090506113ae816112a3565b91506113b981611518565b50919050565b60006113d26113cc610f52565b8361152e565b9050919050565b60008060006113ea8787878761156f565b915091506113f781611651565b8192505050949350505050565b505050565b505050565b60007f8b73c3c69bb8fe3d512ecc4cf759cc79239f7b179b0ffacaa9a75d522b39400f7f00000000000000000000000000000000000000000000000000000000000000007f0000000000000000000000000000000000000000000000000000000000000000463060405160200161148995949392919061252b565b60405160208183030381529060405280519060200120905090565b606060006114b1836117b7565b90506000602067ffffffffffffffff8111156114d0576114cf611e81565b5b6040519080825280601f01601f1916602001820160405280156115025781602001600182028036833780820191505090505b5090508181528360208201528092505050919050565b6001816000016000828254019250508190555050565b60006040517f190100000000000000000000000000000000000000000000000000000000000081528360028201528260228201526042812091505092915050565b6000807f7fffffffffffffffffffffffffffffff5d576e7357a4501ddfe92f46681b20a08360001c11156115aa576000600391509150611648565b6000600187878787604051600081526020016040526040516115cf949392919061257e565b6020604051602081039080840390855afa1580156115f1573d6000803e3d6000fd5b505050602060405103519050600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff160361163f57600060019250925050611648565b80600092509250505b94509492505050565b60006004811115611665576116646125c3565b5b816004811115611678576116776125c3565b5b03156117b45760016004811115611692576116916125c3565b5b8160048111156116a5576116a46125c3565b5b036116e5576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016116dc9061263e565b60405180910390fd5b600260048111156116f9576116f86125c3565b5b81600481111561170c5761170b6125c3565b5b0361174c576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401611743906126aa565b60405180910390fd5b600360048111156117605761175f6125c3565b5b816004811115611773576117726125c3565b5b036117b3576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016117aa9061273c565b60405180910390fd5b5b50565b60008060ff8360001c169050601f8111156117fe576040517fb3512b0c00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b80915050919050565b600081519050919050565b600082825260208201905092915050565b60005b83811015611841578082015181840152602081019050611826565b60008484015250505050565b6000601f19601f8301169050919050565b600061186982611807565b6118738185611812565b9350611883818560208601611823565b61188c8161184d565b840191505092915050565b600060208201905081810360008301526118b1818461185e565b905092915050565b600080fd5b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b60006118e9826118be565b9050919050565b6118f9816118de565b811461190457600080fd5b50565b600081359050611916816118f0565b92915050565b6000819050919050565b61192f8161191c565b811461193a57600080fd5b50565b60008135905061194c81611926565b92915050565b60008060408385031215611969576119686118b9565b5b600061197785828601611907565b92505060206119888582860161193d565b9150509250929050565b60008115159050919050565b6119a781611992565b82525050565b60006020820190506119c2600083018461199e565b92915050565b6119d18161191c565b82525050565b60006020820190506119ec60008301846119c8565b92915050565b600080600060608486031215611a0b57611a0a6118b9565b5b6000611a1986828701611907565b9350506020611a2a86828701611907565b9250506040611a3b8682870161193d565b9150509250925092565b600060ff82169050919050565b611a5b81611a45565b82525050565b6000602082019050611a766000830184611a52565b92915050565b6000819050919050565b611a8f81611a7c565b82525050565b6000602082019050611aaa6000830184611a86565b92915050565b600060208284031215611ac657611ac56118b9565b5b6000611ad484828501611907565b91505092915050565b60007fff0000000000000000000000000000000000000000000000000000000000000082169050919050565b611b1281611add565b82525050565b611b21816118de565b82525050565b600081519050919050565b600082825260208201905092915050565b6000819050602082019050919050565b611b5c8161191c565b82525050565b6000611b6e8383611b53565b60208301905092915050565b6000602082019050919050565b6000611b9282611b27565b611b9c8185611b32565b9350611ba783611b43565b8060005b83811015611bd8578151611bbf8882611b62565b9750611bca83611b7a565b925050600181019050611bab565b5085935050505092915050565b600060e082019050611bfa600083018a611b09565b8181036020830152611c0c818961185e565b90508181036040830152611c20818861185e565b9050611c2f60608301876119c8565b611c3c6080830186611b18565b611c4960a0830185611a86565b81810360c0830152611c5b8184611b87565b905098975050505050505050565b6000602082019050611c7e6000830184611b18565b92915050565b611c8d81611a45565b8114611c9857600080fd5b50565b600081359050611caa81611c84565b92915050565b611cb981611a7c565b8114611cc457600080fd5b50565b600081359050611cd681611cb0565b92915050565b600080600080600080600060e0888a031215611cfb57611cfa6118b9565b5b6000611d098a828b01611907565b9750506020611d1a8a828b01611907565b9650506040611d2b8a828b0161193d565b9550506060611d3c8a828b0161193d565b9450506080611d4d8a828b01611c9b565b93505060a0611d5e8a828b01611cc7565b92505060c0611d6f8a828b01611cc7565b91505092959891949750929550565b60008060408385031215611d9557611d946118b9565b5b6000611da385828601611907565b9250506020611db485828601611907565b9150509250929050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b60006002820490506001821680611e0557607f821691505b602082108103611e1857611e17611dbe565b5b50919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b6000611e588261191c565b9150611e638361191c565b9250828201905080821115611e7b57611e7a611e1e565b5b92915050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b7f45524332303a2064656372656173656420616c6c6f77616e63652062656c6f7760008201527f207a65726f000000000000000000000000000000000000000000000000000000602082015250565b6000611f0c602583611812565b9150611f1782611eb0565b604082019050919050565b60006020820190508181036000830152611f3b81611eff565b9050919050565b7f45524332305065726d69743a206578706972656420646561646c696e65000000600082015250565b6000611f78601d83611812565b9150611f8382611f42565b602082019050919050565b60006020820190508181036000830152611fa781611f6b565b9050919050565b600060c082019050611fc36000830189611a86565b611fd06020830188611b18565b611fdd6040830187611b18565b611fea60608301866119c8565b611ff760808301856119c8565b61200460a08301846119c8565b979650505050505050565b7f45524332305065726d69743a20696e76616c6964207369676e61747572650000600082015250565b6000612045601e83611812565b91506120508261200f565b602082019050919050565b6000602082019050818103600083015261207481612038565b9050919050565b7f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160008201527f6464726573730000000000000000000000000000000000000000000000000000602082015250565b60006120d7602683611812565b91506120e28261207b565b604082019050919050565b60006020820190508181036000830152612106816120ca565b9050919050565b7f45524332303a20617070726f76652066726f6d20746865207a65726f2061646460008201527f7265737300000000000000000000000000000000000000000000000000000000602082015250565b6000612169602483611812565b91506121748261210d565b604082019050919050565b600060208201905081810360008301526121988161215c565b9050919050565b7f45524332303a20617070726f766520746f20746865207a65726f20616464726560008201527f7373000000000000000000000000000000000000000000000000000000000000602082015250565b60006121fb602283611812565b91506122068261219f565b604082019050919050565b6000602082019050818103600083015261222a816121ee565b9050919050565b7f45524332303a20696e73756666696369656e7420616c6c6f77616e6365000000600082015250565b6000612267601d83611812565b915061227282612231565b602082019050919050565b600060208201905081810360008301526122968161225a565b9050919050565b7f45524332303a207472616e736665722066726f6d20746865207a65726f20616460008201527f6472657373000000000000000000000000000000000000000000000000000000602082015250565b60006122f9602583611812565b91506123048261229d565b604082019050919050565b60006020820190508181036000830152612328816122ec565b9050919050565b7f45524332303a207472616e7366657220746f20746865207a65726f206164647260008201527f6573730000000000000000000000000000000000000000000000000000000000602082015250565b600061238b602383611812565b91506123968261232f565b604082019050919050565b600060208201905081810360008301526123ba8161237e565b9050919050565b7f45524332303a207472616e7366657220616d6f756e742065786365656473206260008201527f616c616e63650000000000000000000000000000000000000000000000000000602082015250565b600061241d602683611812565b9150612428826123c1565b604082019050919050565b6000602082019050818103600083015261244c81612410565b9050919050565b7f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e6572600082015250565b6000612489602083611812565b915061249482612453565b602082019050919050565b600060208201905081810360008301526124b88161247c565b9050919050565b7f45524332303a206d696e7420746f20746865207a65726f206164647265737300600082015250565b60006124f5601f83611812565b9150612500826124bf565b602082019050919050565b60006020820190508181036000830152612524816124e8565b9050919050565b600060a0820190506125406000830188611a86565b61254d6020830187611a86565b61255a6040830186611a86565b61256760608301856119c8565b6125746080830184611b18565b9695505050505050565b60006080820190506125936000830187611a86565b6125a06020830186611a52565b6125ad6040830185611a86565b6125ba6060830184611a86565b95945050505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602160045260246000fd5b7f45434453413a20696e76616c6964207369676e61747572650000000000000000600082015250565b6000612628601883611812565b9150612633826125f2565b602082019050919050565b600060208201905081810360008301526126578161261b565b9050919050565b7f45434453413a20696e76616c6964207369676e6174757265206c656e67746800600082015250565b6000612694601f83611812565b915061269f8261265e565b602082019050919050565b600060208201905081810360008301526126c381612687565b9050919050565b7f45434453413a20696e76616c6964207369676e6174757265202773272076616c60008201527f7565000000000000000000000000000000000000000000000000000000000000602082015250565b6000612726602283611812565b9150612731826126ca565b604082019050919050565b6000602082019050818103600083015261275581612719565b905091905056fea264697066735822122009f6772b510fd26a4fdafab69286db47390abcfb257d11a8d1756106da0a251464736f6c63430008130033'
nonce = 0

def int2bytes(i):
    return i.to_bytes(math.ceil(i.bit_length() / 8), byteorder='big', signed=False)


def tx_bytes(tx):
    if 'to' in tx:
        return (bytes.fromhex(tx['sender'][2:])+
                int2bytes(tx['nonce'])+
                bytes.fromhex(tx['to'][2:])+
                int2bytes(tx['amount'])+
                bytes.fromhex(tx['input']))
    else:
        return (bytes.fromhex(tx['sender'][2:])+
                int2bytes(tx['nonce'])+
                #int2bytes(tx['amount'])+
                bytes.fromhex(tx['input']))


def hash_and_sign(tx, signer):
    sk = Ed25519PrivateKey.from_private_bytes(bytes.fromhex(signer['private_key'][2:]))
    data = tx_bytes(tx)
    sha256 = hashes.Hash(hashes.SHA256())
    sha256.update(data)
    tx_hash = sha256.finalize()
    tx['signature'] = '0x'+sk.sign(tx_hash).hex()
    tx['public_key'] = signer['public_key']
    return '0x'+tx_hash.hex()


def rpc(url, method, arg):
    headers = {'content-type': 'application/json'}
    payload = { 'method': method,
                'params': arg,
                'jsonrpc': '2.0',
                'id': 0, }
    res = requests.post(url, data=json.dumps(payload), headers=headers).json()
    time.sleep(1)
    return res


def deploy_contract():
    global nonce
    tx = {'sender': alice['address'][2:],
          'nonce': nonce,
          #'amount': int(1e6),
          'input': erc20token, }
    nonce = nonce+1
    tx_hash = hash_and_sign(tx, alice)
    assert 'signature' in tx # TODO: remove if ok
    url = f'http://node{random.choice(range(1, n+1))}:{rpc_port}/rpc'
    rpc(url, 'Node.Broadcast', [tx])
    return tx_hash


def get_result(tx_hash):
    url = f'http://node{random.choice(range(1, n+1))}:{rpc_port}/rpc'
    return rpc(url, 'Node.TxResult', [tx_hash])['result']


def mint_tokens(contract_address):
    global nonce
    for payee in [alice, bob, charlie, david]:
        # call mint
        tx = {'sender': alice['address'][2:],
              'nonce': nonce,
              'to': contract_address,
              'amount': int(1e6),
              'input': f'40c10f19{12*"00"}{payee["address"][2:]}{10000:064x}', }
        nonce = nonce+1
        hash_and_sign(tx, alice)
        url = f'http://node{random.choice(range(1, n+1))}:{rpc_port}/rpc'
        rpc(url, 'Node.Broadcast', [tx])


def send_transfers(contract_address):
    global nonce
    if sys.argv[1] == 'tester1':
        payer = alice
    elif sys.argv[1] == 'tester2':
        payer = bob
    elif sys.argv[1] == 'tester3':
        payer = charlie
    others = [person['address']
              for person in [alice, bob, charlie, david]
              if person['address'] != payer['address']]
    for _ in range(20):
        payee_address = random.choice(others)
        tokens = random.choice(range(1, 501))
        # call transfer
        tx = {'sender': payer['address'],
              'nonce': nonce,
              'to': contract_address,
              'amount': int(1e6),
              'input': f'a9059cbb{12*"00"}{payee_address[2:]}{tokens:064x}', }
        nonce = nonce+1
        hash_and_sign(tx, payer)
        if random.random() > 0.85:
            # set some parts to 0 to make sigantures invalid
            t = list(tx['signature'])
            t[10] = '0'
            t[15] = '0'
            t[20] = '0'
            tx['signature'] = ''.join(t)
        url = f'http://node{random.choice(range(1, n+1))}:{rpc_port}/rpc'
        rpc(url, 'Node.Broadcast', [tx])
        # TODO: log which transfers were made etc.


def check_outputs(contract_address):
    for i in range(1, n+1):
        url = f'http://node{i}:{rpc_port}/rpc'
        block = rpc(url, 'Node.HighestBlock', [])['result']
        logging.info('\n'.join([f'node{i}', f'    {block}']))
    for person in [alice, bob, charlie, david]:
        # call balanceOf
        tx = {'sender': alice['address'],
              'to': contract_address,
              'input': f'70a08231{12*"00"}{person["address"][2:]}', }
        url = f'http://node1:{rpc_port}/rpc'
        output = rpc(url, 'Node.QueryAddress', [tx])['result']
        logging.info('\n'.join(['node1', f'    token balance of {person["address"]}: {output}']))


if __name__ == '__main__':
    wait_for_consensus = 30  # timeout can be adjusted

    if sys.argv[1] == 'tester1':
        tx = deploy_contract()
    time.sleep(wait_for_consensus)
    if sys.argv[1] == 'tester1':
        contract = get_result(tx)['0']
        logging.info(f'(TxResult:) created token contract at {contract}')
        with open('contract.txt', 'w') as f:
            f.write(contract)
        mint_tokens(contract)
    time.sleep(wait_for_consensus)

    if sys.argv[1] != 'tester1':
        while True:
            try:
                with open('contract.txt', 'r') as f:
                    contract = f.read()
                break
            except FileNotFoundError:
                time.sleep(1)
    send_transfers(contract)
    time.sleep(wait_for_consensus)

    if sys.argv[1] == 'tester1':
        check_outputs(contract)
        os.remove('contract.txt')