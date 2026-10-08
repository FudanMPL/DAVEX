package DavexCenter.module.health.controller;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.time.Instant;
import java.util.Map;

@RestController
@RequestMapping("/api/v1")
public class HealthController {

    @GetMapping("/health")
    public Map<String, Object> health() {
        return Map.of(
                "code", 0,
                "message", "ok",
                "data", Map.of(
                        "service", "davex-center",
                        "time", Instant.now().toString())
        );
    }
}
