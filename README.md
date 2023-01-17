# SDS
<img src="https://github.com/stratosnet/sds/blob/main/_stratos-logo-hb-bz.svg" height="100" alt="Stratos Logo"/>  

The Stratos Decentralized Storage (SDS) network is a scalable, reliable, self-balancing elastic acceleration network driven by data traffic. It accesses data efficiently and safely. The user has the full flexibility to store any data regardless of the size and type.

SDS is composed of many resource nodes (also called PP nodes) that store data, and a few meta nodes (also called indexing or SP nodes) that coordinate everything.
The current repository contains the code for resource nodes only. For more information about meta nodes, we will open source it once it's ready.

Here, we provide a concise quickstart guide to help set up and run a SDS resource node. For more details of SDS as well as the Tropos-Incentive-Testnet rewards distribution, please refer to [Tropos Incentive Testnet](https://github.com/stratosnet/sds/wiki/Tropos-Incentive-Testnet).


## Building a Resource Node From Source
```bash
git clone https://github.com/stratosnet/sds.git
cd sds
git checkout v0.8.1
make build
```
Then you will find the executable binary `ppd` under `./target`
### Installing the Binary
The binary can be installed to the default $GOPATH/bin folder by running:
```bash
make install
```
The binary should then be runnable from any folder if you have set up `go env` properly

## Creating a SDS Resource Node

### Creating a Root Directory for Your Resource Node
To start a resource node, you need to be in a directory dedicated to your resource node. Create a new directory, or go to the root directory of your existing node.
```bash
# create a new folder 
mkdir rsnode
cd rsnode
```
### Configuring Your Resource Node
Next, you need to configure your resource node. The binary will help you create a configuration file at `configs/config.toml`.
```bash
ppd config -w -p
# following the instructions to generate a new wallet account or recovery an existing wallet account
```
You will need to edit a few lines in the `configs/config.toml` file to configure your resource node.

First, make sure or change the SDS version section in the `configs/config.toml` file as the following.

```toml
[version]
app_ver = 8
min_app_ver = 8
show = 'v0.8.1'
```

To connect to the Stratos-chain Tropos testnet, make the following changes:
```toml
stratos_chain_url = 'https://rest-tropos.thestratos.org:443' 
```

and then the indexing node list:
```toml
[[sp_list]]
p2p_address = ''
p2p_public_key = ''
network_address = '18.223.175.117:8888'
[[sp_list]]
p2p_address = ''
p2p_public_key = ''
network_address = '52.13.28.64:8888'
[[sp_list]]
p2p_address = ''
p2p_public_key = ''
network_address = '46.51.251.196:8888'
[[sp_list]]
p2p_address = ''
p2p_public_key = ''
network_address = '35.73.160.68:8888'
[[sp_list]]
p2p_address = ''
p2p_public_key = ''
network_address = '35.74.33.155:8888'
[[sp_list]]
p2p_address = ''
p2p_public_key = ''
network_address = '18.130.202.53:8888'
[[sp_list]]
p2p_address = ''
p2p_public_key = ''
network_address = '3.9.152.251:8888'
```
You also need to change the `chain_id` to the value visible [`Stratos Explorer`](https://explorer-tropos.thestratos.org/) right next to the search bar at the top of the page. Currently, it is `tropos-4`.
```toml
chain_id = 'tropos-4'
```
Finally, make sure to set the `network_address` to your public IP address and port.

Please note, it is not the meta node network_address in [[sp_list]] section

```toml
# if your node is behind a router, you probably need to configure port forwarding on the router
port = '18081'
network_address = 'your node external ip' 
```
### Acquiring STOS Tokens
Before you can do anything with your resource node, you will need to acquire some STOS tokens.  
You can get some by using the faucet API:
````bash
  curl --header "Content-Type: application/json" --request POST --data '{"denom":"ustos","address":"put_your_wallet_address_here"} ' https://faucet-tropos.thestratos.org/credit
````
Just put your wallet address in the command above, and you should be good to go.

## Starting Your Resource Node
Once your configuration file is set up properly, and you own tokens in your wallet account, you can start your resource node.

to start the node as a daemon in background without interactivity:
```bash
# Make sure we are inside the root directory of the resource node
cd rsnode
# start the resource node
ppd start
```

### Starting Mining
In order to interact with the resource node, you need to open a new COMMAND-LINE TERMINAL, and enter the root directory of the same resource node.
Then, use `ppd terminal` command to start the interaction.
```bash
# Open a new command-line terminal
# Make sure we are inside the root directory of the resource node
cd rsnode
# Interact with resource node through a set of "ppd terminal" subcommands
ppd terminal
```

#### Registering to a Meta Node
Your resource node needs to register to a meta node before doing anything else.  
In the `ppd terminal` command-line terminal, input one of the two following identical commands:
```bash
rp
# or
registerpeer
```

#### Activating the Resource Node by Staking
Now you need to activate your node within the blockchain.  
Use this command in the `ppd terminal` command-line terminal:
```bash
activate stakingAmount feeAmount gasAmount
```
> `stakingAmount` is the amount of token you want to stake. 1stos = 10^9gwei = 10^18wei.
>
>`feeAmount` is the amount of token to pay as a fee for the activation transaction. 10000wei would work. it will use
> default number if not provide
>
>`gasAmount` is the amount of gas to use for the transaction. 1000000wei would be a safe number. it will use default
> number if not provide

Resource node will start to receive tasks from meta nodes and thus gain mining rewards automatically after it has been activated successfully.

## What to Do With a Running Resource Node?
Here are a set of `ppd terminal` subcommands you can try in the `ppd terminal` command-line terminal.

You can find more details about these subcommands at `ppd terminal` [subcommands](https://github.com/stratosnet/sds/wiki/%60ppd-terminal%60--subcommands)

### Check the current status of a resource node

```bash
status
```

### Update stake of an active resource node

```shell
updateStake stakeDelta fee gas isIncrStake 
```

> `stakeDelta` is the absolute amount of difference between the original and the updated stake. It should be a valid
> token, in the unit of `stos`/`gwei`/`wei`.
>
> `isIncrStake` is a flag with `0` for decreasing the original stake and `1` for increasing the original stake.
>
> When a resource node is suspended, use this command to update its state and re-start mining by increasing its stake.

### Purchase Ozone

Ozone is the unit of traffic used by SDS. Operations involving network traffic require ozone to be executed.  
You can purchase ozone with the following command:

```bash
prepay purchaseAmount feeAmount gasAmount
```

> `purchaseAmount` is the amount of token you want to spend to purchase ozone.
>
> The other two parameters are the same as above.

### Query Ozone Balance of Resource Node's Wallet

```bash
getoz WALLET_ADDRESS
```

### Upload a File

```bash
put FILE_PATH
```
> `FILE_PATH` is the location of the file to upload, starting from your resource node folder. It is better to be an absolute path.


### Upload a media file for streaming
Streaming is the continuous transmission of audio or video files(media files) from a server to a client.
In order to upload a streaming file, first you need to install a tool [`ffmpeg`](https://linuxize.com/post/how-to-install-ffmpeg-on-ubuntu-20-04/) for transcoding multimedia files.
```bash
putstream FILE_PATH
```

### List Your Uploaded Files
```bash
list
# or
ls
```

### Download a File
```bash
get sdm://WALLET_ADDRESS/FILE_HASH SAVE_AS
```

Every file uploaded to SDS is attributed a unique file hash. You can view the file hash for each of your files when your list your uploaded files.   
You can use an optional parameter `SAVE_AS` to rename the file after downloading

### Delete a File
```bash
delete FILE_HASH
```

### Share a File
```bash
sharefile FILE_HASH DURATION IS_PRIVATE
```
> `DURATION` is the time period(in seconds) when the file share expires. Put `0` for unlimited time.
>
> `IS_PRIVATE` is whether the file shared should be protected by a password. Put `0` for no password, and `1` for a password.
>
> After this command has been executed successfully, SDS will provide a password to this shared file, like ` SharePassword 3gxw`. Please keep this password for future use.

### List All Shared Files
```bash
allshare
```

### Download a Shared File
```bash
getsharefile SHARE_LINK PASSWORD
```
> Leave the `PASSWORD` blank if it's a public shared file.

### Cancel File Share
```bash
cancelshare SHARE_ID
```

### View Resource Utilization
Type `monitor` to show the resource utilization monitor, and `stopmonitor`to hide it.
```shell
# show the resource utilization monitor
monitor

# hide the resource utilization monitor
stopmonitor
```

You can exit the `ppd terminal` command-line terminal by typing `exit` and leave the `ppd start` terminal to run the resource node in background.

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
