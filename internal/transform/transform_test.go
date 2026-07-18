package transform

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stellar/go-stellar-sdk/network"

	"github.com/miguelnietoa/stellar-explorer/indexer/internal/source"
)

func loadLedgersFixture(t *testing.T) []source.LedgerEntry {
	t.Helper()
	data, err := os.ReadFile("testdata/ledgers.json")
	if err != nil {
		t.Fatalf("failed to read ledgers fixture: %v", err)
	}
	var ledgers []source.LedgerEntry
	if err := json.Unmarshal(data, &ledgers); err != nil {
		t.Fatalf("failed to unmarshal ledgers fixture: %v", err)
	}
	if len(ledgers) == 0 {
		t.Fatalf("no ledgers found in fixture")
	}
	return ledgers
}

func loadTransactionsFixture(t *testing.T) []source.TransactionEntry {
	t.Helper()
	data, err := os.ReadFile("testdata/transactions.json")
	if err != nil {
		t.Fatalf("failed to read transactions fixture: %v", err)
	}
	var txs []source.TransactionEntry
	if err := json.Unmarshal(data, &txs); err != nil {
		t.Fatalf("failed to unmarshal transactions fixture: %v", err)
	}
	if len(txs) == 0 {
		t.Fatalf("no transactions found in fixture")
	}
	return txs
}

func TestLedgerFromRPC(t *testing.T) {
	ledgers := loadLedgersFixture(t)

	for i, entry := range ledgers {
		ledger, err := LedgerFromRPC(entry)
		if err != nil {
			t.Fatalf("LedgerFromRPC[%d] failed: %v", i, err)
		}

		if ledger.Sequence == 0 {
			t.Errorf("ledger[%d]: expected non-zero sequence", i)
		}
		if ledger.Hash == "" {
			t.Errorf("ledger[%d]: expected non-empty hash", i)
		}
		if ledger.PrevHash == "" {
			t.Errorf("ledger[%d]: expected non-empty prev_hash", i)
		}
		if ledger.ClosedAt.IsZero() {
			t.Errorf("ledger[%d]: expected non-zero closed_at", i)
		}
		if ledger.ProtocolVersion == 0 {
			t.Errorf("ledger[%d]: expected non-zero protocol_version", i)
		}
		if ledger.BaseFee == 0 {
			t.Errorf("ledger[%d]: expected non-zero base_fee", i)
		}
		if ledger.HeaderXDR == nil {
			t.Errorf("ledger[%d]: expected non-nil header_xdr", i)
		}
	}
}

func TestTransactionFromRPC(t *testing.T) {
	txs := loadTransactionsFixture(t)

	for i, entry := range txs {
		tx, err := TransactionFromRPC(entry, network.TestNetworkPassphrase)
		if err != nil {
			t.Fatalf("TransactionFromRPC[%d] failed: %v", i, err)
		}

		if tx.Hash == "" {
			t.Errorf("tx[%d]: expected non-empty hash", i)
		}
		if tx.LedgerSequence == 0 {
			t.Errorf("tx[%d]: expected non-zero ledger_sequence", i)
		}
		if tx.Account == "" {
			t.Errorf("tx[%d]: expected non-empty account", i)
		}
		if tx.EnvelopeXDR == "" {
			t.Errorf("tx[%d]: expected non-empty envelope_xdr", i)
		}
		if tx.CreatedAt.IsZero() {
			t.Errorf("tx[%d]: expected non-zero created_at", i)
		}
	}
}

func TestOperationsFromRPC(t *testing.T) {
	txs := loadTransactionsFixture(t)

	totalOps := 0
	for i, entry := range txs {
		ops, err := OperationsFromRPC(entry, network.TestNetworkPassphrase)
		if err != nil {
			t.Fatalf("OperationsFromRPC[%d] failed: %v", i, err)
		}

		for j, op := range ops {
			if op.TransactionHash == "" {
				t.Errorf("tx[%d].op[%d]: expected non-empty transaction_hash", i, j)
			}
			if op.TypeName == "" {
				t.Errorf("tx[%d].op[%d]: expected non-empty type_name", i, j)
			}
			if op.Details == "" {
				t.Errorf("tx[%d].op[%d]: expected non-empty details", i, j)
			}
			if op.ApplicationOrder == 0 {
				t.Errorf("tx[%d].op[%d]: expected non-zero application_order", i, j)
			}
			totalOps++
		}
	}

	if totalOps == 0 {
		t.Error("expected at least one operation across transactions")
	}

	t.Logf("Parsed %d operations from %d transactions", totalOps, len(txs))
}
