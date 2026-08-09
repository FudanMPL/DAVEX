package httpapi

import (
	"net/http"
	"strconv"

	"didcontract/backend/internal/domain"
	"didcontract/backend/internal/service"
	"didcontract/protocol"
	"github.com/gin-gonic/gin"
)

func handle(c *gin.Context, result service.Result, err error, status int) {
	if err != nil {
		failure(c, err)
		return
	}
	success(c, status, result)
}

func (a *API) ready(c *gin.Context) {
	err := a.service.Ready(c.Request.Context())
	if err != nil {
		failure(c, domain.WrapError("NOT_READY", http.StatusServiceUnavailable, "链或合约未就绪", err))
		return
	}
	success(c, http.StatusOK, service.Result{Data: map[string]string{"status": "ready"}})
}
func (a *API) systemStatus(c *gin.Context) {
	value, err := a.service.SystemStatus(c.Request.Context())
	handle(c, service.Result{Data: value}, err, http.StatusOK)
}
func (a *API) sessionInfo(c *gin.Context) {
	status, err := a.service.SystemStatus(c.Request.Context())
	if err != nil {
		failure(c, domain.WrapError("STATUS_UNAVAILABLE", http.StatusBadGateway, "无法读取当前主体信息", err))
		return
	}
	actor, err := a.service.Actor(actorAlias(c))
	if err != nil {
		failure(c, domain.WrapError("ACTOR_NOT_FOUND", http.StatusUnauthorized, "当前执行主体不存在", err))
		return
	}
	success(c, http.StatusOK, service.Result{Data: map[string]interface{}{
		"loginAlias": loginActorAlias(c), "admin": sessionAdmin(c), "executionAlias": actorAlias(c),
		"actor": actor, "actors": status.Actors,
	}})
}
func (a *API) audits(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	value, err := a.service.Audits(limit)
	handle(c, service.Result{Data: value}, err, http.StatusOK)
}

func (a *API) setGovernance(c *gin.Context) {
	var input protocol.SetGovernanceDIDRequest
	if !bind(c, &input) {
		return
	}
	result, err := a.service.SetGovernance(c.Request.Context(), actorAlias(c), input.DID)
	handle(c, result, err, http.StatusOK)
}
func (a *API) updateRoles(c *gin.Context) {
	var input protocol.UpdateRolesRequest
	if !bind(c, &input) {
		return
	}
	result, err := a.service.UpdateRoles(c.Request.Context(), actorAlias(c), input)
	handle(c, result, err, http.StatusOK)
}
func (a *API) getRoles(c *gin.Context) {
	result, err := a.service.GetRoles(c.Request.Context(), actorAlias(c), c.Query("did"))
	handle(c, result, err, http.StatusOK)
}

func (a *API) generateDID(c *gin.Context) {
	var input service.GenerateDIDInput
	if !bind(c, &input) {
		return
	}
	result, err := a.service.GenerateDID(actorAlias(c), input)
	handle(c, result, err, http.StatusCreated)
}
func (a *API) registerDID(c *gin.Context) {
	var input service.RegisterDIDInput
	if !bind(c, &input) {
		return
	}
	result, err := a.service.RegisterDID(c.Request.Context(), actorAlias(c), input)
	handle(c, result, err, http.StatusCreated)
}
func (a *API) queryDID(c *gin.Context) {
	result, err := a.service.QueryDID(c.Request.Context(), actorAlias(c), c.Query("did"))
	handle(c, result, err, http.StatusOK)
}
func (a *API) listDIDs(c *gin.Context) { success(c, http.StatusOK, a.service.ListDIDs()) }
func (a *API) rotateDID(c *gin.Context) {
	var input service.RotateDIDInput
	if !bind(c, &input) {
		return
	}
	result, err := a.service.RotateDID(c.Request.Context(), actorAlias(c), input)
	handle(c, result, err, http.StatusOK)
}
func (a *API) deactivateDID(c *gin.Context) {
	result, err := a.service.DeactivateDID(c.Request.Context(), actorAlias(c), c.Query("did"))
	handle(c, result, err, http.StatusOK)
}

