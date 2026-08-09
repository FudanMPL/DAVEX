package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"chainmaker.org/chainmaker/contract-sdk-go/v2/pb/protogo"
	"chainmaker.org/chainmaker/contract-sdk-go/v2/sandbox"
	"chainmaker.org/chainmaker/contract-sdk-go/v2/sdk"
	"didcontract/protocol"
)

func main() {
	if err := sandbox.Start(&Contract{service: NewService()}); err != nil {
		sdk.Instance.Errorf("contract start failed: %s", err.Error())
	}
}

type Contract struct {
	service *Service
}

func (c *Contract) InitContract() protogo.Response {
	store := c.service.store
	if exists, err := store.exists(prefixSystem, "config"); err != nil {
		return sdk.Error("read system configuration: " + err.Error())
	} else if exists {
		return sdk.Success([]byte("Gov-DID contract already initialized"))
	}
	origin, err := senderAddress()
	if err != nil {
		return sdk.Error(err.Error())
	}
	if err := store.putJSON(prefixSystem, "config", &SystemConfig{BootstrapAddress: origin}); err != nil {
		return sdk.Error("initialize system configuration: " + err.Error())
	}
	return sdk.Success([]byte("Gov-DID contract initialized"))
}

func (c *Contract) UpgradeContract() protogo.Response {
	return sdk.Success([]byte("Gov-DID contract upgrade accepted; protocol state preserved"))
}

func decodeRequest(out interface{}) error {
	args := sdk.Instance.GetArgs()
	raw, ok := args["request"]
	if !ok || len(raw) == 0 {
		return errors.New("required JSON argument request is missing")
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode request: %w", err)
	}
	return nil
}

func jsonSuccess(value interface{}) protogo.Response {
	data, err := json.Marshal(value)
	if err != nil {
		return sdk.Error("encode response: " + err.Error())
	}
	return sdk.Success(data)
}

func okSuccess() protogo.Response {
	return jsonSuccess(map[string]bool{"success": true})
}

func errorResponse(method string, err error) protogo.Response {
	sdk.Instance.Warnf("method %s failed: %s", method, err.Error())
	return sdk.Error(err.Error())
}

