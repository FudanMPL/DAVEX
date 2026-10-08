package DavexCenter.module.did.controller;

import DavexBase.did.context.DidActorContext;
import DavexBase.did.dto.DidApiResponse;
import DavexBase.did.service.DidService;
import com.fasterxml.jackson.databind.JsonNode;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api/v1/did")
public class DidController {
    private final DidService didService;

    public DidController(DidService didService) {
        this.didService = didService;
    }

    @GetMapping("/health")
    public DidApiResponse<JsonNode> health(
            @RequestHeader(value = "X-Request-ID", required = false) String requestId) {
        return didService.health(context(null, requestId));
    }

    @GetMapping("/ready")
    public DidApiResponse<JsonNode> ready(
            @RequestHeader(value = "X-DID-Actor", required = false) String actorAlias,
            @RequestHeader(value = "X-Request-ID", required = false) String requestId) {
        return didService.ready(context(actorAlias, requestId));
    }

    @GetMapping("/session")
    public DidApiResponse<JsonNode> session(
            @RequestHeader(value = "X-DID-Actor", required = false) String actorAlias,
            @RequestHeader(value = "X-Request-ID", required = false) String requestId) {
        return didService.session(context(actorAlias, requestId));
    }

    @GetMapping("/status")
    public DidApiResponse<JsonNode> status(
            @RequestHeader(value = "X-DID-Actor", required = false) String actorAlias,
            @RequestHeader(value = "X-Request-ID", required = false) String requestId) {
        return didService.status(context(actorAlias, requestId));
    }

    @PostMapping("/governance")
    public DidApiResponse<JsonNode> setGovernance(
            @RequestBody(required = false) JsonNode body,
            @RequestHeader(value = "X-DID-Actor", required = false) String actorAlias,
            @RequestHeader(value = "X-Request-ID", required = false) String requestId) {
        return didService.setGovernance(body, context(actorAlias, requestId));
    }

    @PutMapping("/roles")
    public DidApiResponse<JsonNode> updateRoles(
            @RequestBody(required = false) JsonNode body,
            @RequestHeader(value = "X-DID-Actor", required = false) String actorAlias,
            @RequestHeader(value = "X-Request-ID", required = false) String requestId) {
        return didService.updateRoles(body, context(actorAlias, requestId));
    }

    @GetMapping("/roles")
    public DidApiResponse<JsonNode> getRoles(
            @RequestParam("did") String did,
            @RequestHeader(value = "X-DID-Actor", required = false) String actorAlias,
            @RequestHeader(value = "X-Request-ID", required = false) String requestId) {
        return didService.getRoles(did, context(actorAlias, requestId));
    }

    @PostMapping("/identity/generate")
    public DidApiResponse<JsonNode> generateDid(
            @RequestBody(required = false) JsonNode body,
            @RequestHeader(value = "X-DID-Actor", required = false) String actorAlias,
            @RequestHeader(value = "X-Request-ID", required = false) String requestId) {
        return didService.generateDid(body, context(actorAlias, requestId));
    }

    @PostMapping("/identity/register")
    public DidApiResponse<JsonNode> registerDid(
            @RequestBody(required = false) JsonNode body,
            @RequestHeader(value = "X-DID-Actor", required = false) String actorAlias,
            @RequestHeader(value = "X-Request-ID", required = false) String requestId) {
        return didService.registerDid(body, context(actorAlias, requestId));
    }

    @GetMapping("/identity/query")
    public DidApiResponse<JsonNode> queryDid(
            @RequestParam("did") String did,
            @RequestHeader(value = "X-DID-Actor", required = false) String actorAlias,
            @RequestHeader(value = "X-Request-ID", required = false) String requestId) {
        return didService.queryDid(did, context(actorAlias, requestId));
    }

    @PostMapping("/policy/register")
    public DidApiResponse<JsonNode> registerPolicy(
            @RequestBody(required = false) JsonNode body,
            @RequestHeader(value = "X-DID-Actor", required = false) String actorAlias,
            @RequestHeader(value = "X-Request-ID", required = false) String requestId) {
        return didService.registerPolicy(body, context(actorAlias, requestId));
    }

