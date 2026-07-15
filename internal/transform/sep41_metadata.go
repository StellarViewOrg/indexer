package transform

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"

	"github.com/stellar/go-stellar-sdk/strkey"
	"github.com/stellar/go-stellar-sdk/xdr"

	"github.com/miguelnietoa/stellar-explorer/indexer/internal/source"
)

// placeholderSourceAccount is a dummy Stellar account used as the transaction
// source for simulateTransaction calls. Simulation does not validate the account
// exists or has a valid sequence number.
const placeholderSourceAccount = "GBKPF4URAGUGPBFKQNMDDD4IY5BRRXRK2VEBULJEMVULCCODND436NIO"

// FetchSep41Metadata calls simulateTransaction for name(), symbol(), and decimals()
// on the given SEP-41 contract. Returns nil pointers for any value that cannot be
// fetched — the caller should treat nil as "unknown". Never returns a non-nil error;
// partial failures are logged and result in nil fields.
func FetchSep41Metadata(ctx context.Context, rpc *source.RPCClient, contractID string) (name, symbol *string, decimals *int32, err error) {
	nameVal, nameErr := simulateNoArgStringCall(ctx, rpc, contractID, "name")
	if nameErr != nil {
		log.Printf("sep41_metadata: %s name(): %v", contractID, nameErr)
	}

	symbolVal, symbolErr := simulateNoArgStringCall(ctx, rpc, contractID, "symbol")
	if symbolErr != nil {
		log.Printf("sep41_metadata: %s symbol(): %v", contractID, symbolErr)
	}

	decimalsVal, decimalsErr := simulateNoArgUint32Call(ctx, rpc, contractID, "decimals")
	if decimalsErr != nil {
		log.Printf("sep41_metadata: %s decimals(): %v", contractID, decimalsErr)
	}

	return nameVal, symbolVal, decimalsVal, nil
}

// simulateNoArgStringCall invokes a no-arg contract function that returns a string.
func simulateNoArgStringCall(ctx context.Context, rpc *source.RPCClient, contractID, functionName string) (*string, error) {
	scVal, err := simulateCall(ctx, rpc, contractID, functionName)
	if err != nil {
		return nil, err
	}
	if scVal.Type != xdr.ScValTypeScvString {
		return nil, fmt.Errorf("%s() returned unexpected type %v", functionName, scVal.Type)
	}
	s := string(*scVal.Str)
	return &s, nil
}

// simulateNoArgUint32Call invokes a no-arg contract function that returns a uint32.
func simulateNoArgUint32Call(ctx context.Context, rpc *source.RPCClient, contractID, functionName string) (*int32, error) {
	scVal, err := simulateCall(ctx, rpc, contractID, functionName)
	if err != nil {
		return nil, err
	}
	if scVal.Type != xdr.ScValTypeScvU32 {
		return nil, fmt.Errorf("%s() returned unexpected type %v", functionName, scVal.Type)
	}
	d := int32(*scVal.U32)
	return &d, nil
}

// simulateCall builds a no-arg InvokeHostFunction transaction, simulates it via
// the Stellar RPC, and returns the first result as a parsed ScVal.
func simulateCall(ctx context.Context, rpc *source.RPCClient, contractID, functionName string) (xdr.ScVal, error) {
	txXDR, err := buildInvokeHostFunctionTxXDR(contractID, functionName)
	if err != nil {
		return xdr.ScVal{}, fmt.Errorf("build tx: %w", err)
	}

	result, err := rpc.SimulateTransaction(ctx, txXDR)
	if err != nil {
		return xdr.ScVal{}, fmt.Errorf("simulate: %w", err)
	}
	if result.Error != "" {
		return xdr.ScVal{}, fmt.Errorf("simulation error: %s", result.Error)
	}
	if len(result.Results) == 0 {
		return xdr.ScVal{}, fmt.Errorf("no results returned")
	}

	var scVal xdr.ScVal
	if err := xdr.SafeUnmarshalBase64(result.Results[0].XDR, &scVal); err != nil {
		return xdr.ScVal{}, fmt.Errorf("unmarshal result ScVal: %w", err)
	}
	return scVal, nil
}

// buildInvokeHostFunctionTxXDR constructs a minimal TransactionEnvelope XDR that
// calls a no-arg function on the given contract. The result is base64-encoded and
// suitable for passing to simulateTransaction.
func buildInvokeHostFunctionTxXDR(contractID, functionName string) (string, error) {
	contractIDBytes, err := strkey.Decode(strkey.VersionByteContract, contractID)
	if err != nil {
		return "", fmt.Errorf("decode contract ID: %w", err)
	}
	var cID xdr.ContractId
	copy(cID[:], contractIDBytes)

	acctBytes, err := strkey.Decode(strkey.VersionByteAccountID, placeholderSourceAccount)
	if err != nil {
		return "", fmt.Errorf("decode placeholder account: %w", err)
	}
	var ed25519Key xdr.Uint256
	copy(ed25519Key[:], acctBytes)
	sourceAccount := xdr.MuxedAccount{
		Type:    xdr.CryptoKeyTypeKeyTypeEd25519,
		Ed25519: &ed25519Key,
	}

	op := xdr.Operation{
		Body: xdr.OperationBody{
			Type: xdr.OperationTypeInvokeHostFunction,
			InvokeHostFunctionOp: &xdr.InvokeHostFunctionOp{
				HostFunction: xdr.HostFunction{
					Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
					InvokeContract: &xdr.InvokeContractArgs{
						ContractAddress: xdr.ScAddress{
							Type:       xdr.ScAddressTypeScAddressTypeContract,
							ContractId: &cID,
						},
						FunctionName: xdr.ScSymbol(functionName),
						Args:         xdr.ScVec{},
					},
				},
				Auth: []xdr.SorobanAuthorizationEntry{},
			},
		},
	}

	tx := xdr.Transaction{
		SourceAccount: sourceAccount,
		Fee:           100,
		SeqNum:        0,
		Cond:          xdr.Preconditions{Type: xdr.PreconditionTypePrecondNone},
		Memo:          xdr.Memo{Type: xdr.MemoTypeMemoNone},
		Operations:    []xdr.Operation{op},
		Ext: xdr.TransactionExt{
			V: 1,
			SorobanData: &xdr.SorobanTransactionData{
				Ext:         xdr.SorobanTransactionDataExt{V: 0},
				Resources:   xdr.SorobanResources{},
				ResourceFee: 0,
			},
		},
	}

	env := xdr.TransactionEnvelope{
		Type: xdr.EnvelopeTypeEnvelopeTypeTx,
		V1: &xdr.TransactionV1Envelope{
			Tx:         tx,
			Signatures: []xdr.DecoratedSignature{},
		},
	}

	eb := xdr.NewEncodingBuffer()
	b, err := eb.MarshalBinary(env)
	if err != nil {
		return "", fmt.Errorf("marshal envelope: %w", err)
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
