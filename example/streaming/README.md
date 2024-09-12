# Stratos Streaming Demo

## Getting Started
The project has two parts: client and server. The client is a pure frontend project and can run standalone and plays videos 
from SDS Network. However, running the project without a dedicated backend server means that user has to include the wallet 
key information in the build package and there's risks of leaking wallet key if the website is open to public. We highly 
encourage launching the node server that is dedicated for signing messages.

### Client
#### Setup wallet and network info
Before starting the project, user needs to set up wallet as well as network information in the `.env` file. Please rename
the `.env.template` file to `.env` and input the required variables accordingly.

```
#wallet address of the user that is logging in (mandatory)
REACT_APP_WALLET_ADDRESS=

#mnemonic of the user that is logging in, could be empty if a backend server is setup for signing message (optional, when not given, a backend server is needed)
REACT_APP_WALLET_MNEMONIC=

#your login logic to check and verify the password or pin (optional, when not given, a backend server is needed)
REACT_APP_WALLET_PASSWORD=

#url to the rest api of the sds node
REACT_APP_SERVICE_URL=

#url to the backend server that signs message, could be empty if REACT_APP_WALLET_MNEMONIC is given and let frontend handles message signing
REACT_APP_NODE_SIGN_URL=
```
#### Add video links
In the file `videos.json`, user can add links to the streaming videos in the SDS network. There are two types of links, 
file hash and share link. To play the video by file hash, the wallet that is configured in the `.env` has to own the video 
file while the share link doesn't have this requirement as long as user gives the correct share password if there's one.

```json
[
  {
    "linkType": "filehash",
    "fileHash": "<file hash>",
    "fileName": "<file name 1>"
  },
  {
    "linkType": "sharelink",
    "shareLink": "<share link>",
    "sharePassword": "<share password>",
    "fileName": "<file name 2>"
  }
]
```

### Start the Development environment
To start the frontend UI, please execute the following commands

```bash
$ npm i
$ npm start
```

After start, user can see a table that contains video links that are given in the `videos.json` file under `client` folder. 
Click on the link will redirect to the video player page and user can start playing the video once the metadata and the 
first video segment are fetched from the SDS network. The last row of the table gives a way for user to play a video by 
inputting video link. Modifying `videos.json` and saving the file, the code would be re-compiled, and the rendered 
video table would be updated.

### Build
Execute the following commands in the project directory to build resources for execution in the production environment.

```bash

$ npm build

```

> Compiled bundles as well as the exported types, would be located in **"root directory/dist"**

### Server
#### Setup wallet info
Similar to the frontend project, user needs to set up wallet as well as network information in the `.env` file. Please rename
the `.env.template` file to `.env` and input the required variables accordingly.

```
#port that the service is listening to
PORT=

#wallet address of the user that is logging in,
WALLET_ADDRESS=

#mnemonic of the user that is logging in,
WALLET_MNEMONIC=

#your login logic to check and verify the password or pin
WALLET_PASSWORD=
```

### Add video links
User can also add video links to the `videos.json` file under the `server` folder and the links will be provided to the 
frontend via api call.

#### Start the server
To start the node server, please execute the following command

```bash
$ node server.js