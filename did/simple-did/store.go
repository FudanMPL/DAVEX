package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"chainmaker.org/chainmaker/contract-sdk-go/v2/sdk"
)

var errNotFound = errors.New("state not found")

type Store struct{}

func stateKey(value string) string {
	h := sha256.Sum256([]byte(value))
	return hex.EncodeToString(h[:])
}

func (s *Store) putJSON(prefix, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal %s state: %w", prefix, err)
	}
	return sdk.Instance.PutStateByte(prefix, stateKey(key), data)
}

func (s *Store) getJSON(prefix, key string, out interface{}) error {
	data, err := sdk.Instance.GetStateByte(prefix, stateKey(key))
	if err != nil {
		return fmt.Errorf("read %s state: %w", prefix, err)
	}
	if len(data) == 0 {
		return errNotFound
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decode %s state: %w", prefix, err)
	}
	return nil
}

func (s *Store) exists(prefix, key string) (bool, error) {
	data, err := sdk.Instance.GetStateByte(prefix, stateKey(key))
	if err != nil {
		return false, err
	}
	return len(data) != 0, nil
}

func (s *Store) putString(prefix, key, value string) error {
	return sdk.Instance.PutState(prefix, stateKey(key), value)
}

func (s *Store) getString(prefix, key string) (string, error) {
	value, err := sdk.Instance.GetState(prefix, stateKey(key))
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", errNotFound
	}
	return value, nil
}

func (s *Store) appendIndex(prefix, key, value string) error {
	var values []string
	err := s.getJSON(prefix, key, &values)
	if err != nil && !errors.Is(err, errNotFound) {
		return err
	}
	for _, existing := range values {
		if existing == value {
			return nil
		}
	}
	values = append(values, value)
	return s.putJSON(prefix, key, values)
}

func (s *Store) getIndex(prefix, key string) ([]string, error) {
	var values []string
	err := s.getJSON(prefix, key, &values)
	if errors.Is(err, errNotFound) {
		return []string{}, nil
	}
	return values, err
}

func txTime() (int64, error) {
	value, err := sdk.Instance.GetTxTimeStamp()
	if err != nil {
		return 0, err
	}
	timestamp, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid transaction timestamp: %w", err)
	}
	return timestamp, nil
}

func txID() string {
	value, _ := sdk.Instance.GetTxId()
	return value
}

func senderAddress() (string, error) {
	value, err := sdk.Instance.Origin()
	if err != nil {
		return "", fmt.Errorf("resolve transaction origin: %w", err)
	}
	return value, nil
}
