package DavexBase.did.client;

import com.fasterxml.jackson.databind.JsonNode;

public record DidBackendResponse(int statusCode, JsonNode body) {
}
