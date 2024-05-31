import React, { useState, createContext, useEffect } from "react";
import * as stratosSdk from "@stratos-network/stratos-sdk.js";

export const KeyPairContext = createContext(null);

const KeyPairContextProvider = (props) => {
    const [keyPair, setKeyPair] = useState(null);

    async function initKeyPair () {
        const mnemonic = process.env.REACT_APP_MNEMONIC;
        const password = process.env.REACT_APP_WALLET_PASSWORD;

        const hdPathIndex = 0;

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
        initKeyPair().then(key => setKeyPair(key))
    }, []);

    return(
        <KeyPairContext.Provider value={keyPair}>
            {props.children}
        </KeyPairContext.Provider>
    )
}

export default KeyPairContextProvider;
