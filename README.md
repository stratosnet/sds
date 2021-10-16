# SDS
<img src="https://github.com/stratosnet/sds/blob/main/_stratos-logo-hb-bz.svg" height="100" alt="Stratos Logo"/>  

The Stratos Decentralized Storage (SDS) network is a scalable, reliable, self-balancing elastic acceleration network driven by data traffic. It accesses data efficiently and safely. The user has the full flexibility to store any data regardless of the size and type.

SDS is composed of many resource nodes (also called PP nodes) that store data, and a few meta nodes (also called indexing or SP nodes) that coordinate everything.
The current repository contains the code for resource nodes only. For more information about meta nodes, we will open source it once it's ready.

## Building a Resource Node From Source
```bash
git clone https://github.com/stratosnet/sds.git
cd sds
git checkout v0.3.0
make build
```
Then you will find the executable binary `ppd` under `./target`
### Installing the Binary
The binary can be installed to the default $GOPATH/bin folder by running:  
```bash
make install
```
The binary should then be runnable from any folder if you have setup `go env` properly
## How to Run and Create Your Own Resource Node
### Creating a Root Directory
To start a resource node, you need to be in a directory dedicated to your resource node. Create a new directory, or go to the root directory of your existing node.
```bash
# create a new folder 
mkdir rsnode
cd rsnode
```
### Configuring the Node
Next, you need to generate the configuration file for your node. The binary will help you create the necessary key pairs.
```bash
ppd config
# then follow the instructions
```
This should create a configuration file at `configs/config.yaml`.  
You will need to edit a few lines in that file to specify the blockchain you want to connect to.  
To connect to the Stratos chain testnet, make the following changes:
```yaml
StratosChainUrl: https://rest-test.thestratos.org:443
# you can also configure it to your own `stchaincli rest-server` if you already run one with your stchaind node
```

and then the indexing node list:
```yaml
SPList: 
- P2PAddress: ""
  P2PPublicKey: ""
  NetworkAddress: 54.212.112.93:8888 
```
You also need to change the `ChainId` to the value visible [on this page](https://big-dipper-test.thestratos.org/) right next to the search bar at the top of the page.  
```yaml
ChainId: stratos-testnet-3 
```
Finally, make sure to set the `NetworkAddress` to your public IP address and port. Please note, is is not the SPList's indexing node NetworkAddress
```yaml
# if your node is behind a router, you probably need to configure port forwarding on the router
Port: :18081
NetworkAddress: <your node external ip> 
```
### Acquiring STOS Tokens
Before you can do anything with your resource node, you will need to acquire some STOS tokens.  
You can get some by using the faucet API:
````bash
curl -X POST https://faucet-test.thestratos.org/faucet/WALLET_ADDRESS
````
Just put your wallet address in the command above and you should be good to go.

### Starting the Node
Once your configuration file is set up properly, and you own tokens, you can finally start your node.

To start the node with a terminal for inputting commands: 
```bash
ppd terminal
```
OR
to start the node as a daemon without interactivity:
```bash
ppd start
```
please note if you didn't finish the `ppd terminal` steps until `startmining` before run `ppd start`, your node WON'T participate in the traffic mining

### Registering to a Meta node
Your node will need to register to a meta node before doing anything else.  
When you have a resource node running with a terminal, input one of the two following identical commands:
```bash
rp
#or
registerpeer
```

### Activating the resource node by staking
Now you need to activate your node within the blockchain.  
Use this command in the terminal:
```bash
activate stakingAmount feeAmount gasAmount
```
The `stakingAmount` is the amount of uSTOS you want to stake. A basic amount would be 1000000000.  
The `feeAmount` is the amount of uSTOS to pay as a fee for the activation transaction. 10000 would work. it will use default number if not provide  
The `gasAmount` is the amount of gas to use for the transaction. 1000000 would be a safe number. it will use default number if not provide

### Starting to Mine
Use this command in the terminal to start mining. Your node will start acting as a resource node and receiving traffic.
```bash
startmining
```
and now you can exit the terminal by typing `exit`, and run `ppd start` to run the node in background

## What to Do With a Running Resource Node?
Once you have an active resource node running with a terminal.(you have run `ppd terminal`)

here are a few of the things you can do.

### Purchase Ozone
Ozone is the unit of traffic used by SDS. Operations involving network traffic require ozone to be executed.  
You can purchase ozone with the following command:
```bash
prepay purchaseAmount feeAmount gasAmount
```
`purchaseAmount` is the amount of uSTOS you want to spend to purchase ozone.  
The other two parameters are the same as above.

### Upload a File
```bash
put FILE_PATH
```
`FILE_PATH` is the location of the file to upload, starting from your resource node folder.
### List Your Uploaded Files
```bash
list
```

### Download a File
```bash
get sdm://WALLET_ADDRESS/FILE_HASH
```
Every file uploaded to SDS is attributed a unique file hash. You can view the file hash for each of your files when your list your uploaded files.

### Delete a File
```bash
delete FILE_HASH
```

### Share a File
```bash
sharefile FILE_HASH EXPIRY_TIME PRIVATE
```
`EXPIRY_TIME` is the unix timestamp where the file share expires. Put `0` for unlimited time.  
`PRIVATE` is whether the file share should be protected by a password. Put `0` for no password, and `1` for a password.

### List All Shared Files
```bash
allshare
```

### Download a Shared File
```bash
getsharefile SHARE_LINK PASSWORD
```
Leave the `PASSWORD` blank if it's a public shared file.

### Cancel File Share
```bash
cancelshare SHARE_ID
```

### View Resource Utilization
Type `monitor` to show the resource utilization monitor, and `stopmonitor`to hide it.


##
# Contribution

Thank you for considering to help out with the source code! We welcome contributions
from anyone on the internet, and are grateful for even the smallest of fixes!

If you'd like to contribute to SDS(Stratos Decentralized Storage), please fork, fix, commit and send a pull request
for the maintainers to review and merge into the main code base.

Please make sure your contributions adhere to our coding guidelines:

* Code must adhere to the official Go [formatting](https://golang.org/doc/effective_go.html#formatting)
  guidelines (i.e. uses [gofmt](https://golang.org/cmd/gofmt/)).
* Code must be documented adhering to the official Go [commentary](https://golang.org/doc/effective_go.html#commentary)
  guidelines.
* Pull requests need to be based on and opened against the `dev` branch, PR name should follow `conventional commits`.
* Commit messages should be prefixed with the package(s) they modify.
    * E.g. "pp: make trace configs optional"

--- ---

# License

Copyright 2021 Stratos

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the [License](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.