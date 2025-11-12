# Quickstart

## Build docker image
In the project root folder (not the current folder):
```
$ docker build --tag sds -f example/ubuntu-24.04/Dockerfile .
```

## Run a docker container
You need to set some env vars to run a docker container:
```
$ docker run \
  --rm \
  --name sds-node \
  -p 18081:18081 \
  -e NETWORK_ADDRESS=$PUBLIC_IP \
  -e MNEMONIC_PHRASE="$MNEMONIC_PHRASE" \
  -v sds-data:/sds \
  sds
```

## Register peer to meta node
Start a new terminal and run:
```
$ docker exec -it sds-node ppd terminal
```

In the PPD terminal, enter the command `rp` to register peer:
```
> rp
```

## Upload and download files
If your wallet account has ozone, you can try uploading and downloading files on PPD terminal:
```
// upload file
> put <filepath>

// query uploaded file by self
> list <filename>

// download file
> get <sdm://account/filehash> <saveAs>

// delete file
> delete <filehash>
```

## Upload/Download without get into the docker container

```
docker exec -it sds-node ppd terminal exec 'put <filepath>'

docker exec -it sds-node ppd terminal exec 'get <sdm://account/filehash> <saveAs>'
```
