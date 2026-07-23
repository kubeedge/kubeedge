# ServiceBus TLS Configuration

## Overview

The ServiceBus embedded HTTP server can optionally be protected with
transport-layer TLS (HTTPS). When TLS is enabled all traffic between
local applications and the ServiceBus HTTP server is encrypted in
transit, and clients can verify the server identity via certificate.

**Scope:** transport encryption and server authentication only.
TLS does not restrict which local processes may submit ServiceBus
requests. Any process that can reach the configured port can still send
requests. Caller authentication (mTLS, application credentials, or a
Unix-socket with permission controls) is a separate concern.

## Prerequisites

You need a dedicated server certificate and private key for the
ServiceBus endpoint. The certificate must satisfy:

| Requirement | Details |
|---|---|
| Key usage | `ExtKeyUsageServerAuth` |
| Subject Alternative Name | An IP or DNS SAN matching `ServiceBus.Server` (default `127.0.0.1`) |
| Format | PEM-encoded X.509 certificate (`.crt` / `.pem`) |
| Private key | PEM-encoded RSA or ECDSA key matching the certificate (`.key`) |

> **Important:** The EdgeHub client certificate issued by CloudCore
> **cannot** be reused. It carries only `ExtKeyUsageClientAuth` and has
> no ServiceBus SANs. Attempting to use it will cause TLS handshake
> failures in any standard HTTPS client.

### Generating a self-signed certificate (example)

```bash
# Generate a 4096-bit RSA key
openssl genrsa -out /etc/kubeedge/certs/servicebus-server.key 4096

# Generate a self-signed certificate valid for 1 year,
# with IP SAN 127.0.0.1 (the default ServiceBus listen address).
openssl req -new -x509 -days 365 \
  -key /etc/kubeedge/certs/servicebus-server.key \
  -out /etc/kubeedge/certs/servicebus-server.crt \
  -subj "/CN=servicebus.edge.local" \
  -addext "subjectAltName=IP:127.0.0.1" \
  -addext "extendedKeyUsage=serverAuth"
```

For production deployments, issue the certificate from your internal CA
or a certificate management tool (e.g. cert-manager) rather than using
a self-signed certificate.

## Enabling TLS in EdgeCore configuration

Set `tlsCertFile` and `tlsPrivateKeyFile` under `modules.serviceBus` in
your EdgeCore configuration (`/etc/kubeedge/config/edgecore.yaml`):

```yaml
apiVersion: edgecore.config.kubeedge.io/v1alpha2
kind: EdgeCore
modules:
  serviceBus:
    enable: true
    server: 127.0.0.1
    port: 9060
    timeout: 60
    # Enable TLS by providing a dedicated server certificate.
    # Leave both fields empty (the default) to use plain HTTP.
    tlsCertFile: /etc/kubeedge/certs/servicebus-server.crt
    tlsPrivateKeyFile: /etc/kubeedge/certs/servicebus-server.key
```

When `tlsCertFile` is non-empty, EdgeCore starts the ServiceBus HTTP
server as an HTTPS server. When both fields are empty (the default),
the server starts in plain-HTTP mode, preserving backward compatibility.

If `tlsCertFile` is set but the certificate or key file is missing or
the key pair cannot be loaded, EdgeCore will **not** silently fall back
to HTTP — it will log an error and refuse to start the ServiceBus HTTPS
server. This prevents a certificate typo or missing file from silently
removing transport security.

Note: EdgeCore validates two things at startup before the HTTPS server
starts:

1. **Key-pair loadability** — the certificate and key files must form a
   valid PEM key pair; otherwise EdgeCore logs an error and refuses to
   start the HTTPS server.
2. **`ExtKeyUsageServerAuth`** — the certificate must contain the
   `serverAuth` extended key usage. EdgeCore checks this at startup via
   `validateServerAuthEKU()`. A ClientAuth-only certificate (such as the
   EdgeHub client certificate issued by CloudCore) is rejected
   immediately with a clear error message, before the server starts.
   The same check is applied again during certificate rotation: a
   rotated certificate that lacks `ExtKeyUsageServerAuth` is also
   rejected, so startup validation cannot be bypassed by a later
   certificate swap.

**SAN matching is not checked by EdgeCore.** A certificate whose Subject
Alternative Names do not include the ServiceBus listen address (e.g.
`127.0.0.1`) will start the server successfully, but HTTPS clients will
reject the TLS handshake because they enforce SAN matching during
certificate verification. Always include the correct IP or DNS SAN in
your certificate (see the `openssl` example above).

## Updating local application clients

When TLS is enabled, local applications that previously connected to
`http://127.0.0.1:9060` must switch to `https://127.0.0.1:9060` and
trust the ServiceBus server certificate (or its CA). For example, using
`curl`:

```bash
# Plain HTTP (TLS disabled — default)
curl http://127.0.0.1:9060/

# HTTPS (TLS enabled) — trust the self-signed cert or your CA bundle
curl --cacert /etc/kubeedge/certs/servicebus-server.crt \
     https://127.0.0.1:9060/
```

## Certificate rotation

The ServiceBus TLS implementation uses `GetCertificate`, which re-reads
the certificate and key files on every new TLS handshake. You can rotate
the certificate by replacing the files on disk; no EdgeCore restart is
required.

## Plain-HTTP default (backward compatibility)

Leaving `tlsCertFile` and `tlsPrivateKeyFile` empty (or absent from
the configuration) keeps the ServiceBus server in plain-HTTP mode,
exactly as before this feature was introduced. Existing deployments are
unaffected.