    @GetMapping("/policy/query")
    public DidApiResponse<JsonNode> queryPolicy(
            @RequestParam("policyID") String policyId,
            @RequestHeader(value = "X-DID-Actor", required = false) String actorAlias,
            @RequestHeader(value = "X-Request-ID", required = false) String requestId) {
        return didService.queryPolicy(policyId, context(actorAlias, requestId));
    }

    @PostMapping("/policy/deactivate")
    public DidApiResponse<JsonNode> deactivatePolicy(
            @RequestBody(required = false) JsonNode body,
            @RequestHeader(value = "X-DID-Actor", required = false) String actorAlias,
            @RequestHeader(value = "X-Request-ID", required = false) String requestId) {
        return didService.deactivatePolicy(body, context(actorAlias, requestId));
    }

    @PostMapping("/credential/issue")
    public DidApiResponse<JsonNode> issueCredential(
            @RequestBody(required = false) JsonNode body,
            @RequestHeader(value = "X-DID-Actor", required = false) String actorAlias,
            @RequestHeader(value = "X-Request-ID", required = false) String requestId) {
        return didService.issueCredential(body, context(actorAlias, requestId));
    }

    @GetMapping("/credential/query")
    public DidApiResponse<JsonNode> queryCredential(
            @RequestParam("vcID") String vcId,
            @RequestHeader(value = "X-DID-Actor", required = false) String actorAlias,
            @RequestHeader(value = "X-Request-ID", required = false) String requestId) {
        return didService.queryCredential(vcId, context(actorAlias, requestId));
    }

    @PostMapping("/credential/verify")
    public DidApiResponse<JsonNode> verifyCredential(
            @RequestBody(required = false) JsonNode body,
            @RequestHeader(value = "X-DID-Actor", required = false) String actorAlias,
            @RequestHeader(value = "X-Request-ID", required = false) String requestId) {
        return didService.verifyCredential(body, context(actorAlias, requestId));
    }

    @PostMapping("/presentation/nonce")
    public DidApiResponse<JsonNode> issueNonce(
            @RequestBody(required = false) JsonNode body,
            @RequestHeader(value = "X-DID-Actor", required = false) String actorAlias,
            @RequestHeader(value = "X-Request-ID", required = false) String requestId) {
        return didService.issueNonce(body, context(actorAlias, requestId));
    }

    @PostMapping("/presentation/generate")
    public DidApiResponse<JsonNode> generatePresentation(
            @RequestBody(required = false) JsonNode body,
            @RequestHeader(value = "X-DID-Actor", required = false) String actorAlias,
            @RequestHeader(value = "X-Request-ID", required = false) String requestId) {
        return didService.generatePresentation(body, context(actorAlias, requestId));
    }

    @PostMapping("/presentation/verify")
    public DidApiResponse<JsonNode> verifyPresentation(
            @RequestBody(required = false) JsonNode body,
            @RequestHeader(value = "X-DID-Actor", required = false) String actorAlias,
            @RequestHeader(value = "X-Request-ID", required = false) String requestId) {
        return didService.verifyPresentation(body, context(actorAlias, requestId));
    }

    @GetMapping("/privacy/group")
    public DidApiResponse<JsonNode> queryPrivacyGroup(
            @RequestParam("groupID") String groupId,
            @RequestHeader(value = "X-DID-Actor", required = false) String actorAlias,
            @RequestHeader(value = "X-Request-ID", required = false) String requestId) {
        return didService.queryPrivacyGroup(groupId, context(actorAlias, requestId));
    }

    @PostMapping("/privacy/group/member")
    public DidApiResponse<JsonNode> addPrivacyGroupMember(
            @RequestBody(required = false) JsonNode body,
            @RequestHeader(value = "X-DID-Actor", required = false) String actorAlias,
            @RequestHeader(value = "X-Request-ID", required = false) String requestId) {
        return didService.addPrivacyGroupMember(body, context(actorAlias, requestId));
    }

    @PostMapping("/privacy/presentation/generate")
    public DidApiResponse<JsonNode> generatePrivacyPresentation(
            @RequestBody(required = false) JsonNode body,
            @RequestHeader(value = "X-DID-Actor", required = false) String actorAlias,
            @RequestHeader(value = "X-Request-ID", required = false) String requestId) {
        return didService.generatePrivacyPresentation(body, context(actorAlias, requestId));
    }

