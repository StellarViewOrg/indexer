package transform

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stellar/go-stellar-sdk/network"
	"github.com/stellar/go-stellar-sdk/xdr"

	"github.com/miguelnietoa/stellar-explorer/indexer/internal/source"
)

const testnetSep41Contract = "CBIELTK6YBZJU5UP2WWQEUCYKLPU6AUNZ2BQ4WWFEIE3USCIHMXQDAMA"

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
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
			Params struct {
				Transaction string `json:"transaction"`
			} `json:"params"`
			ID int `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		
		var env xdr.TransactionEnvelope
		if err := xdr.SafeUnmarshalBase64(req.Params.Transaction, &env); err != nil {
			t.Fatal(err)
		}
		
		funcName := string(env.V1.Tx.Operations[0].Body.InvokeHostFunctionOp.HostFunction.InvokeContract.FunctionName)
		
		var scVal xdr.ScVal
		switch funcName {
		case "name":
			s := xdr.ScString("USDC")
			scVal = xdr.ScVal{Type: xdr.ScValTypeScvString, Str: &s}
		case "symbol":
			s := xdr.ScString("USDC")
			scVal = xdr.ScVal{Type: xdr.ScValTypeScvString, Str: &s}
		case "decimals":
			d := xdr.Uint32(7)
			scVal = xdr.ScVal{Type: xdr.ScValTypeScvU32, U32: &d}
		}
		
		b, _ := scVal.MarshalBinary()
		xdrStr := base64.StdEncoding.EncodeToString(b)
		
		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result": map[string]interface{}{
				"latestLedger": 100,
				"results": []map[string]interface{}{
					{"xdr": xdrStr},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := source.NewRPCClient(ts.URL, network.TestNetworkPassphrase)
	ctx := context.Background()

	name, symbol, decimals, err := FetchSep41Metadata(ctx, client, testnetSep41Contract)
	if err != nil {
		t.Fatalf("FetchSep41Metadata returned error: %v", err)
	}
	if name == nil || *name != "USDC" {
		t.Errorf("expected USDC name")
	}
	if symbol == nil || *symbol != "USDC" {
		t.Errorf("expected USDC symbol")
	}
	if decimals == nil || *decimals != 7 {
		t.Errorf("expected 7 decimals")
	}
}
