package DavexBase.did.service;

import DavexBase.did.client.DidBackendClient;
import DavexBase.did.client.DidBackendResponse;
import DavexBase.did.config.DidProperties;
import DavexBase.did.context.DidActorContext;
import DavexBase.did.dto.DidApiResponse;
import DavexBase.did.exception.DidClientException;
import com.fasterxml.jackson.databind.JsonNode;
import org.springframework.stereotype.Service;

import java.util.Map;
import java.util.function.Supplier;

@Service
public class DidService {
    public static final int FEATURE_DISABLED = 41001;
    public static final int INVALID_REQUEST = 41002;
    public static final int INVALID_RESPONSE = 41003;
    public static final int UPSTREAM_REJECTED = 41005;

    private final DidProperties properties;
    private final DidBackendClient client;

    public DidService(DidProperties properties, DidBackendClient client) {
        this.properties = properties;
        this.client = client;
    }

    public DidApiResponse<JsonNode> health(DidActorContext context) {
        return forward(context, () -> client.get("/api/health", Map.of(), context, false));
    }

    public DidApiResponse<JsonNode> ready(DidActorContext context) {
        return forward(context, () -> client.get("/api/ready", Map.of(), context, true));
    }

    public DidApiResponse<JsonNode> session(DidActorContext context) {
        return forward(context, () -> client.get("/api/system/session", Map.of(), context, true));
    }

    public DidApiResponse<JsonNode> status(DidActorContext context) {
        return forward(context, () -> client.get("/api/system/status", Map.of(), context, true));
    }

    public DidApiResponse<JsonNode> setGovernance(JsonNode body, DidActorContext context) {
        return forward(context, () -> client.post("/api/system/governance", body, context));
    }

    public DidApiResponse<JsonNode> updateRoles(JsonNode body, DidActorContext context) {
        return forward(context, () -> client.put("/api/system/roles", body, context));
    }

    public DidApiResponse<JsonNode> getRoles(String did, DidActorContext context) {
        return getRequired("did", did, context, "/api/system/roles");
    }

    public DidApiResponse<JsonNode> generateDid(JsonNode body, DidActorContext context) {
        return forward(context, () -> client.post("/api/did/generateDID", body, context));
    }

    public DidApiResponse<JsonNode> registerDid(JsonNode body, DidActorContext context) {
        return forward(context, () -> client.post("/api/did/registerDID", body, context));
    }

    public DidApiResponse<JsonNode> queryDid(String did, DidActorContext context) {
        return getRequired("did", did, context, "/api/did/query");
    }

    public DidApiResponse<JsonNode> registerPolicy(JsonNode body, DidActorContext context) {
        return forward(context, () -> client.post("/api/policy/register", body, context));
    }

    public DidApiResponse<JsonNode> queryPolicy(String policyId, DidActorContext context) {
        return getRequired("policyID", policyId, context, "/api/policy/query");
    }

    public DidApiResponse<JsonNode> deactivatePolicy(JsonNode body, DidActorContext context) {
        return forward(context, () -> client.post("/api/policy/deactivate", body, context));
    }

    public DidApiResponse<JsonNode> issueCredential(JsonNode body, DidActorContext context) {
        return forward(context, () -> client.post("/api/vc/issue", body, context));
    }

    public DidApiResponse<JsonNode> queryCredential(String vcId, DidActorContext context) {
        return getRequired("vcID", vcId, context, "/api/vc/query");
    }

    public DidApiResponse<JsonNode> verifyCredential(JsonNode body, DidActorContext context) {
        return forward(context, () -> client.post("/api/vc/verify", body, context));
    }

    public DidApiResponse<JsonNode> issueNonce(JsonNode body, DidActorContext context) {
        return forward(context, () -> client.post("/api/vp/nonce", body, context));
    }

    public DidApiResponse<JsonNode> generatePresentation(JsonNode body, DidActorContext context) {
        return forward(context, () -> client.post("/api/vp/generate", body, context));
    }

    public DidApiResponse<JsonNode> verifyPresentation(JsonNode body, DidActorContext context) {
        return forward(context, () -> client.post("/api/vp/verify", body, context));
    }

    private DidApiResponse<JsonNode> getRequired(
            String parameter,
            String value,
            DidActorContext context,
            String path) {
        if (value == null || value.isBlank()) {
            return DidApiResponse.failure(
                    INVALID_REQUEST,
                    parameter + " 不能为空",
                    null,
                    context.requestId(),
                    null);
        }
        return forward(context, () -> client.get(path, Map.of(parameter, value.trim()), context, true));
    }

    private DidApiResponse<JsonNode> forward(
            DidActorContext context,
            Supplier<DidBackendResponse> request) {
        if (!properties.isEnabled()) {
            return DidApiResponse.failure(
                    FEATURE_DISABLED,
                    "DID 功能未启用",
                    null,
                    context.requestId(),
                    null);
        }

        try {
            DidBackendResponse response = request.get();
            JsonNode body = response.body();
            JsonNode codeNode = body.get("code");
            if (codeNode == null || codeNode.isNull()) {
                return DidApiResponse.failure(
                        INVALID_RESPONSE,
                        "DID backend 响应缺少 code",
                        null,
                        context.requestId(),
                        null);
            }

            String upstreamCode = codeNode.asText();
            String requestId = textOrDefault(body.get("requestId"), context.requestId());
            String message = textOrDefault(body.get("message"), "success");
            JsonNode data = body.get("data");
            boolean successCode = "OK".equalsIgnoreCase(upstreamCode)
                    || (codeNode.isNumber() && codeNode.asInt(-1) == 0);
            boolean successStatus = response.statusCode() >= 200 && response.statusCode() < 300;

            if (successCode && successStatus) {
                return DidApiResponse.success(data, message, requestId, body.get("tx"), upstreamCode);
            }
            return DidApiResponse.failure(
                    UPSTREAM_REJECTED,
                    message,
                    data,
                    requestId,
                    upstreamCode);
        } catch (DidClientException e) {
            return DidApiResponse.failure(
                    e.getCode(),
                    e.getMessage(),
                    null,
                    context.requestId(),
                    null);
        }
    }

    private static String textOrDefault(JsonNode node, String defaultValue) {
        if (node == null || node.isNull() || node.asText().isBlank()) {
            return defaultValue;
        }
        return node.asText();
    }
}
