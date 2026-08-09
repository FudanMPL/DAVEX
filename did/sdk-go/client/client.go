package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	cmcrypto "chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	chainsdk "chainmaker.org/chainmaker/sdk-go/v2"
	cmutils "chainmaker.org/chainmaker/utils/v2"
)

const requestParameter = "request"

type Config struct {
	SDKConfigPath string
	ContractName  string
	Timeout       int64
	DisableSDKLog bool
}

type Response struct {
	Success     bool            `json:"success"`
	TxID        string          `json:"txID,omitempty"`
	BlockHeight uint64          `json:"blockHeight,omitempty"`
	Code        int32           `json:"code"`
	Message     string          `json:"message,omitempty"`
	GasUsed     uint64          `json:"gasUsed,omitempty"`
	Result      json.RawMessage `json:"result,omitempty"`
}

type CallError struct {
	Method   string
	Response *Response
}

func (e *CallError) Error() string {
	if e.Response == nil {
		return "contract call failed"
	}
	message := strings.TrimSpace(e.Response.Message)
	if message == "" {
		message = "unknown chain or contract error"
	}
	return fmt.Sprintf("contract %s failed (code=%d): %s", e.Method, e.Response.Code, message)
}

type ChainClient struct {
	sdk          *chainsdk.ChainClient
	contractName string
	timeout      int64
}

type ChainMetadata struct {
	ChainID       string `json:"chainID"`
	Version       string `json:"version"`
	AuthType      string `json:"authType"`
	HashAlgorithm string `json:"hashAlgorithm"`
	AddressType   string `json:"addressType"`
}

func New(config Config) (*ChainClient, error) {
	if strings.TrimSpace(config.SDKConfigPath) == "" {
		return nil, errors.New("SDK config path is required")
	}
	if strings.TrimSpace(config.ContractName) == "" {
		return nil, errors.New("contract name is required")
	}
	options := []chainsdk.ChainClientOption{chainsdk.WithConfPath(config.SDKConfigPath)}
	if config.DisableSDKLog {
		options = append(options, chainsdk.WithChainClientLogger(discardLogger{}))
	}
	sdkClient, err := chainsdk.NewChainClient(options...)
	if err != nil {
		return nil, fmt.Errorf("create ChainMaker client: %w", err)
	}
	return &ChainClient{sdk: sdkClient, contractName: config.ContractName, timeout: config.Timeout}, nil
}

// discardLogger prevents the upstream SDK debug logger from persisting full
// contract parameters such as VC and VP payloads in backend deployments.
type discardLogger struct{}

func (discardLogger) Debugf(string, ...interface{}) {}
func (discardLogger) Infof(string, ...interface{})  {}
func (discardLogger) Warnf(string, ...interface{})  {}
func (discardLogger) Errorf(string, ...interface{}) {}
func (discardLogger) Debug(...interface{})          {}
func (discardLogger) Info(...interface{})           {}
func (discardLogger) Warn(...interface{})           {}
func (discardLogger) Error(...interface{})          {}

func (c *ChainClient) Close() {
	if c != nil && c.sdk != nil {
		c.sdk.Stop()
	}
}

func (c *ChainClient) ContractName() string { return c.contractName }

func (c *ChainClient) OriginAddress() (string, error) {
	if c == nil || c.sdk == nil {
		return "", errors.New("ChainMaker client is not initialized")
	}
	chainConfig, err := c.sdk.GetChainConfig()
	if err != nil {
		return "", fmt.Errorf("query address type from ChainMaker configuration: %w", err)
	}
	addressType := config.AddrType_CHAINMAKER
	if chainConfig.GetVm() != nil {
		addressType = chainConfig.GetVm().GetAddrType()
	}
	hashName := ""
	if chainConfig.GetCrypto() != nil {
		hashName = chainConfig.GetCrypto().GetHash()
	}
	return deriveOriginAddress(c.sdk.GetCertPEM(), c.sdk.GetPublicKey(), addressType, hashName)
}

func deriveOriginAddress(certPEM []byte, publicKey cmcrypto.PublicKey, addressType config.AddrType, hashName string) (string, error) {
	switch addressType {
	case config.AddrType_CHAINMAKER, config.AddrType_ZXL, config.AddrType_ETHEREUM:
	default:
		return "", fmt.Errorf("unsupported ChainMaker address type %d", addressType)
	}
	if len(certPEM) > 0 {
		certificate, err := cmutils.ParseCert(certPEM)
		if err != nil {
			return "", fmt.Errorf("parse SDK signing certificate: %w", err)
		}
		address, err := cmutils.CertToAddrStr(certificate, addressType)
		if err != nil {
			return "", fmt.Errorf("derive %s caller address from SDK certificate: %w", addressType.String(), err)
		}
		return address, nil
	}
	if publicKey == nil {
		return "", errors.New("SDK identity has neither certificate nor public key")
	}
	hashType := cmcrypto.HASH_TYPE_SHA256
	if addressType == config.AddrType_CHAINMAKER {
		var ok bool
		hashType, ok = cmcrypto.HashAlgoMap[strings.ToUpper(strings.TrimSpace(hashName))]
		if !ok {
			return "", fmt.Errorf("unsupported ChainMaker hash algorithm %q", hashName)
		}
	}
	address, err := cmutils.PkToAddrStr(publicKey, addressType, hashType)
	if err != nil {
		return "", fmt.Errorf("derive %s caller address from SDK public key: %w", addressType.String(), err)
	}
	return address, nil
}

