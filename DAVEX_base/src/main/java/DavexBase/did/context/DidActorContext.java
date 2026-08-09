package DavexBase.did.context;

import java.util.UUID;

public record DidActorContext(String actorAlias, String did, String requestId) {
    public static DidActorContext create(String actorAlias, String did, String requestId) {
        String normalizedRequestId = requestId == null || requestId.isBlank()
                ? UUID.randomUUID().toString()
                : requestId.trim();
        return new DidActorContext(trimToNull(actorAlias), trimToNull(did), normalizedRequestId);
    }

    private static String trimToNull(String value) {
        if (value == null || value.isBlank()) {
            return null;
        }
        return value.trim();
    }
}
