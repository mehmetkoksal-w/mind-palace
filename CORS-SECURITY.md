# WebSocket CORS Security Implementation

## Overview

This document describes the implementation of WebSocket CORS (Cross-Origin Resource Sharing) security for the Mind Palace dashboard.

## Security Issue Addressed

**Previous Implementation:** The dashboard allowed WebSocket connections from ANY origin (`*`), creating a security vulnerability where malicious websites could connect to the local dashboard server.

**Current Implementation:** Environment-aware CORS with strict origin checking based on configuration.

## Configuration

### Development Mode (Default)

When no CORS configuration is provided in `palace.jsonc`, the dashboard defaults to development mode with these allowed origins:

- `http://localhost:4200` (Angular dev server)
- `http://localhost:3000` (React/Next.js dev server)
- `http://127.0.0.1:4200`
- `http://127.0.0.1:3000`

### Production Mode

To restrict origins in production, add a `dashboard` section to your `.palace/palace.jsonc`:

```json
{
  "schemaVersion": "1.0.0",
  "kind": "palace/config",
  "project": {
    "name": "my-project"
  },
  "dashboard": {
    "cors": {
      "allowedOrigins": [
        "https://your-production-domain.com",
        "https://app.your-domain.com"
      ]
    }
  },
  "provenance": {
    "createdBy": "user",
    "createdAt": "2026-01-05T00:00:00Z"
  }
}
```

## Implementation Details

### Files Modified

1. **apps/cli/schemas/palace.schema.json**

   - Added `dashboard` configuration section with CORS settings
   - Defined `allowedOrigins` array for whitelist

2. **apps/cli/internal/config/config.go**

   - Added `DashboardConfig` struct
   - Added `CORSConfig` struct with `AllowedOrigins` field
   - Integrated into `PalaceConfig`

3. **apps/cli/internal/dashboard/server.go**

   - Added `allowedOrigins` field to `Server` struct
   - Updated `Config` to include `AllowedOrigins`
   - Replaced `corsMiddleware` with `configureCORS` method
   - Implements strict origin checking:
     - Only allowed origins receive CORS headers
     - Forbidden response for disallowed origins on OPTIONS
   - Defaults to localhost origins when none configured

4. **apps/cli/internal/dashboard/websocket.go**

   - Removed global `upgrader` variable with `CheckOrigin: true` (security issue)
   - Created `createUpgrader()` function that checks against allowed origins
   - Updated `ServeWS()` to accept `allowedOrigins` parameter

5. **apps/cli/internal/cli/commands/dashboard.go**

   - Added config import
   - Loads CORS configuration from `palace.jsonc`
   - Passes `AllowedOrigins` to dashboard server

6. **apps/cli/internal/dashboard/websocket_test.go**
   - Updated all test cases to pass `allowedOrigins` to `ServeWS()`

## Security Features

### HTTP CORS Middleware

The `configureCORS` method provides:

- **Origin Validation:** Only requests from whitelisted origins receive CORS headers
- **Preflight Handling:** OPTIONS requests from disallowed origins return 403 Forbidden
- **Headers Set:**
  - `Access-Control-Allow-Origin`: Echoes the request origin if allowed (not `*`)
  - `Access-Control-Allow-Methods`: GET, POST, PUT, DELETE, OPTIONS
  - `Access-Control-Allow-Headers`: Content-Type, Authorization
  - `Access-Control-Allow-Credentials`: true

### WebSocket Origin Checking

The WebSocket upgrader:

- Checks the `Origin` header against the allowed list
- Rejects upgrade requests from unauthorized origins
- Returns 403 before establishing WebSocket connection

## Migration Guide

### For Existing Users

No action required. The dashboard will continue working in development mode with localhost origins.

### For Production Deployments

1. Edit `.palace/palace.jsonc`
2. Add the `dashboard` section with your allowed origins
3. Restart the dashboard server

### Example: Corporate Environment

```json
{
  "dashboard": {
    "cors": {
      "allowedOrigins": [
        "https://mindpalace.company.internal",
        "https://dev.mindpalace.company.internal"
      ]
    }
  }
}
```

## Testing

Run the test suite to verify CORS implementation:

```powershell
cd c:\git\mind-palace\apps\cli
go test ./internal/dashboard/... -v
```

## Security Best Practices

1. **Never use `*` for allowed origins in production**
2. **Always specify the exact protocol and port** (e.g., `http://localhost:3000`, not `localhost`)
3. **Use HTTPS in production** to prevent man-in-the-middle attacks
4. **Regularly review allowed origins** and remove any that are no longer needed
5. **Consider IP-based restrictions** for internal-only dashboards

## Future Enhancements

Potential improvements for future versions:

- [ ] Support for wildcard subdomains (e.g., `*.company.com`)
- [ ] Dynamic origin configuration via environment variables
- [ ] Rate limiting per origin
- [ ] Origin-based authentication requirements
- [ ] Audit logging for rejected connection attempts

## References

- [OWASP CORS Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html)
- [MDN Web Docs: CORS](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS)
- [WebSocket Security](https://owasp.org/www-community/vulnerabilities/WebSocket_Security)
