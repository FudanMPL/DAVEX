package domain

import (
	"fmt"
	"time"

	"didcontract/protocol"
)

type Actor struct {
	Alias  string `json:"alias"`
	DID    string `json:"did,omitempty"`
	Origin string `json:"origin"`
}

type Transaction struct {
	TxID        string `json:"txId,omitempty"`
	BlockHeight uint64 `json:"blockHeight,omitempty"`
	GasUsed     uint64 `json:"gasUsed,omitempty"`
}

type VerificationSession struct {
	Nonce       string `json:"nonce"`
	VerifierDID string `json:"verifierDID"`
	Purpose     string `json:"purpose"`
	ExpiresAt   int64  `json:"expiresAt"`
	Status      string `json:"status"`
	CreatedAt   int64  `json:"createdAt"`
}

type RevocationDraft struct {
	ID              string                               `json:"id"`
	Credential      protocol.RevocationCredential        `json:"credential"`
	GroupID         string                               `json:"groupID"`
	HolderDID       string                               `json:"holderDID"`
	HolderPublicKey string                               `json:"holderPublicKey"`
	VCID            string                               `json:"vcID"`
	Nonce           string                               `json:"nonce"`
	Approvals       map[string]protocol.PartialSignature `json:"approvals"`
	Status          string                               `json:"status"`
	CreatedAt       int64                                `json:"createdAt"`
	UpdatedAt       int64                                `json:"updatedAt"`
	ExecutedTxID    string                               `json:"executedTxID,omitempty"`
}

type AuditEntry struct {
	Time       int64  `json:"time"`
	RequestID  string `json:"requestId"`
	Actor      string `json:"actor"`
	LoginActor string `json:"loginActor,omitempty"`
	DID        string `json:"did,omitempty"`
	Action     string `json:"action"`
	Success    bool   `json:"success"`
	TxID       string `json:"txId,omitempty"`
	ErrorCode  string `json:"errorCode,omitempty"`
}

type Error struct {
	Code    string
	Status  int
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Cause)
}

func (e *Error) Unwrap() error { return e.Cause }

func NewError(code string, status int, message string) *Error {
	return &Error{Code: code, Status: status, Message: message}
}

func WrapError(code string, status int, message string, cause error) *Error {
	return &Error{Code: code, Status: status, Message: message, Cause: cause}
}

func NewDraftID(now time.Time, suffix string) string {
	return fmt.Sprintf("revoke-%d-%s", now.UnixNano(), suffix)
}
