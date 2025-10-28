# OpenZiti Controller & SDK Compatibility Matrix

This document provides compatibility information between OpenZiti Controller versions and the Go SDK (`github.com/openziti/sdk-golang`) used by the EVE framework.

## Compatibility Matrix

| Controller Version | SDK Version | Status | Notes |
|-------------------|-------------|---------|-------|
| v1.6.5 and below | v1.2.2 | ✅ **RECOMMENDED** | Stable, tested, production-ready |
| v1.6.5 and below | v1.2.3 - v1.2.10 | ❌ **INCOMPATIBLE** | HA/OIDC features require controller v1.6.8+ |
| v1.6.8 - v1.6.9 | v1.2.2 | ✅ Compatible | Works but doesn't use new features |
| v1.6.8 - v1.6.9 | v1.2.3+ | ✅ Compatible | Full HA/OIDC support enabled |
| v1.6.6 - v1.6.7 | v1.2.3+ | ⚠️ **UNTESTED** | May work partially, upgrade to v1.6.8+ recommended |

## Version-Specific Notes

### SDK v1.2.2 (Current EVE Default)
**Released:** January 2025
**Compatible Controllers:** v1.6.0 - v1.6.9+
**Features:**
- Traditional certificate-based authentication
- No automatic HA controller discovery
- No OIDC authentication support
- Stable and reliable for single-controller deployments

**Known Issues:**
- None

### SDK v1.2.3
**Released:** January 2025
**Compatible Controllers:** v1.6.8+
**Breaking Changes:**
- **Removed EnableHA flag requirement** - HA functionality now automatic
- **Automatic HA controller discovery** - Attempts to list and connect to multiple controllers
- **OIDC authentication attempted by default** if controller supports it

**Known Issues:**
- ❌ **UNAUTHORIZED errors** with controllers v1.6.7 and below
- ❌ Fails when client API uses separate certificate chain (Issue #779)
- Error: `[GET /controllers][401] listControllersUnauthorized`

**Required Controller Features:**
- OIDC authentication support
- Multi-controller listing API
- Compatible certificate configuration

### SDK v1.2.4
**Released:** January 2025
**Compatible Controllers:** v1.6.8+
**Breaking Changes:**
- **Go version requirement: 1.24+**

**Inherits issues from v1.2.3**

### SDK v1.2.5
**Released:** January 2025
**Compatible Controllers:** v1.6.8+
**Changes:**
- Simplified OIDC flow for applications (Issue #804)

**Inherits issues from v1.2.3**

### SDK v1.2.8
**Released:** January 2025
**Compatible Controllers:** v1.6.8+
**Breaking Changes:**
- Migrated from `github.com/mailru/easyjson`
- Major dependency updates

**Inherits issues from v1.2.3**

### SDK v1.2.9
**Released:** January 2025
**Compatible Controllers:** v1.6.8+
**Bug Fixes:**
- Fixed: Full re-auth should not clear services list (Issue #818)
- Fixed: Goroutines stuck when iterating over randomized HA controller list (Issue #817)

**Inherits issues from v1.2.3**

### SDK v1.2.10 (Latest)
**Released:** January 2025
**Compatible Controllers:** v1.6.8+
**New Features:**
- HA Posture Check Support

**Inherits issues from v1.2.3**

## Testing Results

Based on empirical testing with Controller v1.6.5:

| SDK Version | Authentication | Service Discovery | HTTP Requests | Overall |
|-------------|---------------|-------------------|---------------|---------|
| v1.2.2 | ✅ Success | ✅ Success (81 services) | ✅ Success (200 OK) | ✅ **WORKS** |
| v1.2.3 | ❌ UNAUTHORIZED | ❌ Failed | ❌ Failed | ❌ FAILS |
| v1.2.4 | ❌ UNAUTHORIZED | ❌ Failed | ❌ Failed | ❌ FAILS |
| v1.2.10 | ❌ UNAUTHORIZED | ❌ Failed | ❌ Failed | ❌ FAILS |

## Upgrade Path

### Current State
- **Controller:** v1.6.5
- **SDK:** v1.2.2
- **Status:** ✅ Production Ready

### To Use SDK v1.2.3+
1. **Upgrade Controller** to v1.6.8 or later
2. **Test authentication** with upgraded controller
3. **Update SDK** to desired version (v1.2.10 recommended for latest features)
4. **Run compatibility tests** to verify functionality

### Recommended Approach
```bash
# 1. Upgrade controller first
# (Follow OpenZiti controller upgrade documentation)

# 2. Verify controller version
curl -sk https://your-controller:1280/edge/client/v1/version | jq -r '.data.version'

# 3. Update SDK in go.mod
go get github.com/openziti/sdk-golang@v1.2.10

# 4. Run tests
go test ./...
```

## Known Issues & Workarounds

### Issue: UNAUTHORIZED with SDK v1.2.3+ on Controller v1.6.5-
**Symptoms:**
```
error listing controllers: [GET /controllers][401] listControllersUnauthorized
no apiSession, authentication attempt failed: UNAUTHORIZED
```

**Root Cause:**
- SDK v1.2.3+ automatically attempts HA controller discovery
- Controller v1.6.5 doesn't support the new authentication flow
- OIDC authentication fails with separate certificate chains

**Workaround:**
- Stay on SDK v1.2.2 until controller can be upgraded to v1.6.8+

### Issue: Separate Certificate Chains
**Symptoms:** OIDC authentication fails even with controller v1.6.8+

**Root Cause:** Client API uses different certificate chain than controller root identity

**Workaround:**
- Configure client API to use same certificate chain as controller
- Or stay on SDK v1.2.2 which uses traditional certificate authentication

## Version Detection

EVE includes automatic version detection to warn about incompatibilities. See `version_checker.go` for implementation.

## References

- [OpenZiti SDK Changelog](https://github.com/openziti/sdk-golang/blob/main/CHANGELOG.md)
- [OpenZiti Controller Releases](https://github.com/openziti/ziti/releases)
- [Issue #779: Remove EnableHA flag](https://github.com/openziti/sdk-golang/issues/779)
- [Discourse: List controller errors after SDK update](https://openziti.discourse.group/t/list-controller-errors-refresh-services-errors-after-updating-ziti-sdk-versions/5003)

## Support

For issues related to OpenZiti compatibility:
- GitHub Issues: https://github.com/openziti/sdk-golang/issues
- OpenZiti Discourse: https://openziti.discourse.group/

---

**Last Updated:** 2025-01-28
**EVE Version:** Current
**Tested With:** Controller v1.6.5, SDK v1.2.2
