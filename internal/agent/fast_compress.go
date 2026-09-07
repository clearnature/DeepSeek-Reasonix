package agent

// FastCompressManual elides stale tool results into a durable prune projection
// without any model round-trip. It is the /compress-fast host command's entry
// point and lives in its own file (absent upstream) so local-only surface
// merges cannot silently drop it with the upstream prune.go rewrite.
func (a *Agent) FastCompressManual() (PruneStats, error) {
	a.sess.compactionRunMu.Lock()
	defer a.sess.compactionRunMu.Unlock()
	applied, err := a.pruneToolResultsToProjectionLocked(CompactionTriggerManual)
	if err != nil {
		return PruneStats{Mode: toolResultPrune}, err
	}
	if !applied {
		return PruneStats{Mode: toolResultPrune}, nil
	}
	a.sess.compactionMu.Lock()
	receipt := a.sess.compactionState.LastReceipt
	a.sess.compactionMu.Unlock()
	if receipt == nil {
		return PruneStats{Mode: toolResultPrune}, nil
	}
	return PruneStats{Results: receipt.AffectedToolResults, Mode: toolResultPrune}, nil
}
