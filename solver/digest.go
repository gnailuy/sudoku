package solver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// ConfigDigest binds persisted catalog metadata to the ordered strategy
// registry, tier assignment, and weight configuration that classified it.
func ConfigDigest(store Store) string {
	type entry struct {
		Technique string `json:"technique"`
		Tier      string `json:"tier"`
		Weight    int    `json:"weight"`
	}
	entries := make([]entry, 0, len(store.GetAllStrategySolverKeys()))
	for _, key := range store.GetAllStrategySolverKeys() {
		tier, _, _ := StrategyTierForTechnique(key)
		entries = append(entries, entry{Technique: key, Tier: tier, Weight: store.GetStrategySolverByKey(key).GetWeight()})
	}
	data, _ := json.Marshal(entries)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
