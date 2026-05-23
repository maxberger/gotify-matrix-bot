package matrix_test

import (
	"context"
	"testing"

	. "gotify_matrix_bot/matrix"

	"gotest.tools/v3/assert"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type MockVerificationHelper struct {
	CancelCalled bool
	CancelTxnID  id.VerificationTransactionID
	CancelCode   event.VerificationCancelCode
	CancelReason string

	AcceptCalled bool
	AcceptTxnID  id.VerificationTransactionID
	AcceptChan   chan struct{}

	StartSASCalled bool
	StartSASTxnID  id.VerificationTransactionID

	ConfirmSASCalled bool
	ConfirmSASTxnID  id.VerificationTransactionID
	ConfirmChan      chan struct{}

	ErrToReturn error
}

func (m *MockVerificationHelper) CancelVerification(ctx context.Context, txnID id.VerificationTransactionID, code event.VerificationCancelCode, reason string) error {
	m.CancelCalled = true
	m.CancelTxnID = txnID
	m.CancelCode = code
	m.CancelReason = reason
	return m.ErrToReturn
}

func (m *MockVerificationHelper) AcceptVerification(ctx context.Context, txnID id.VerificationTransactionID) error {
	m.AcceptCalled = true
	m.AcceptTxnID = txnID
	if m.AcceptChan != nil {
		m.AcceptChan <- struct{}{}
	}
	return m.ErrToReturn
}

func (m *MockVerificationHelper) StartSAS(ctx context.Context, txnID id.VerificationTransactionID) error {
	m.StartSASCalled = true
	m.StartSASTxnID = txnID
	return m.ErrToReturn
}

func (m *MockVerificationHelper) ConfirmSAS(ctx context.Context, txnID id.VerificationTransactionID) error {
	m.ConfirmSASCalled = true
	m.ConfirmSASTxnID = txnID
	if m.ConfirmChan != nil {
		m.ConfirmChan <- struct{}{}
	}
	return m.ErrToReturn
}

func TestVerificationRequested(t *testing.T) {
	mockHelper := &MockVerificationHelper{
		AcceptChan: make(chan struct{}, 1),
	}
	callbacks := &AcceptAllVerificationCallbacks{
		VerificationHelper: mockHelper,
	}

	txnID := id.VerificationTransactionID("test-txn")
	callbacks.VerificationRequested(context.Background(), txnID, "@user:example.com", "DEVICEID")

	<-mockHelper.AcceptChan

	assert.Equal(t, mockHelper.AcceptCalled, true)
	assert.Equal(t, mockHelper.AcceptTxnID, txnID)
}

func TestVerificationReady_SupportsSAS(t *testing.T) {
	mockHelper := &MockVerificationHelper{}
	callbacks := &AcceptAllVerificationCallbacks{
		VerificationHelper: mockHelper,
	}

	txnID := id.VerificationTransactionID("test-txn")
	callbacks.VerificationReady(context.Background(), txnID, "DEVICEID", true, false, nil)

	assert.Equal(t, mockHelper.StartSASCalled, false)
}

func TestVerificationReady_NoSAS(t *testing.T) {
	mockHelper := &MockVerificationHelper{}
	callbacks := &AcceptAllVerificationCallbacks{
		VerificationHelper: mockHelper,
	}

	txnID := id.VerificationTransactionID("test-txn")
	callbacks.VerificationReady(context.Background(), txnID, "DEVICEID", false, false, nil)

	assert.Equal(t, mockHelper.StartSASCalled, false)
}

func TestShowSAS(t *testing.T) {
	mockHelper := &MockVerificationHelper{
		ConfirmChan: make(chan struct{}, 1),
	}
	callbacks := &AcceptAllVerificationCallbacks{
		VerificationHelper: mockHelper,
	}

	txnID := id.VerificationTransactionID("test-txn")
	emojis := []rune{'🦊', '🐶'}
	descriptions := []string{"Fox", "Dog"}
	decimals := []int{1, 2}

	callbacks.ShowSAS(context.Background(), txnID, emojis, descriptions, decimals)

	<-mockHelper.ConfirmChan

	assert.Equal(t, mockHelper.ConfirmSASCalled, true)
	assert.Equal(t, mockHelper.ConfirmSASTxnID, txnID)
}