func (a *API) registerPolicy(c *gin.Context) {
	var input service.PolicyInput
	if !bind(c, &input) {
		return
	}
	result, err := a.service.RegisterPolicy(c.Request.Context(), actorAlias(c), input)
	handle(c, result, err, http.StatusCreated)
}
func (a *API) queryPolicy(c *gin.Context) {
	result, err := a.service.QueryPolicy(c.Request.Context(), actorAlias(c), c.Query("policyID"))
	handle(c, result, err, http.StatusOK)
}
func (a *API) deactivatePolicy(c *gin.Context) {
	var input protocol.IDRequest
	if !bind(c, &input) {
		return
	}
	result, err := a.service.DeactivatePolicy(c.Request.Context(), actorAlias(c), input.ID)
	handle(c, result, err, http.StatusOK)
}

func (a *API) issueVC(c *gin.Context) {
	var input service.IssueVCInput
	if !bind(c, &input) {
		return
	}
	result, err := a.service.IssueVC(c.Request.Context(), actorAlias(c), input)
	handle(c, result, err, http.StatusCreated)
}
func (a *API) queryVC(c *gin.Context) {
	result, err := a.service.QueryVC(c.Request.Context(), actorAlias(c), c.Query("vcID"))
	handle(c, result, err, http.StatusOK)
}
func (a *API) verifyVC(c *gin.Context) {
	var input protocol.IDRequest
	if !bind(c, &input) {
		return
	}
	result, err := a.service.VerifyVC(c.Request.Context(), actorAlias(c), input.ID)
	handle(c, result, err, http.StatusOK)
}

func (a *API) issueNonce(c *gin.Context) {
	var input service.IssueNonceInput
	if !bind(c, &input) {
		return
	}
	result, err := a.service.IssueNonce(c.Request.Context(), actorAlias(c), input)
	handle(c, result, err, http.StatusCreated)
}
func (a *API) generateVP(c *gin.Context) {
	var input service.GenerateVPInput
	if !bind(c, &input) {
		return
	}
	result, err := a.service.GenerateVP(actorAlias(c), input)
	handle(c, result, err, http.StatusOK)
}
func (a *API) verifyVP(c *gin.Context) {
	var input service.VerifyVPInput
	if !bind(c, &input) {
		return
	}
	result, err := a.service.VerifyVP(c.Request.Context(), actorAlias(c), input)
	handle(c, result, err, http.StatusOK)
}
func (a *API) verifyPolicy(c *gin.Context) {
	var input protocol.VerifyPolicyMembershipRequest
	if !bind(c, &input) {
		return
	}
	result, err := a.service.VerifyPolicy(c.Request.Context(), actorAlias(c), input)
	handle(c, result, err, http.StatusOK)
}

func (a *API) createGroup(c *gin.Context) {
	var input service.CreateGroupInput
	if !bind(c, &input) {
		return
	}
	result, err := a.service.CreateGroup(c.Request.Context(), actorAlias(c), input)
	handle(c, result, err, http.StatusCreated)
}
func (a *API) addGroupMember(c *gin.Context) {
	var input service.AddGroupMemberInput
	if !bind(c, &input) {
		return
	}
	result, err := a.service.AddGroupMember(c.Request.Context(), actorAlias(c), input)
	handle(c, result, err, http.StatusCreated)
}
func (a *API) queryGroup(c *gin.Context) {
	result, err := a.service.QueryGroup(c.Request.Context(), actorAlias(c), c.Query("groupID"))
	handle(c, result, err, http.StatusOK)
}
func (a *API) setGroupStatus(c *gin.Context) {
	var input service.GroupStatusInput
	if !bind(c, &input) {
		return
	}
	result, err := a.service.SetGroupStatus(c.Request.Context(), actorAlias(c), input)
	handle(c, result, err, http.StatusOK)
}
func (a *API) generatePrivacyVP(c *gin.Context) {
	var input service.GeneratePrivacyVPInput
	if !bind(c, &input) {
		return
	}
	result, err := a.service.GeneratePrivacyVP(c.Request.Context(), actorAlias(c), input)
	handle(c, result, err, http.StatusOK)
}
func (a *API) verifyPrivacyVP(c *gin.Context) {
	var input protocol.VerifyPrivacyVPRequest
	if !bind(c, &input) {
		return
	}
	result, err := a.service.VerifyPrivacyVP(c.Request.Context(), actorAlias(c), input)
	handle(c, result, err, http.StatusOK)
}
func (a *API) computeKeyImage(c *gin.Context) {
	var input service.KeyImageInput
	if !bind(c, &input) {
		return
	}
	result, err := a.service.ComputeKeyImage(c.Request.Context(), actorAlias(c), input)
	handle(c, result, err, http.StatusOK)
}
func (a *API) checkKeyImage(c *gin.Context) {
	result, err := a.service.CheckKeyImage(c.Request.Context(), actorAlias(c), c.Query("value"))
	handle(c, result, err, http.StatusOK)
}

