package client

import "strings"

type Contract struct {
	chain *ChainClient
}

func NewContract(chain *ChainClient) *Contract { return &Contract{chain: chain} }

func (c *Contract) Invoke(method string, request, out interface{}) (*Response, error) {
	response, err := c.chain.CallJSON(method, request, false)
	if err != nil {
		return response, err
	}
	return response, DecodeResult(response, out)
}

func (c *Contract) Query(method string, request, out interface{}) (*Response, error) {
	response, err := c.chain.CallJSON(method, request, true)
	if err != nil {
		return response, err
	}
	return response, DecodeResult(response, out)
}

func (c *Contract) Call(method string, request, out interface{}) (*Response, error) {
	if IsQueryMethod(method) {
		return c.Query(method, request, out)
	}
	return c.Invoke(method, request, out)
}

var queryMethods = map[string]struct{}{
	"GetDIDRegistry": {}, "ValidateDID": {}, "GetDIDByAddress": {}, "GetRoles": {},
	"GetPolicy": {}, "VerifyPolicyMembership": {}, "VerifyVC": {}, "GetVCProof": {},
	"GetCredentialGroup": {}, "GetGroupPublicKeys": {}, "GetMemberBinding": {}, "CheckKeyImage": {},
	"GetEventIssuer": {}, "GetRevocationCommittee": {}, "IsCredentialConsumed": {},
	"GetRevocationLogs": {}, "GetCommitteeRotationLogs": {},
}

func IsQueryMethod(method string) bool {
	_, ok := queryMethods[strings.TrimSpace(method)]
	return ok
}
