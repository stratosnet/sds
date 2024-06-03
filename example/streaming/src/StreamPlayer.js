import React, {useContext} from 'react';
import videojs from 'video.js';
import VideoJS from './VideoJS'
import * as stratosSdk from "@stratos-network/stratos-sdk.js";
import {useParams} from "react-router";
import { useSearchParams } from "react-router-dom";
import {KeyPairContext} from "./KeyPairContext";
import LinkType from "./LinkType";

const networkInfo = {
    url: process.env.REACT_APP_NODE_REST_URL
}

const StreamPlayer = () => {
    const [searchParams, setSearchParams] = useSearchParams();
    const params= useParams()

    const linkType = params.linkType
    const keyPair = useContext(KeyPairContext)

    const [streamInfo, setStreamInfo] = React.useState(null);
    const playerRef = React.useRef(null);
    const { url } = networkInfo;

    function handlePlayerReady(player) {
        playerRef.current = player;

        player.on('waiting', () => {
            videojs.log('player is waiting');
        });

        player.on('dispose', () => {
            videojs.log('player will dispose');
        });
    }

    function playVideo(player) {
        videojs.Vhs.xhr.beforeRequest = function (options) {
            const videoSegment = options.uri.split('/').pop();
            const sliceInfo = streamInfo.segment_to_slice_info[videoSegment];
            options.uri = `${url}/getVideoSliceCache/${sliceInfo.slice_storage_info.slice_hash}`
            options.method = "POST";
            options.body = JSON.stringify({
                fileHash: streamInfo.file_hash,
                fileReqId: streamInfo.req_id,
                sliceInfo,
            })
            return options;
        };
        player.ready(() => {
            player.src({
                src: `${url}/getVideoSliceCache/${streamInfo.header_file}`,
                type: "application/x-mpegURL",
            });
        });
    }

    async function fetchStreamInfo() {
        const walletAddress = keyPair.address;
        const fileHash = params.link;
        const ozoneResp = await fetch(`${url}/getOzone/${walletAddress}`);
        const ozoneInfo = await ozoneResp.json();
        const reqTime = Math.floor(Date.now() / 1000);
        const message = fileHash + walletAddress + ozoneInfo.sequenceNumber + reqTime

        let signature = "";
        if (keyPair?.privateKey != null) {
            signature = await stratosSdk.crypto.hdVault.keyUtils.signWithPrivateKey(message, keyPair.privateKey);
        }

        const requestOptions = {
            method: "POST",
            mode: "cors",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify({
                pubKey: keyPair.publicKey,
                walletAddress,
                signature,
                reqTime
            })
        };

        const streamInfoResp = await fetch(`${url}/prepareVideoFileCache/${walletAddress}/${fileHash}`, requestOptions);
        return await streamInfoResp.json()
    }

    async function fetchStreamInfoByShareLink() {
        const walletAddress = keyPair.address;
        const shareLink = params.link;
        const sharePassword = searchParams.get("pw")
        const ozoneResp = await fetch(`${url}/getOzone/${walletAddress}`);
        const ozoneInfo = await ozoneResp.json();
        const reqTime = Math.floor(Date.now() / 1000);
        const message = shareLink + walletAddress + ozoneInfo.sequenceNumber + reqTime;

        let signature = "";
        if (keyPair?.privateKey != null) {
            signature = await stratosSdk.crypto.hdVault.keyUtils.signWithPrivateKey(message, keyPair.privateKey);
        }

        const requestOptions = {
            method: "POST",
            mode: "cors",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify({
                pubKey: keyPair.publicKey,
                walletAddress,
                signature,
                reqTime
            })
        };

        const streamInfoResp = await fetch(`${url}/prepareSharedVideoFileCache/${shareLink}?password=${sharePassword}`, requestOptions);
        return await streamInfoResp.json()
    }

    React.useEffect(() => {
        if (keyPair == null || linkType == null) {
            return
        }

        switch (linkType) {
            case LinkType.FILE_HASH:
                fetchStreamInfo(keyPair?.address)
                    .then(info => setStreamInfo(info));
                return
            case LinkType.SHARE_LINK:
                fetchStreamInfoByShareLink(keyPair?.address)
                    .then(info => setStreamInfo(info));
                return
            default:
                return
        }
    }, [keyPair]);

    React.useEffect(() => {
        if (streamInfo == null) {
            return
        }
        playVideo(playerRef.current)
    }, [streamInfo])


    const videoJsOptions = {
        autoplay: false,
        controls: true,
        responsive: true,
        width: 1280,
        height: 730
    };

    return (
        <div className="center-player">
            <VideoJS options={videoJsOptions} onReady={handlePlayerReady} />
        </div>
    );
}

export default StreamPlayer;