func (c *ChainClient) Ping() error {
	if c == nil || c.sdk == nil {
		return errors.New("ChainMaker client is not initialized")
	}
	if _, err := c.sdk.GetChainConfig(); err != nil {
		return fmt.Errorf("query ChainMaker chain configuration: %w", err)
	}
	return nil
}

func (c *ChainClient) Metadata() (ChainMetadata, error) {
	if c == nil || c.sdk == nil {
		return ChainMetadata{}, errors.New("ChainMaker client is not initialized")
	}
	chainConfig, err := c.sdk.GetChainConfig()
	if err != nil {
		return ChainMetadata{}, fmt.Errorf("query ChainMaker chain configuration: %w", err)
	}
	metadata := ChainMetadata{
		ChainID: chainConfig.GetChainId(), Version: chainConfig.GetVersion(), AuthType: chainConfig.GetAuthType(),
	}
	if chainConfig.GetCrypto() != nil {
		metadata.HashAlgorithm = chainConfig.GetCrypto().GetHash()
	}
	addressType := config.AddrType_CHAINMAKER
	if chainConfig.GetVm() != nil {
		addressType = chainConfig.GetVm().GetAddrType()
	}
	metadata.AddressType = addressType.String()
	return metadata, nil
}

func (c *ChainClient) ContractInfo() (*common.Contract, error) {
	if c == nil || c.sdk == nil {
		return nil, errors.New("ChainMaker client is not initialized")
	}
	contract, err := c.sdk.GetContractInfo(c.contractName)
	if err != nil {
		return nil, fmt.Errorf("query contract %s: %w", c.contractName, err)
	}
	return contract, nil
}

func (c *ChainClient) ContractNames() ([]string, error) {
	contracts, err := c.sdk.GetContractList()
	if err != nil {
		return nil, fmt.Errorf("query ChainMaker contract list: %w", err)
	}
	names := make([]string, 0, len(contracts))
	for _, contract := range contracts {
		if contract != nil && contract.Name != "" {
			names = append(names, contract.Name)
		}
	}
	return names, nil
}

func (c *ChainClient) CallJSON(method string, request interface{}, query bool) (*Response, error) {
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("encode %s request: %w", method, err)
	}
	params := []*common.KeyValuePair{{Key: requestParameter, Value: requestBytes}}
	var txResponse *common.TxResponse
	if query {
		txResponse, err = c.sdk.QueryContract(c.contractName, method, params, c.timeout)
	} else {
		txResponse, err = c.sdk.InvokeContract(c.contractName, method, "", params, c.timeout, true)
	}
	if err != nil {
		return nil, fmt.Errorf("submit %s to ChainMaker: %w", method, err)
	}
	response := normalizeResponse(txResponse)
	if !response.Success {
		return response, &CallError{Method: method, Response: response}
	}
	return response, nil
}

func (c *ChainClient) CallRaw(method string, requestJSON []byte, query bool) (*Response, error) {
	var request interface{}
	if err := json.Unmarshal(requestJSON, &request); err != nil {
		return nil, fmt.Errorf("request is not valid JSON: %w", err)
	}
	return c.CallJSON(method, request, query)
}

func normalizeResponse(txResponse *common.TxResponse) *Response {
	if txResponse == nil {
		return &Response{Message: "empty ChainMaker response"}
	}
	response := &Response{
		TxID:        txResponse.TxId,
		BlockHeight: txResponse.TxBlockHeight,
		Code:        int32(txResponse.Code),
		Message:     txResponse.Message,
	}
	if txResponse.Code != common.TxStatusCode_SUCCESS {
		return response
	}
	if txResponse.ContractResult == nil {
		response.Message = "ChainMaker response has no contract result"
		return response
	}
	result := txResponse.ContractResult
	response.GasUsed = result.GasUsed
	response.Result = append(json.RawMessage(nil), result.Result...)
	if result.Code != 0 {
		response.Code = int32(result.Code)
		response.Message = result.Message
		return response
	}
	response.Success = true
	return response
}

func DecodeResult(response *Response, out interface{}) error {
	if response == nil || !response.Success {
		return errors.New("cannot decode an unsuccessful response")
	}
	if out == nil || len(response.Result) == 0 {
		return nil
	}
	if err := json.Unmarshal(response.Result, out); err != nil {
		return fmt.Errorf("decode contract result: %w", err)
	}
	return nil
}
