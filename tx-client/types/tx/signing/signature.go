package signing

import (
	"fmt"

	signingv1beta1 "cosmossdk.io/api/cosmos/tx/signing/v1beta1"
	"google.golang.org/protobuf/types/known/anypb"
)

// SignatureV2 is a convenience type that is easier to use in application logic
// than the protobuf SignerInfo's and raw signature bytes. It goes beyond the
// first sdk.Signature types by supporting sign modes and explicitly nested
// multi-signatures. It is intended to be used for both building and verifying
// signatures.
type SignatureV2 struct {
	// PubKey is the public key to use for verifying the signature
	PubKey *anypb.Any

	// Data is the actual data of the signature which includes SignMode's and
	// the signatures themselves for either single or multi-signatures.
	Data SignatureData

	// Sequence is the sequence of this account. Only populated in
	// SIGN_MODE_DIRECT.
	Sequence uint64
}

// SignatureDataToProto converts a SignatureData to SignatureDescriptor_Data.
// SignatureDescriptor_Data is considered an encoding type whereas SignatureData is used for
// business logic.
func SignatureDataToProto(data SignatureData) *signingv1beta1.SignatureDescriptor_Data {
	switch data := data.(type) {
	case *SingleSignatureData:
		return &signingv1beta1.SignatureDescriptor_Data{
			Sum: &signingv1beta1.SignatureDescriptor_Data_Single_{
				Single: &signingv1beta1.SignatureDescriptor_Data_Single{
					Mode:      data.SignMode,
					Signature: data.Signature,
				},
			},
		}
	case *MultiSignatureData:
		descDatas := make([]*signingv1beta1.SignatureDescriptor_Data, len(data.Signatures))

		for j, d := range data.Signatures {
			descDatas[j] = SignatureDataToProto(d)
		}

		return &signingv1beta1.SignatureDescriptor_Data{
			Sum: &signingv1beta1.SignatureDescriptor_Data_Multi_{
				Multi: &signingv1beta1.SignatureDescriptor_Data_Multi{
					Bitarray:   data.BitArray,
					Signatures: descDatas,
				},
			},
		}
	default:
		panic(fmt.Errorf("unexpected case %+v", data))
	}
}

// SignatureDataFromProto converts a SignatureDescriptor_Data to SignatureData.
// SignatureDescriptor_Data is considered an encoding type whereas SignatureData is used for
// business logic.
func SignatureDataFromProto(descData *signingv1beta1.SignatureDescriptor_Data) SignatureData {
	switch descData := descData.Sum.(type) {
	case *signingv1beta1.SignatureDescriptor_Data_Single_:
		return &SingleSignatureData{
			SignMode:  descData.Single.Mode,
			Signature: descData.Single.Signature,
		}
	case *signingv1beta1.SignatureDescriptor_Data_Multi_:
		multi := descData.Multi
		datas := make([]SignatureData, len(multi.Signatures))

		for j, d := range multi.Signatures {
			datas[j] = SignatureDataFromProto(d)
		}

		return &MultiSignatureData{
			BitArray:   multi.Bitarray,
			Signatures: datas,
		}
	default:
		panic(fmt.Errorf("unexpected case %+v", descData))
	}
}

//var _, _ codectypes.UnpackInterfacesMessage = &SignatureDescriptors{}, &SignatureDescriptor{}
//
//// UnpackInterfaces implements the UnpackInterfaceMessages.UnpackInterfaces method
//func (sds *SignatureDescriptors) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
//	for _, sig := range sds.Signatures {
//		err := sig.UnpackInterfaces(unpacker)
//		if err != nil {
//			return err
//		}
//	}
//
//	return nil
//}
//
//// UnpackInterfaces implements the UnpackInterfaceMessages.UnpackInterfaces method
//func (sd *SignatureDescriptor) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
//	return unpacker.UnpackAny(sd.PublicKey, new(fwcryptotypes.PubKey))
//}
