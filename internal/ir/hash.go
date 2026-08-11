package ir

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
)

// EffectID is the stable digest ValidateReport recomputes. Layout is fixed so
// gold cases and adapters agree across hosts; changing it breaks the contract.
func EffectID(schemaVersion string, ef Effect) string {
	canon := fmt.Sprintf(`{"kind":%q,"depends_on":%d}`, string(ef.Condition.Kind), ef.Condition.DependsOn)
	payload := schemaVersion + strconv.Itoa(ef.Stage) + string(ef.Kind) + ef.RawTarget + ef.Target + canon + string(ef.Provenance)
	sum := sha256.Sum256([]byte(payload))
	return "sha256:" + hex.EncodeToString(sum[:])
}
