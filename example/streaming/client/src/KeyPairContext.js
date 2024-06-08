import React, { useState, createContext, useEffect } from "react";
import * as stratosSdk from "@stratos-network/stratos-sdk.js";

export const KeyPairContext = createContext({});

const KeyPairContextProvider = (props) => {
    const [keyPair, setKeyPair] = useState(null);
    const [walletAddress, setWalletAddress] = useState(null);

    async function initKeyPair () {
        const mnemonic = process.env.REACT_APP_WALLET_MNEMONIC;
        const password = process.env.REACT_APP_WALLET_PASSWORD;
        const hdPathIndex = 0;

        if (!mnemonic) {
            return null
        }

        const phrase = stratosSdk.crypto.hdVault.mnemonic.convertStringToArray(mnemonic);
        const masterKeySeedInfo = await stratosSdk.crypto.hdVault.keyManager.createMasterKeySeed(
            phrase,
            password,
            hdPathIndex,
        );

        const encryptedMasterKeySeed = masterKeySeedInfo.encryptedMasterKeySeed.toString();
        return await stratosSdk.crypto.hdVault.wallet.deriveKeyPair(
            hdPathIndex,
            password,
            encryptedMasterKeySeed,
        );
    }

    useEffect(() => {
        const walletAddress = process.env.REACT_APP_WALLET_ADDRESS;
        initKeyPair().then(key => {
            if (key?.address === walletAddress) {
                setKeyPair(key);
            }
            setWalletAddress(walletAddress);
        });
    }, []);

    return(
        <KeyPairContext.Provider value={{keyPair, walletAddress}}>
            {props.children}
        </KeyPairContext.Provider>
    )
}

export default KeyPairContextProvider;
