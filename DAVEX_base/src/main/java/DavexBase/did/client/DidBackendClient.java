package DavexBase.did.client;

import DavexBase.did.config.DidProperties;
import DavexBase.did.context.DidActorContext;
import DavexBase.did.exception.DidClientException;
import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.springframework.stereotype.Component;

import java.io.IOException;
import java.net.URI;
import java.net.URLEncoder;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.util.LinkedHashMap;
import java.util.Map;

@Component
public class DidBackendClient {
    public static final int INVALID_REQUEST = 41002;
    public static final int INVALID_RESPONSE = 41003;
    public static final int BACKEND_UNAVAILABLE = 41004;

    private final DidProperties properties;
    private final ObjectMapper objectMapper;
    private final HttpClient httpClient;
    private final Duration requestTimeout;

    public DidBackendClient(DidProperties properties, ObjectMapper objectMapper) {
        this.properties = properties;
        this.objectMapper = objectMapper;
        this.requestTimeout = Duration.ofMillis(properties.getRequestTimeoutMs());
        this.httpClient = HttpClient.newBuilder()
                .connectTimeout(Duration.ofMillis(properties.getConnectTimeoutMs()))
                .build();
    }

    public DidBackendResponse get(
            String path,
            Map<String, String> query,
            DidActorContext context,
            boolean authenticated) {
        return send("GET", path, query, null, context, authenticated);
    }

    public DidBackendResponse post(String path, JsonNode body, DidActorContext context) {
        return send("POST", path, Map.of(), body, context, true);
    }

    public DidBackendResponse put(String path, JsonNode body, DidActorContext context) {
        return send("PUT", path, Map.of(), body, context, true);
    }

    private DidBackendResponse send(
            String method,
            String path,
            Map<String, String> query,
            JsonNode body,
            DidActorContext context,
            boolean authenticated) {
        HttpRequest.Builder builder = HttpRequest.newBuilder()
                .uri(buildUri(path, query))
                .timeout(requestTimeout)
                .header("Accept", "application/json")
                .header("X-Request-ID", context.requestId());

        if (authenticated) {
            if (properties.getToken() != null && !properties.getToken().isBlank()) {
                builder.header("Authorization", "Bearer " + properties.getToken().trim());
            }
            if (context.actorAlias() != null) {
                builder.header("X-Actor-Alias", context.actorAlias());
            }
        }

        if ("GET".equals(method)) {
            builder.GET();
        } else {
            builder.header("Content-Type", "application/json; charset=utf-8")
                    .method(method, HttpRequest.BodyPublishers.ofString(serializeBody(body)));
        }

        try {
            HttpResponse<String> response = httpClient.send(
                    builder.build(),
                    HttpResponse.BodyHandlers.ofString(StandardCharsets.UTF_8));
            return new DidBackendResponse(response.statusCode(), parseBody(response.body(), response.statusCode()));
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            throw new DidClientException(BACKEND_UNAVAILABLE, "DID backend 调用被中断", e);
        } catch (IOException e) {
            String detail = e.getMessage() == null || e.getMessage().isBlank()
                    ? e.getClass().getSimpleName()
                    : e.getMessage();
            throw new DidClientException(BACKEND_UNAVAILABLE, "DID backend 不可用: " + detail, e);
        }
    }

    private URI buildUri(String path, Map<String, String> query) {
        if (path == null || !path.startsWith("/")) {
            throw new DidClientException(INVALID_REQUEST, "DID backend path 必须以 / 开头");
        }
        StringBuilder url = new StringBuilder(normalizeBaseUrl(properties.getBackendUrl())).append(path);
        Map<String, String> normalized = new LinkedHashMap<>();
        if (query != null) {
            query.forEach((key, value) -> {
                if (key != null && value != null) {
                    normalized.put(key, value);
                }
            });
        }
        if (!normalized.isEmpty()) {
            url.append('?');
            boolean first = true;
            for (Map.Entry<String, String> entry : normalized.entrySet()) {
                if (!first) {
                    url.append('&');
                }
                first = false;
                url.append(encode(entry.getKey())).append('=').append(encode(entry.getValue()));
            }
        }
        try {
            return URI.create(url.toString());
        } catch (IllegalArgumentException e) {
            throw new DidClientException(INVALID_REQUEST, "DID backend URL 无效", e);
        }
    }

    private String serializeBody(JsonNode body) {
        try {
            return objectMapper.writeValueAsString(body == null ? objectMapper.createObjectNode() : body);
        } catch (JsonProcessingException e) {
            throw new DidClientException(INVALID_REQUEST, "DID 请求 JSON 序列化失败", e);
        }
    }

    private JsonNode parseBody(String body, int statusCode) {
        if (body == null || body.isBlank()) {
            throw new DidClientException(INVALID_RESPONSE, "DID backend 返回空响应, status=" + statusCode);
        }
        try {
            return objectMapper.readTree(body);
        } catch (JsonProcessingException e) {
            throw new DidClientException(INVALID_RESPONSE, "DID backend 返回无效 JSON, status=" + statusCode, e);
        }
    }

    private static String normalizeBaseUrl(String value) {
        String baseUrl = value == null || value.isBlank() ? "http://localhost:8081" : value.trim();
        while (baseUrl.endsWith("/")) {
            baseUrl = baseUrl.substring(0, baseUrl.length() - 1);
        }
        return baseUrl;
    }

    private static String encode(String value) {
        return URLEncoder.encode(value, StandardCharsets.UTF_8);
    }
}
