package DavexAgent.module.did.controller;

import DavexBase.did.context.DidActorContext;
import DavexBase.did.dto.DidApiResponse;
import DavexBase.did.service.DidService;
import com.fasterxml.jackson.databind.JsonNode;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
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

    private DidActorContext context(String actorAlias, String requestId) {
        return DidActorContext.create(actorAlias, null, requestId);
    }
}
