package DavexBase.did.exception;

public class DidClientException extends RuntimeException {
    private final int code;

    public DidClientException(int code, String message) {
        super(message);
        this.code = code;
    }

    public DidClientException(int code, String message, Throwable cause) {
        super(message, cause);
        this.code = code;
    }

    public int getCode() {
        return code;
    }
}
