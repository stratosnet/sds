import React, {useContext} from "react";
import videojs from "video.js";
import VideoJS from "./VideoJS"
import * as stratosSdk from "@stratos-network/stratos-sdk.js";
import {useParams} from "react-router";
import { useSearchParams } from "react-router-dom";
import {KeyPairContext} from "./KeyPairContext";
import LinkType from "./LinkType";

const networkInfo = {
    videoServiceUrl: process.env.REACT_APP_SERVICE_URL,
    signUrl: process.env.REACT_APP_NODE_SIGN_URL,
    serviceToken: process.env.REACT_APP_SERVICE_TOKEN
}

function getVideoUrl(api, ...parameters) {
    let apiUrl = `${networkInfo.videoServiceUrl}/${api}`;
    if (networkInfo.serviceToken) {
        apiUrl =  `${apiUrl}/${networkInfo.serviceToken}`;
    }
    if (parameters.length === 0) {
        return apiUrl;
    }
    return `${apiUrl}/${[...parameters].join("/")}`;
}

const StreamPlayer = () => {
    const [searchParams, setSearchParams] = useSearchParams();
    const params= useParams();

    const linkType = params.linkType;
    const {walletAddress, keyPair}  = useContext(KeyPairContext);

    const [streamInfo, setStreamInfo] = React.useState(null);
    const [isReady, setIsReady] = React.useState(false);
    const { signUrl } = networkInfo;

    async function fetchStreamInfo() {
        const fileHash = params.link;
        const ozoneResp = await fetch(getVideoUrl("getOzone", walletAddress));
        const ozoneInfo = await ozoneResp.json();
        const reqTime = Math.floor(Date.now() / 1000);

        const { signature, pubKey } = await getSignature({
            walletAddress,
            link: fileHash,
            sequenceNumber: ozoneInfo.sequenceNumber,
            reqTime
        });

        const requestOptions = {
            method: "POST",
            mode: "cors",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify({
                pubKey,
                walletAddress,
                signature,
                reqTime
            })
        };

        const streamInfoResp = await fetch(getVideoUrl("prepareVideoFileCache", walletAddress, fileHash), requestOptions);
        return await streamInfoResp.json()
    }

    async function fetchStreamInfoByShareLink() {
        const shareLink = params.link;
        const sharePassword = searchParams.get("pw")
        const ozoneResp = await fetch(getVideoUrl("getOzone", walletAddress));
        const ozoneInfo = await ozoneResp.json();
        const reqTime = Math.floor(Date.now() / 1000);

        const { signature, pubKey } = await getSignature({
            walletAddress,
            link: shareLink,
            sequenceNumber: ozoneInfo.sequenceNumber,
            reqTime
        });

        const requestOptions = {
            method: "POST",
            mode: "cors",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify({
                pubKey,
                walletAddress,
                signature,
                reqTime
            })
        };

        const streamInfoResp = await fetch(getVideoUrl("prepareSharedVideoFileCache", shareLink) + `?password=${sharePassword}`, requestOptions);
        return await streamInfoResp.json()
    }

    async function getSignature(body) {
        if (keyPair?.privateKey != null) {
            const {walletAddress, link, sequenceNumber, reqTime} = body;
            const message = link + walletAddress + sequenceNumber + reqTime;
            const signature = await stratosSdk.crypto.hdVault.keyUtils.signWithPrivateKey(message, keyPair.privateKey);
            return {
                signature,
                pubKey: keyPair.publicKey
            }
        } else if (signUrl) {
            const signResp = await fetch(`${signUrl}/api/sign`, {
                method: "POST",
                mode: "cors",
                headers: {
                    "Content-Type": "application/json",
                },
                body: JSON.stringify(body)
            })
            return await signResp.json()
        }
        return {};
    }

    function handlePlayerReady(player) {
        player.on("waiting", () => {
            videojs.log("player is waiting");
        });

        player.on("dispose", () => {
            videojs.log("player will dispose");
        });
    }

    React.useEffect(() => {
        if (walletAddress == null || linkType == null) {
            return
        }

        switch (linkType) {
            case LinkType.FILE_HASH:
                fetchStreamInfo()
                    .then(info => setStreamInfo(info));
                return
            case LinkType.SHARE_LINK:
                fetchStreamInfoByShareLink()
                    .then(info => setStreamInfo(info));
                return
            default:
                return
        }
    }, [walletAddress, keyPair]);

    React.useEffect(() => {
        if (streamInfo == null) {
            return
        }
        setIsReady(true);
    }, [streamInfo])

    const videoJsOptions = {
        autoplay: false,
        controls: true,
        responsive: true,
        withCredentials: false,
        nativeControlsForTouch: true,
        html5: {
            vhs: {
                withCredentials: false,
            }
        },
    };

    return (
        (isReady
            ? <div className="video-js-responsive-container vjs-hd">
                    <VideoJS
                        options={{
                            ...videoJsOptions,
                            sources: {
                                src: getVideoUrl("getVideoSliceCache", streamInfo.reqId, streamInfo.headerFile),
                                type: "application/x-mpegURL",
                            }
                        }}
                        onReady={handlePlayerReady}
                    />
                </div>
            : <div className="center">
                    <div className="loader"/>
              </div>
        )
    );
}

export default StreamPlayer;