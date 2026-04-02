# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability in MXKeys, please report it responsibly:

1. **Do NOT** open a public GitHub issue
2. Send details to: **security@matrix.family** or **@support:matrix.family**
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

We will respond within 48 hours and work with you to:
- Confirm the vulnerability
- Develop a fix
- Coordinate disclosure

## Security Model

MXKeys implements several security measures:

### Cryptographic Verification
- Ed25519 signature verification for all server keys
- Canonical JSON per Matrix specification
- Key length validation (32 bytes for public keys, 64 bytes for signatures)

### Input Validation
- Server name format and length validation
- Key ID format validation (ed25519:*)
- Request body size limits (1MB max)
- Maximum servers per query (configurable, default 100)

### Rate Limiting
- Per-IP token bucket rate limiting
- Separate limits for query endpoint
- Configurable via config.yaml

### Network Security
- TLS 1.2+ required for outbound connections
- Connection timeouts enforced
- Concurrent fetch limits (semaphore)

### DoS Protection
- Circuit breaker for failing upstreams
- Negative caching for resolution/fetch errors
- Request body size limits

## Hardening Recommendations

### Production Deployment

1. **Run as non-root user**
   ```bash
   useradd -r -s /bin/false mxkeys
   chown mxkeys:mxkeys /var/lib/mxkeys
   ```

2. **Use PostgreSQL with SSL**
   ```yaml
   database:
     url: postgres://mxkeys:pass@db/mxkeys?sslmode=verify-full
   ```

3. **Enable JSON logging for SIEM**
   ```yaml
   logging:
     format: json
   ```

4. **Configure firewall**
   ```bash
   ufw allow 8448/tcp
   ```

5. **Use reverse proxy with TLS termination**
   ```nginx
   server {
       listen 443 ssl http2;
       ssl_certificate /etc/letsencrypt/live/mxkeys.example.org/fullchain.pem;
       ssl_certificate_key /etc/letsencrypt/live/mxkeys.example.org/privkey.pem;
       
       location / {
           proxy_pass http://127.0.0.1:8448;
           proxy_set_header X-Real-IP $remote_addr;
           proxy_set_header X-Request-ID $request_id;
       }
   }
   ```

### Docker Deployment

1. **Use read-only root filesystem**
   ```yaml
   security_opt:
     - no-new-privileges:true
   read_only: true
   tmpfs:
     - /tmp
   ```

2. **Drop capabilities**
   ```yaml
   cap_drop:
     - ALL
   ```

3. **Resource limits**
   ```yaml
   deploy:
     resources:
       limits:
         cpus: '1'
         memory: 256M
   ```

## Dependencies

MXKeys uses minimal external dependencies:

| Dependency | Purpose | Security |
|------------|---------|----------|
| github.com/lib/pq | PostgreSQL driver | Maintained, widely used |
| golang.org/x/sync | Singleflight | Official Go package |
| golang.org/x/time | Rate limiter | Official Go package |

All other functionality is implemented internally in `internal/zero/` packages.

## Audit

The codebase is designed for auditability:
- No hidden network calls
- No external service dependencies beyond PostgreSQL
- Deterministic behavior
- Structured logging with request IDs

## License

Apache 2.0 - see LICENSE file.