    @PostMapping("/privacy/presentation/verify")
    public DidApiResponse<JsonNode> verifyPrivacyPresentation(
            @RequestBody(required = false) JsonNode body,
            @RequestHeader(value = "X-DID-Actor", required = false) String actorAlias,
            @RequestHeader(value = "X-Request-ID", required = false) String requestId) {
        return didService.verifyPrivacyPresentation(body, context(actorAlias, requestId));
    }

    @GetMapping("/privacy/keyimage")
    public DidApiResponse<JsonNode> queryKeyImage(
            @RequestParam("value") String value,
            @RequestHeader(value = "X-DID-Actor", required = false) String actorAlias,
            @RequestHeader(value = "X-Request-ID", required = false) String requestId) {
        return didService.queryKeyImage(value, context(actorAlias, requestId));
    }

    @GetMapping("/revocation/issuer")
    public DidApiResponse<JsonNode> queryEventIssuer(
            @RequestParam("eventType") String eventType,
            @RequestParam("issuerDID") String issuerDid,
            @RequestHeader(value = "X-DID-Actor", required = false) String actorAlias,
            @RequestHeader(value = "X-Request-ID", required = false) String requestId) {
        return didService.queryEventIssuer(eventType, issuerDid, context(actorAlias, requestId));
    }

    @GetMapping("/revocation/committee")
    public DidApiResponse<JsonNode> queryRevocationCommittee(
            @RequestParam("groupID") String groupId,
            @RequestHeader(value = "X-DID-Actor", required = false) String actorAlias,
            @RequestHeader(value = "X-Request-ID", required = false) String requestId) {
        return didService.queryRevocationCommittee(groupId, context(actorAlias, requestId));
    }

    @PostMapping("/revocation/requests")
    public DidApiResponse<JsonNode> createRevocationRequest(
            @RequestBody(required = false) JsonNode body,
            @RequestHeader(value = "X-DID-Actor", required = false) String actorAlias,
            @RequestHeader(value = "X-Request-ID", required = false) String requestId) {
        return didService.createRevocationRequest(body, context(actorAlias, requestId));
    }

    @GetMapping("/revocation/requests/{draftID}")
    public DidApiResponse<JsonNode> queryRevocationRequest(
            @PathVariable("draftID") String draftId,
            @RequestHeader(value = "X-DID-Actor", required = false) String actorAlias,
            @RequestHeader(value = "X-Request-ID", required = false) String requestId) {
        return didService.queryRevocationRequest(draftId, context(actorAlias, requestId));
    }

    @PostMapping("/revocation/requests/{draftID}/approvals")
    public DidApiResponse<JsonNode> approveRevocationRequest(
            @PathVariable("draftID") String draftId,
            @RequestHeader(value = "X-DID-Actor", required = false) String actorAlias,
            @RequestHeader(value = "X-Request-ID", required = false) String requestId) {
        return didService.approveRevocationRequest(draftId, context(actorAlias, requestId));
    }

    @PostMapping("/revocation/execute")
    public DidApiResponse<JsonNode> executeRevocation(
            @RequestBody(required = false) JsonNode body,
            @RequestHeader(value = "X-DID-Actor", required = false) String actorAlias,
            @RequestHeader(value = "X-Request-ID", required = false) String requestId) {
        return didService.executeRevocation(body, context(actorAlias, requestId));
    }

    @GetMapping("/revocation/logs")
    public DidApiResponse<JsonNode> queryRevocationLogs(
            @RequestParam("vcID") String vcId,
            @RequestHeader(value = "X-DID-Actor", required = false) String actorAlias,
            @RequestHeader(value = "X-Request-ID", required = false) String requestId) {
        return didService.queryRevocationLogs(vcId, context(actorAlias, requestId));
    }

    @GetMapping("/revocation/consumed")
    public DidApiResponse<JsonNode> queryRevocationConsumed(
            @RequestParam("hash") String hash,
            @RequestHeader(value = "X-DID-Actor", required = false) String actorAlias,
            @RequestHeader(value = "X-Request-ID", required = false) String requestId) {
        return didService.queryRevocationConsumed(hash, context(actorAlias, requestId));
    }

    private DidActorContext context(String actorAlias, String requestId) {
        return DidActorContext.create(actorAlias, null, requestId);
    }
}
