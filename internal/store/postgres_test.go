package store

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func getTestDB(t *testing.T) *PostgresStore {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = "postgresql://explorer:explorer_dev@localhost:54320/stellar_explorer?sslmode=disable"
	}
	store, err := NewPostgresStore(url)
	if err != nil {
		t.Skipf("Skipping: cannot connect to test database: %v", err)
	}
	return store
}

func TestInsertLedger(t *testing.T) {
	store := getTestDB(t)
	defer store.Close()

	now := time.Now().UTC().Truncate(time.Microsecond)
	ledger := &Ledger{
		Sequence:          99999999,
		Hash:              "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		PrevHash:          "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		ClosedAt:          now,
		TotalCoins:        1000000000000,
		FeePool:           100000,
		BaseFee:           100,
		BaseReserve:       5000000,
		MaxTxSetSize:      1000,
		ProtocolVersion:   21,
		TransactionCount:  5,
		OperationCount:    10,
		SuccessfulTxCount: 4,
		FailedTxCount:     1,
	}

	ctx := context.Background()

	err := store.InsertLedger(ctx, ledger)
	if err != nil {
		t.Fatalf("InsertLedger failed: %v", err)
	}

	// Insert again — should be idempotent (ON CONFLICT DO NOTHING)
	err = store.InsertLedger(ctx, ledger)
	if err != nil {
		t.Fatalf("Idempotent InsertLedger failed: %v", err)
	}

	// Clean up test data
	_, _ = store.db.ExecContext(ctx, "DELETE FROM ledgers WHERE sequence = 99999999")
}

func TestInsertTransactionBatch(t *testing.T) {
	store := getTestDB(t)
	defer store.Close()

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)

	txs := []Transaction{
		{
			Hash:             "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			LedgerSequence:   88888888,
			ApplicationOrder: 1,
			Account:          "GABC",
			AccountSequence:  100,
			FeeCharged:       100,
			MaxFee:           200,
			OperationCount:   1,
			MemoType:         0,
			Status:           1,
			IsSoroban:        false,
			EnvelopeXDR:      "AAAA",
			ResultXDR:        "BBBB",
			CreatedAt:        now,
		},
	}

	err := store.InsertTransactionBatch(ctx, txs)
	if err != nil {
		t.Fatalf("InsertTransactionBatch failed: %v", err)
	}

	// Insert again — idempotent
	err = store.InsertTransactionBatch(ctx, txs)
	if err != nil {
		t.Fatalf("Idempotent InsertTransactionBatch failed: %v", err)
	}

	// Empty batch should be no-op
	err = store.InsertTransactionBatch(ctx, nil)
	if err != nil {
		t.Fatalf("Empty InsertTransactionBatch failed: %v", err)
	}

	// Clean up
	_, _ = store.db.ExecContext(ctx, "DELETE FROM transactions WHERE hash = 'cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc'")
}

func TestInsertOperationBatch(t *testing.T) {
	store := getTestDB(t)
	defer store.Close()

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	typeName := "payment"

	ops := []Operation{
		{
			TransactionID:    0,
			TransactionHash:  "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
			ApplicationOrder: 1,
			Type:             1,
			TypeName:         typeName,
			Details:          `{"type": "payment"}`,
			CreatedAt:        now,
		},
	}

	err := store.InsertOperationBatch(ctx, ops)
	if err != nil {
		t.Fatalf("InsertOperationBatch failed: %v", err)
	}

	// Clean up
	_, _ = store.db.ExecContext(ctx, "DELETE FROM operations WHERE transaction_hash = 'dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd'")
}

func TestIngestionState(t *testing.T) {
	store := getTestDB(t)
	defer store.Close()

	ctx := context.Background()

	// Clean up first to ensure a fresh state
	_, _ = store.db.ExecContext(ctx, "DELETE FROM ingestion_state WHERE key = 'last_ingested_ledger'")

	err := store.SetLastIngestedLedger(ctx, 12345)
	if err != nil {
		t.Fatalf("SetLastIngestedLedger failed: %v", err)
	}

	seq, err := store.GetLastIngestedLedger(ctx)
	if err != nil {
		t.Fatalf("GetLastIngestedLedger failed: %v", err)
	}
	if seq != 12345 {
		t.Errorf("expected 12345, got %d", seq)
	}

	// Verify forward-only: setting a lower value should not regress
	err = store.SetLastIngestedLedger(ctx, 100)
	if err != nil {
		t.Fatalf("SetLastIngestedLedger (lower) failed: %v", err)
	}
	seq, err = store.GetLastIngestedLedger(ctx)
	if err != nil {
		t.Fatalf("GetLastIngestedLedger after lower set failed: %v", err)
	}
	if seq != 12345 {
		t.Errorf("cursor should not regress: expected 12345, got %d", seq)
	}

	// Clean up
	_, _ = store.db.ExecContext(ctx, "DELETE FROM ingestion_state WHERE key = 'last_ingested_ledger'")
}

