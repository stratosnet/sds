package tx

import (
	"fmt"

	"github.com/cosmos/gogoproto/proto"

	multisigv1beta1 "cosmossdk.io/api/cosmos/crypto/multisig/v1beta1"
	txv1beta1 "cosmossdk.io/api/cosmos/tx/v1beta1"

	txsigning "github.com/stratosnet/sds/tx-client/types/tx/signing"
)

// SignatureDataToModeInfoAndSig converts a SignatureData to a ModeInfo and raw bytes signature
func SignatureDataToModeInfoAndSig(data txsigning.SignatureData) (*txv1beta1.ModeInfo, []byte) {
	if data == nil {
		return nil, nil
	}

	switch data := data.(type) {
	case *txsigning.SingleSignatureData:
		return &txv1beta1.ModeInfo{
			Sum: &txv1beta1.ModeInfo_Single_{
				Single: &txv1beta1.ModeInfo_Single{Mode: data.SignMode},
			},
		}, data.Signature
	case *txsigning.MultiSignatureData:
		n := len(data.Signatures)
		modeInfos := make([]*txv1beta1.ModeInfo, n)
		sigs := make([][]byte, n)

		for i, d := range data.Signatures {
			modeInfos[i], sigs[i] = SignatureDataToModeInfoAndSig(d)
		}

		multisig := multisigv1beta1.MultiSignature{
			Signatures: sigs,
		}

		sig, err := proto.Marshal(&multisig)
		//sig, err := multisig.Marshal() //TODO compare difference
		if err != nil {
			panic(err)
		}

		return &txv1beta1.ModeInfo{
			Sum: &txv1beta1.ModeInfo_Multi_{
				Multi: &txv1beta1.ModeInfo_Multi{
					Bitarray:  data.BitArray,
					ModeInfos: modeInfos,
				},
			},
		}, sig
	default:
		panic(fmt.Sprintf("unexpected signature data type %T", data))
	}
}
