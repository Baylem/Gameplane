# Redaction filter for diagnostic output written to CI logs.
# Shared by every step of the dump-cluster-state composite action and
# invoked as: sed -E -f "$GITHUB_ACTION_PATH/redact.sed" (GNU sed).

# PEM private key blocks are replaced as a whole.
/-----BEGIN [A-Z ]*PRIVATE KEY-----/,/-----END [A-Z ]*PRIVATE KEY-----/c\
***REDACTED-KEY***

# Credentials embedded in URL userinfo (scheme://user:pass@host).
s|([A-Za-z][A-Za-z0-9+.-]*://)[^/?#@[:space:]"']+@|\1***REDACTED***@|g

# JSON Web Tokens.
s/eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]*\.[A-Za-z0-9_-]*/***REDACTED-JWT***/g

# Bare auth-scheme credentials (Authorization header values without a key=).
s/(bearer|basic)[[:space:]]+[^[:space:]"',;]+/\1 ***REDACTED***/Ig

# Values following sensitive key names, case-insensitive. Handles
# key=value, key: value and quoted/JSON forms ("key": "value"); the rest of
# the line after the separator is replaced.
s/(password|passwd|passphrase|credentials?|private[-_]?key|client[-_]?secret|access[-_]?key[-_]?i?d?|secret[-_]?key|x-api-key|auth[-_]?config|api[-_]?key|secret|token|set-cookie|cookie|session|bearer|authorization)(["']*[[:space:]]*[=:][[:space:]]*["']*).*/\1\2***REDACTED***/I