func (a *API) registerEventIssuer(c *gin.Context) {
	var input protocol.RegisterEventIssuerRequest
	if !bind(c, &input) {
		return
	}
	result, err := a.service.RegisterEventIssuer(c.Request.Context(), actorAlias(c), input)
	handle(c, result, err, http.StatusCreated)
}
func (a *API) removeEventIssuer(c *gin.Context) {
	var input protocol.RegisterEventIssuerRequest
	if !bind(c, &input) {
		return
	}
	result, err := a.service.RemoveEventIssuer(c.Request.Context(), actorAlias(c), input)
	handle(c, result, err, http.StatusOK)
}
func (a *API) getEventIssuer(c *gin.Context) {
	input := protocol.RegisterEventIssuerRequest{EventType: c.Query("eventType"), IssuerDID: c.Query("issuerDID")}
	result, err := a.service.GetEventIssuer(c.Request.Context(), actorAlias(c), input)
	handle(c, result, err, http.StatusOK)
}
func (a *API) createCommittee(c *gin.Context) {
	var input service.CreateCommitteeInput
	if !bind(c, &input) {
		return
	}
	result, err := a.service.CreateCommittee(c.Request.Context(), actorAlias(c), input)
	handle(c, result, err, http.StatusCreated)
}
func (a *API) addCommitteeMember(c *gin.Context) {
	var input service.CommitteeMemberInput
	if !bind(c, &input) {
		return
	}
	result, err := a.service.AddCommitteeMember(c.Request.Context(), actorAlias(c), input)
	handle(c, result, err, http.StatusOK)
}
func (a *API) removeCommitteeMember(c *gin.Context) {
	var input service.CommitteeMemberInput
	if !bind(c, &input) {
		return
	}
	result, err := a.service.RemoveCommitteeMember(c.Request.Context(), actorAlias(c), input)
	handle(c, result, err, http.StatusOK)
}
func (a *API) queryCommittee(c *gin.Context) {
	result, err := a.service.QueryCommittee(c.Request.Context(), actorAlias(c), c.Query("groupID"))
	handle(c, result, err, http.StatusOK)
}
func (a *API) rotationLogs(c *gin.Context) {
	result, err := a.service.QueryRotationLogs(c.Request.Context(), actorAlias(c), c.Query("groupID"))
	handle(c, result, err, http.StatusOK)
}
func (a *API) updateThreshold(c *gin.Context) {
	var input protocol.UpdateThresholdRequest
	if !bind(c, &input) {
		return
	}
	result, err := a.service.UpdateCommitteeThreshold(c.Request.Context(), actorAlias(c), input)
	handle(c, result, err, http.StatusOK)
}
func (a *API) createRevocationDraft(c *gin.Context) {
	var input service.CreateRevocationInput
	if !bind(c, &input) {
		return
	}
	result, err := a.service.CreateRevocationDraft(c.Request.Context(), actorAlias(c), input)
	handle(c, result, err, http.StatusCreated)
}
func (a *API) getRevocationDraft(c *gin.Context) {
	result, err := a.service.GetRevocationDraft(c.Param("id"))
	handle(c, result, err, http.StatusOK)
}
func (a *API) approveRevocation(c *gin.Context) {
	result, err := a.service.ApproveRevocation(c.Request.Context(), actorAlias(c), c.Param("id"))
	handle(c, result, err, http.StatusOK)
}
func (a *API) executeRevocation(c *gin.Context) {
	var input service.DraftInput
	if !bind(c, &input) {
		return
	}
	result, err := a.service.ExecuteRevocation(c.Request.Context(), actorAlias(c), input.DraftID)
	handle(c, result, err, http.StatusOK)
}
func (a *API) revocationLogs(c *gin.Context) {
	result, err := a.service.QueryRevocationLogs(c.Request.Context(), actorAlias(c), c.Query("vcID"))
	handle(c, result, err, http.StatusOK)
}
func (a *API) credentialConsumed(c *gin.Context) {
	result, err := a.service.IsCredentialConsumed(c.Request.Context(), actorAlias(c), c.Query("hash"))
	handle(c, result, err, http.StatusOK)
}
