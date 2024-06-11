require('dotenv').config();
const express = require('express');
const cors = require('cors');
const stratosSdk = require("@stratos-network/stratos-sdk.js");
const fs = require('fs');

const app = express();
const port = process.env.PORT;

app.use(cors());
app.use(express.json());

app.post("/api/sign", async (req, res) => {
    const {walletAddress, link, sequenceNumber, reqTime} = req.body;
    const keyPair = await initKeyPair()
    if (keyPair.address === walletAddress && keyPair?.privateKey != null) {
        const message = link + keyPair.address + sequenceNumber + reqTime;
        const signature = await stratosSdk.crypto.hdVault.keyUtils.signWithPrivateKey(message, keyPair.privateKey);
        res.json({
            signature,
            pubKey: keyPair.publicKey
        });
        return;
    }
    res.json({})
});

app.get("/api/link-list", async(req, res) => {
    const obj = JSON.parse(fs.readFileSync('videos.json', 'utf8'));
    res.json(obj)
})

app.listen(port, () => {
    console.log(`Server is running on port ${port}.`);
});

async function initKeyPair() {
    const mnemonic = process.env.WALLET_MNEMONIC;
    const password = process.env.WALLET_PASSWORD;

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