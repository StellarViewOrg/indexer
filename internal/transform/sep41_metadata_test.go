package transform

import (
	"context"
	"os"
	"testing"

	"github.com/stellar/go-stellar-sdk/network"

	"github.com/miguelnietoa/stellar-explorer/indexer/internal/source"
)

// testnetSep41Contract is a known SEP-41 token on testnet.
// Find a valid contract at https://stellar.expert/explorer/testnet/contracts
// Look for contracts with is_sep41_token=true or search for "USDC" on testnet.
const testnetSep41Contract = "CBIELTK6YBZJU5UP2WWQEUCYKLPU6AUNZ2BQ4WWFEIE3USCIHMXQDAMA"

func getSep41RPCClient(t *testing.T) *source.RPCClient {
	if testing.Short() {
		t.Skip("skipping test that requires live network access")
	}
	endpoint := os.Getenv("TEST_RPC_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://soroban-testnet.stellar.org"
	}
	return source.NewRPCClient(endpoint, network.TestNetworkPassphrase)
}

func TestBuildInvokeHostFunctionTxXDR(t *testing.T) {
	txXDR, err := buildInvokeHostFunctionTxXDR(testnetSep41Contract, "decimals")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if txXDR == "" {
		t.Fatal("expected non-empty XDR string")
	}
	t.Logf("built tx XDR (first 60 chars): %.60s...", txXDR)
}

func TestFetchSep41Metadata(t *testing.T) {
	client := getSep41RPCClient(t)
	ctx := context.Background()

	name, symbol, decimals, err := FetchSep41Metadata(ctx, client, testnetSep41Contract)
	if err != nil {
		t.Fatalf("FetchSep41Metadata returned error: %v", err)
	}
	if name == nil || *name == "" {
		t.Error("expected non-empty name")
	}
	if symbol == nil || *symbol == "" {
		t.Error("expected non-empty symbol")
	}
	if decimals == nil {
		t.Error("expected non-nil decimals")
	}
	if name != nil && symbol != nil && decimals != nil {
		t.Logf("name=%q symbol=%q decimals=%d", *name, *symbol, *decimals)
	}
}
