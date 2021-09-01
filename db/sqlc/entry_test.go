package db

import (
	"context"
	"testing"
	"time"

	"github.com/qwerqy/mock_bank/util"
	"github.com/stretchr/testify/require"
)

func createRandomEntry(t *testing.T, AccountId int64) Entry {
	arg := CreateEntryParams{
		AccountID: AccountId,
		Amount:    util.RandomMoney(),
	}

	entry, err := testQueries.CreateEntry(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, entry)

	require.Equal(t, entry.AccountID, arg.AccountID)
	require.Equal(t, entry.Amount, arg.Amount)

	return entry
}

func createEntriesFromOneAccount(t *testing.T, AccountId int64) {

	for i := 0; i < 10; i++ {
		createRandomEntry(t, AccountId)
	}
}

func TestCreateEntry(t *testing.T) {
	account1 := createRandomAccount(t)
	createRandomEntry(t, account1.ID)
}

func TestGetEntry(t *testing.T) {
	account1 := createRandomAccount(t)
	entry1 := createRandomEntry(t, account1.ID)

	entry2, err := testQueries.GetEntry(context.Background(), entry1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, entry2)

	require.Equal(t, entry2.AccountID, entry1.AccountID)
	require.Equal(t, entry2.Amount, entry1.Amount)
	require.Equal(t, entry2.ID, entry1.ID)
	require.WithinDuration(t, entry2.CreatedAt, entry1.CreatedAt, time.Second)
}

func TestListEntry(t *testing.T) {
	account1 := createRandomAccount(t)

	for i := 0; i < 10; i++ {
		createEntriesFromOneAccount(t, account1.ID)
	}

	arg := ListEntryParams{
		AccountID: account1.ID,
		Limit:     5,
		Offset:    5,
	}

	entries, err := testQueries.ListEntry(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, entries, 5)

	for _, entry := range entries {
		require.NotEmpty(t, entry)
	}
}
