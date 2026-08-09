package DavexBase.did.config;

import lombok.Data;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

@Data
@Component
public class DidProperties {
    @Value("${did.enabled:${DID_ENABLED:false}}")
    private boolean enabled;

    @Value("${did.backend-url:${DID_BACKEND_URL:http://localhost:8081}}")
    private String backendUrl;

    @Value("${did.connect-timeout-ms:${DID_CONNECT_TIMEOUT_MS:2000}}")
    private long connectTimeoutMs;

    @Value("${did.request-timeout-ms:${DID_REQUEST_TIMEOUT_MS:5000}}")
    private long requestTimeoutMs;

    @Value("${did.token:${DID_BACKEND_TOKEN:}}")
    private String token;
}