// insertSyntheticLedgers inserts one minimal ledger row per sequence in
// [from, to] (inclusive), for use as fixtures in gap-detection tests.
func insertSyntheticLedgers(t *testing.T, store *PostgresStore, from, to uint32) {
	t.Helper()
	ctx := context.Background()
	base := time.Now().UTC().Truncate(time.Microsecond)

	for seq := from; seq <= to; seq++ {
		ledger := &Ledger{
			Sequence:        seq,
			Hash:            fmt.Sprintf("%064x", seq),
			PrevHash:        fmt.Sprintf("%064x", seq-1),
			ClosedAt:        base.Add(time.Duration(seq) * time.Second),
			TotalCoins:      1,
			FeePool:         1,
			BaseFee:         100,
			BaseReserve:     5000000,
			MaxTxSetSize:    1000,
			ProtocolVersion: 21,
		}
		if err := store.InsertLedger(ctx, ledger); err != nil {
			t.Fatalf("insertSyntheticLedgers: InsertLedger(%d) failed: %v", seq, err)
		}
	}
}

// TestFindMissingLedgerSequences is a hermetic gap-finding test: it never
// contacts the Stellar network, only a local Postgres instance. It ingests a
// contiguous range of synthetic ledgers, deletes a few rows to open a
// synthetic gap, and asserts that FindMissingLedgerSequences reports exactly
// those sequences and no others.
func TestFindMissingLedgerSequences(t *testing.T) {
	store := getTestDB(t)
	defer store.Close()

	ctx := context.Background()
	const from, to = 97000000, 97000019 // isolated test range

	_, _ = store.db.ExecContext(ctx, "DELETE FROM ledgers WHERE sequence >= $1 AND sequence <= $2", from, to)
	insertSyntheticLedgers(t, store, from, to)
	defer func() {
		_, _ = store.db.ExecContext(ctx, "DELETE FROM ledgers WHERE sequence >= $1 AND sequence <= $2", from, to)
	}()

	// Bounds should reflect the full range before any gap is introduced.
	min, max, err := store.GetLedgerSequenceBounds(ctx)
	if err != nil {
		t.Fatalf("GetLedgerSequenceBounds failed: %v", err)
	}
	if max < to || min > from {
		t.Fatalf("expected bounds to cover [%d,%d], got [%d,%d]", from, to, min, max)
	}

	missing, err := store.FindMissingLedgerSequences(ctx, from, to, 100)
	if err != nil {
		t.Fatalf("FindMissingLedgerSequences failed: %v", err)
	}
	if len(missing) != 0 {
		t.Fatalf("expected no gaps before deletion, got %v", missing)
	}

	// Punch a synthetic gap: delete a few non-contiguous rows.
	wantGaps := []uint32{from + 3, from + 4, from + 10}
	for _, seq := range wantGaps {
		if _, err := store.db.ExecContext(ctx, "DELETE FROM ledgers WHERE sequence = $1", seq); err != nil {
			t.Fatalf("failed to delete ledger %d: %v", seq, err)
		}
	}

	missing, err = store.FindMissingLedgerSequences(ctx, from, to, 100)
	if err != nil {
		t.Fatalf("FindMissingLedgerSequences failed: %v", err)
	}
	if len(missing) != len(wantGaps) {
		t.Fatalf("expected missing=%v, got %v", wantGaps, missing)
	}
	for i, seq := range wantGaps {
		if missing[i] != seq {
			t.Errorf("missing[%d] = %d, want %d", i, missing[i], seq)
		}
	}

	// The limit parameter must cap results even when more gaps are present.
	limited, err := store.FindMissingLedgerSequences(ctx, from, to, 2)
	if err != nil {
		t.Fatalf("FindMissingLedgerSequences with limit failed: %v", err)
	}
	if len(limited) != 2 {
		t.Fatalf("expected limit=2 to cap results at 2, got %d", len(limited))
	}
}
