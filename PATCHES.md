# Vendor Patches

## update
When the dependency updates:

  go get golang.org/x/crypto@<new-version>
  go mod vendor
  # then apply the two changes documented in PATCHES.md
  go build ./...

  The find/replace blocks are exact so it's a mechanical operation — or you could even script it with sed if you want to automate it.

## `vendor/golang.org/x/crypto/ssh/proxy.go`

**Why:** The upstream library's `handleAuthMsg` silently swallows all auth failures
with no logging. These patches add diagnostic log lines so operators can distinguish
between the three failure cases without modifying application code.

**After any `go mod vendor` run, replay these changes.**

---

### 1. Add `"log"` import

```go
import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"        // add this
	"net"
	"os"
	"path"
)
```

---

### 2. Replace the silent failure block in `handleAuthMsg` (around the `checkPublicKeyRegistration` / `VerifySignature` calls)

**Find:**
```go
		authKeys, err := proxyConf.FetchAuthorizedKeysHook(username, p.DestinationHost)
		if err != nil {
			return noneAuthMsg(username), nil
		}

		ok, err := checkPublicKeyRegistration(authKeys, downStreamPublicKey)
		if err != nil || !ok {
			return noneAuthMsg(username), nil
		}

		ok, err = p.VerifySignature(msg, downStreamPublicKey, algo, sig)
		if err != nil || !ok {
			break
		}

		privateBytes, err := fetchPrivateKey(proxyConf, p.User)
		if err != nil {
			break
		}
```

**Replace with:**
```go
		authKeys, err := proxyConf.FetchAuthorizedKeysHook(username, p.DestinationHost)
		if err != nil {
			log.Printf("[ssh/proxy] user=%s authorized_keys unreadable: %v", username, err)
			return noneAuthMsg(username), nil
		}

		ok, err := checkPublicKeyRegistration(authKeys, downStreamPublicKey)
		if err != nil {
			log.Printf("[ssh/proxy] user=%s authorized_keys parse error: %v", username, err)
			return noneAuthMsg(username), nil
		}
		if !ok {
			log.Printf("[ssh/proxy] user=%s pubkey NOT found in authorized_keys type=%s fp=%s", username, downStreamPublicKey.Type(), FingerprintSHA256(downStreamPublicKey))
			return noneAuthMsg(username), nil
		}

		log.Printf("[ssh/proxy] user=%s pubkey FOUND in authorized_keys type=%s fp=%s — verifying signature", username, downStreamPublicKey.Type(), FingerprintSHA256(downStreamPublicKey))
		ok, err = p.VerifySignature(msg, downStreamPublicKey, algo, sig)
		if err != nil {
			log.Printf("[ssh/proxy] user=%s signature verification error: %v", username, err)
			break
		}
		if !ok {
			log.Printf("[ssh/proxy] user=%s signature MISMATCH (key is in authorized_keys but sig invalid)", username)
			break
		}

		privateBytes, err := fetchPrivateKey(proxyConf, p.User)
		if err != nil {
			log.Printf("[ssh/proxy] user=%s fetch master private key failed: %v", username, err)
			break
		}
```