func (c *Contract) InvokeContract(method string) protogo.Response {
	switch method {
	case "RegisterDID":
		var req protocol.RegisterDIDRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		value, err := c.service.RegisterDID(req)
		if err != nil {
			return errorResponse(method, err)
		}
		return jsonSuccess(value)
	case "UpdateDID":
		var req protocol.UpdateDIDRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		value, err := c.service.UpdateDID(req)
		if err != nil {
			return errorResponse(method, err)
		}
		return jsonSuccess(value)
	case "DeactivateDID":
		var req protocol.DeactivateDIDRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		if err := c.service.DeactivateDID(req); err != nil {
			return errorResponse(method, err)
		}
		return okSuccess()
	case "GetDIDRegistry":
		var req protocol.IDRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		value, err := c.service.getDID(req.ID)
		if err != nil {
			return errorResponse(method, err)
		}
		return jsonSuccess(value)
	case "ValidateDID":
		var req protocol.IDRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		value, err := c.service.requireActiveDID(req.ID)
		if err != nil {
			return errorResponse(method, err)
		}
		return jsonSuccess(value)
	case "GetDIDByAddress":
		var req protocol.IDRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		value, err := c.service.store.getString(prefixAddress, req.ID)
		if err != nil {
			return errorResponse(method, err)
		}
		return jsonSuccess(map[string]string{"did": value})

	case "SetGovernanceDID":
		var req protocol.SetGovernanceDIDRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		if err := c.service.SetGovernanceDID(req); err != nil {
			return errorResponse(method, err)
		}
		return okSuccess()
	case "UpdateRoles":
		var req protocol.UpdateRolesRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		value, err := c.service.UpdateRoles(req)
		if err != nil {
			return errorResponse(method, err)
		}
		return jsonSuccess(value)
	case "GetRoles":
		var req protocol.IDRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		var value protocol.RoleBinding
		if err := c.service.store.getJSON(prefixRole, req.ID, &value); err != nil {
			return errorResponse(method, err)
		}
		return jsonSuccess(value)

	case "RegisterPolicy":
		var req protocol.RegisterPolicyRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		value, err := c.service.RegisterPolicy(req)
		if err != nil {
			return errorResponse(method, err)
		}
		return jsonSuccess(value)
	case "DeactivatePolicy":
		var req protocol.DeactivatePolicyRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		value, err := c.service.DeactivatePolicy(req)
		if err != nil {
			return errorResponse(method, err)
		}
		return jsonSuccess(value)
	case "GetPolicy":
		var req protocol.IDRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		value, err := c.service.getPolicy(req.ID)
		if err != nil {
			return errorResponse(method, err)
		}
		return jsonSuccess(value)
	case "VerifyPolicyMembership":
		var req protocol.VerifyPolicyMembershipRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		if err := c.service.VerifyPolicyMembership(req); err != nil {
			return errorResponse(method, err)
		}
		return okSuccess()

	case "IssueVC":
		var req protocol.IssueVCRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		value, err := c.service.IssueVC(req)
		if err != nil {
			return errorResponse(method, err)
		}
		return jsonSuccess(value)
	case "VerifyVC":
		var req protocol.IssueVCRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		now, err := txTime()
		if err != nil {
			return errorResponse(method, err)
		}
		value, err := c.service.validateCredential(req.Credential, req.Proofs, now)
		if err != nil {
			return errorResponse(method, err)
		}
		return jsonSuccess(value)
	case "GetVCProof":
		var req protocol.IDRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		value, err := c.service.getVCProof(req.ID)
		if err != nil {
			return errorResponse(method, err)
		}
		return jsonSuccess(value)
	case "IssueNonce":
		var req protocol.IssueNonceRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		value, err := c.service.IssueNonce(req)
		if err != nil {
			return errorResponse(method, err)
		}
		return jsonSuccess(value)
	case "VerifyVP", "VerifyVPWithPolicy":
		var req protocol.VerifyVPRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		if err := c.service.VerifyVP(req); err != nil {
			return errorResponse(method, err)
		}
		return okSuccess()

	case "CreateCredentialGroup":
		var req protocol.CreateGroupRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		value, err := c.service.CreateCredentialGroup(req)
		if err != nil {
			return errorResponse(method, err)
		}
		return jsonSuccess(value)
	case "AddMemberToGroup":
		var req protocol.AddGroupMemberRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		value, err := c.service.AddMemberToGroup(req)
		if err != nil {
			return errorResponse(method, err)
		}
		return jsonSuccess(value)
	case "GetCredentialGroup":
		var req protocol.GroupRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		value, err := c.service.getGroup(req.GroupID)
		if err != nil {
			return errorResponse(method, err)
		}
		return jsonSuccess(value)
	case "GetGroupPublicKeys":
		var req protocol.GroupRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		value, err := c.service.getGroup(req.GroupID)
		if err != nil {
			return errorResponse(method, err)
		}
		return jsonSuccess(value.PublicKeys)
	case "GetMemberBinding":
		var req protocol.MemberBindingRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		value, err := c.service.getMemberBinding(req.GroupID, req.VCID)
		if err != nil {
			return errorResponse(method, err)
		}
		return jsonSuccess(value)
	case "SuspendGroup":
		var req protocol.GroupRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		value, err := c.service.SetGroupStatus(req.GroupID, protocol.StatusInactive)
		if err != nil {
			return errorResponse(method, err)
		}
		return jsonSuccess(value)
	case "ActivateGroup":
		var req protocol.GroupRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		value, err := c.service.SetGroupStatus(req.GroupID, protocol.StatusActive)
		if err != nil {
			return errorResponse(method, err)
		}
		return jsonSuccess(value)
	case "VerifyPrivacyVP":
		var req protocol.VerifyPrivacyVPRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		if err := c.service.VerifyPrivacyVP(req); err != nil {
			return errorResponse(method, err)
		}
		return okSuccess()
	case "CheckKeyImage":
		var req protocol.IDRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		record, used, err := c.service.CheckKeyImage(req.ID)
		if err != nil {
			return errorResponse(method, err)
		}
		return jsonSuccess(map[string]interface{}{"used": used, "record": record})

	case "RegisterEventIssuer":
		var req protocol.RegisterEventIssuerRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		value, err := c.service.RegisterEventIssuer(req)
		if err != nil {
			return errorResponse(method, err)
		}
		return jsonSuccess(value)
	case "RemoveEventIssuer":
		var req protocol.RegisterEventIssuerRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		value, err := c.service.RemoveEventIssuer(req)
		if err != nil {
			return errorResponse(method, err)
		}
		return jsonSuccess(value)
	case "GetEventIssuer":
		var req protocol.RegisterEventIssuerRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		value, err := c.service.getEventIssuer(req.EventType, req.IssuerDID)
		if err != nil {
			return errorResponse(method, err)
		}
		return jsonSuccess(value)
	case "CreateRevocationCommittee":
		var req protocol.CreateCommitteeRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		value, err := c.service.CreateRevocationCommittee(req)
		if err != nil {
			return errorResponse(method, err)
		}
		return jsonSuccess(value)
	case "AddCommitteeMember":
		var req protocol.CommitteeMemberRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		value, err := c.service.AddCommitteeMember(req)
		if err != nil {
			return errorResponse(method, err)
		}
		return jsonSuccess(value)
	case "RemoveCommitteeMember":
		var req protocol.CommitteeMemberRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		value, err := c.service.RemoveCommitteeMember(req)
		if err != nil {
			return errorResponse(method, err)
		}
		return jsonSuccess(value)
	case "UpdateCommitteeThreshold":
		var req protocol.UpdateThresholdRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		value, err := c.service.UpdateCommitteeThreshold(req)
		if err != nil {
			return errorResponse(method, err)
		}
		return jsonSuccess(value)
	case "GetRevocationCommittee":
		var req protocol.GroupRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		value, err := c.service.getCommittee(req.GroupID)
		if err != nil {
			return errorResponse(method, err)
		}
		return jsonSuccess(value)
	case "ExecuteRevocation":
		var req protocol.ExecuteRevocationRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		value, err := c.service.ExecuteRevocation(req)
		if err != nil {
			return errorResponse(method, err)
		}
		return jsonSuccess(value)
	case "IsCredentialConsumed":
		var req protocol.IDRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		value, err := c.service.IsCredentialConsumed(req.ID)
		if err != nil {
			return errorResponse(method, err)
		}
		return jsonSuccess(map[string]bool{"consumed": value})
	case "GetRevocationLogs":
		var req protocol.IDRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		value, err := c.service.GetRevocationLogs(req.ID)
		if err != nil {
			return errorResponse(method, err)
		}
		return jsonSuccess(value)
	case "GetCommitteeRotationLogs":
		var req protocol.GroupRequest
		if err := decodeRequest(&req); err != nil {
			return errorResponse(method, err)
		}
		value, err := c.service.GetRotationLogs(req.GroupID)
		if err != nil {
			return errorResponse(method, err)
		}
		return jsonSuccess(value)
	default:
		return sdk.Error("unknown contract method: " + method)
	}
}
