package DavexBase.did.dto;

import com.fasterxml.jackson.databind.JsonNode;
import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@NoArgsConstructor
@AllArgsConstructor
public class DidApiResponse<T> {
    private int code;
    private String message;
    private T data;
    private String requestId;
    private JsonNode tx;
    private String upstreamCode;

    public static <T> DidApiResponse<T> success(
            T data,
            String message,
            String requestId,
            JsonNode tx,
            String upstreamCode) {
        return new DidApiResponse<>(0, message, data, requestId, tx, upstreamCode);
    }

    public static <T> DidApiResponse<T> failure(
            int code,
            String message,
            T data,
            String requestId,
            String upstreamCode) {
        return new DidApiResponse<>(code, message, data, requestId, null, upstreamCode);
    }
}
