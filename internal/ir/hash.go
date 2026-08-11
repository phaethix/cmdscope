package ir

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
)

// EffectID is the stable digest ValidateReport recomputes. Layout is fixed so
// gold cases and adapters agree across hosts; changing it breaks the contract.
// Inputs: schema version, stage, kind, raw_target, target, canonical condition,
// provenance — never certainty or evidence (those may differ without identity drift).
func EffectID(schemaVersion string, ef Effect) string {
	canon := canonicalCondition(ef.Condition)
	payload := schemaVersion + strconv.Itoa(ef.Stage) + string(ef.Kind) + ef.RawTarget + ef.Target + canon + string(ef.Provenance)
	sum := sha256.Sum256([]byte(payload))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// canonicalCondition is the fixed JSON fragment inside EffectID. Keep it
// byte-stable; a prettier encoder would reshuffle gold digests.
func canonicalCondition(c Condition) string {
	return fmt.Sprintf(`{"kind":%q,"depends_on":%d}`, string(c.Kind), c.DependsOn)
}
