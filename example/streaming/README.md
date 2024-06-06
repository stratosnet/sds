# Stratos Streaming Demo

## Getting Started

### Before start
#### Setup wallet and network info
Before starting the project, user needs to set up wallet as well as network information in the `.env` file. Please rename
the `.env.template` file to `.env` and input the required variables accordingly.

```
# mnemonic of the user that is logging in,
REACT_APP_MNEMONIC=

# your login logic to check and verify the password or pin
REACT_APP_WALLET_PASSWORD=

# url to the rest api of the sds node
REACT_APP_NODE_REST_URL=
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
To start the project, execute the following commands

```bash
$ npm i
$ npm start
```

After start, user can see a table that contains video links that are given in the `video.json` file. Click on the link
will redirect to the video player page and user can start playing the video once the metadata and the first video 
segment are fetched from the SDS network. The last row of the table gives a way for user to play a video by 
inputting video link. Modifying `video.json` and saving the file, the code would be re-compiled, and the rendered 
video table would be updated.

### Build
Execute the following commands in the project directory to build resources for execution in the production environment.

```bash

$ npm build

```

> Compiled bundles as well as the exported types, would be located in **"root directory/dist"**
