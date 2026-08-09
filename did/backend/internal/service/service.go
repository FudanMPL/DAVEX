package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"didcontract/backend/internal/chain"
	"didcontract/backend/internal/domain"
	"didcontract/backend/internal/store"
	"didcontract/sdk-go/client"
)

const timeSlotSeconds int64 = 300

type Service struct {
	gateway chain.Gateway
	store   *store.Repository
	now     func() time.Time
}

type Result struct {
	Data interface{}
	Tx   *domain.Transaction
}

func New(gateway chain.Gateway, repository *store.Repository) *Service {
	return &Service{gateway: gateway, store: repository, now: time.Now}
}

func (s *Service) actor(alias string) (domain.Actor, error) {
	actor, err := s.gateway.Actor(alias)
	if err != nil {
		return domain.Actor{}, domain.WrapError("ACTOR_NOT_FOUND", http.StatusUnauthorized, "调用主体不存在", err)
	}
	return actor, nil
}

func requireActorDID(actor domain.Actor, did string) error {
	did = strings.TrimSpace(did)
	if did == "" {
		return domain.NewError("DID_REQUIRED", http.StatusBadRequest, "DID 不能为空")
	}
	if actor.DID != "" && actor.DID != did {
		return domain.NewError("ACTOR_DID_MISMATCH", http.StatusForbidden, "当前登录主体不能代表该 DID")
	}
	return nil
}

func (s *Service) call(ctx context.Context, actorAlias, method string, request, out interface{}) (Result, error) {
	response, err := s.gateway.Call(ctx, actorAlias, method, request, out)
	if err != nil {
		return Result{}, mapChainError(err)
	}
	return Result{Data: out, Tx: transaction(response)}, nil
}

func transaction(response *client.Response) *domain.Transaction {
	if response == nil || response.TxID == "" {
		return nil
	}
	return &domain.Transaction{TxID: response.TxID, BlockHeight: response.BlockHeight, GasUsed: response.GasUsed}
}

func mapChainError(err error) error {
	var callError *client.CallError
	if errors.As(err, &callError) {
		message := "链上业务校验失败"
		if callError.Response != nil && strings.TrimSpace(callError.Response.Message) != "" {
			message = callError.Response.Message
		}
		status := http.StatusConflict
		lower := strings.ToLower(message)
		if strings.Contains(lower, "not found") || strings.Contains(message, "不存在") {
			status = http.StatusNotFound
		} else if strings.Contains(lower, "sender") || strings.Contains(lower, "role") || strings.Contains(lower, "governance") {
			status = http.StatusForbidden
		}
		return domain.WrapError("CONTRACT_REJECTED", status, message, err)
	}
	return domain.WrapError("CHAIN_UNAVAILABLE", http.StatusBadGateway, "区块链调用失败", err)
}

func badRequest(message string) error {
	return domain.NewError("INVALID_REQUEST", http.StatusBadRequest, message)
}
func notFound(message string, err error) error {
	return domain.WrapError("NOT_FOUND", http.StatusNotFound, message, err)
}
func conflict(message string, err error) error {
	return domain.WrapError("CONFLICT", http.StatusConflict, message, err)
}

func randomID(prefix string) (string, error) {
	value := make([]byte, 12)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return prefix + "-" + hex.EncodeToString(value), nil
}

func (s *Service) Ready(ctx context.Context) error { return s.gateway.Ready(ctx) }
func (s *Service) SystemStatus(ctx context.Context) (chain.Status, error) {
	return s.gateway.Status(ctx)
}
func (s *Service) Actor(alias string) (domain.Actor, error)      { return s.gateway.Actor(alias) }
func (s *Service) Audits(limit int) ([]domain.AuditEntry, error) { return s.store.Audits(limit) }
func (s *Service) AppendAudit(entry domain.AuditEntry)           { _ = s.store.AppendAudit(entry) }
