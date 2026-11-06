package blame

import (
	"context"
	"fmt"
	"time"

	"betterpd/internal/pd"
)

type SyncResult struct {
	Changes    []Change
	IsInitial  bool
	ScheduleCount int
}

func Sync(ctx context.Context, client *pd.Client, store *DB, scheduleIDs map[string]string, lookAheadDays int, syncInterval time.Duration, force bool) (*SyncResult, error) {
	if !force {
		lastSync, err := store.GetLastSyncTime()
		if err != nil {
			return nil, fmt.Errorf("checking last sync time: %w", err)
		}
		if !lastSync.IsZero() && time.Since(lastSync) < syncInterval {
			return &SyncResult{}, nil
		}
	}

	now := time.Now()
	until := now.Add(time.Duration(lookAheadDays) * 24 * time.Hour)

	auditRecords, _ := client.ListScheduleAuditRecords(ctx, now.Add(-syncInterval))

	auditBySchedule := make(map[string][]pd.ScheduleAuditRecord)
	for _, r := range auditRecords {
		auditBySchedule[r.ScheduleID] = append(auditBySchedule[r.ScheduleID], r)
	}

	var allChanges []Change
	isInitial := false

	for name, id := range scheduleIDs {
		entries, err := client.GetRenderedSchedule(ctx, id, now, until)
		if err != nil {
			return nil, fmt.Errorf("fetching schedule %q: %w", name, err)
		}

		current := toStoredEntries(entries)

		previous, err := store.GetLastSnapshot(id)
		if err != nil {
			return nil, fmt.Errorf("getting last snapshot for %q: %w", name, err)
		}

		if previous == nil {
			isInitial = true
		} else {
			changes := diffEntries(id, name, previous, current, auditBySchedule[id])
			for _, c := range changes {
				if err := store.RecordChange(c); err != nil {
					return nil, fmt.Errorf("recording change: %w", err)
				}
			}
			allChanges = append(allChanges, changes...)
		}

		overrides, err := client.ListScheduleOverrides(ctx, id, now, until)
		if err == nil {
			overrideChanges := detectOverrideChanges(id, name, overrides, entries)
			for _, c := range overrideChanges {
				if err := store.RecordChange(c); err != nil {
					return nil, fmt.Errorf("recording override change: %w", err)
				}
			}
			allChanges = append(allChanges, overrideChanges...)
		}

		if err := store.SaveSnapshot(id, name, current); err != nil {
			return nil, fmt.Errorf("saving snapshot for %q: %w", name, err)
		}
	}

	if err := store.SetLastSyncTime(now); err != nil {
		return nil, fmt.Errorf("updating sync time: %w", err)
	}

	return &SyncResult{
		Changes:       allChanges,
		IsInitial:     isInitial,
		ScheduleCount: len(scheduleIDs),
	}, nil
}

func toStoredEntries(entries []pd.ScheduleEntry) []StoredEntry {
	stored := make([]StoredEntry, len(entries))
	for i, e := range entries {
		stored[i] = StoredEntry{
			UserID:   e.UserID,
			UserName: e.UserName,
			Start:    e.Start.UTC().Format(time.RFC3339),
			End:      e.End.UTC().Format(time.RFC3339),
		}
	}
	return stored
}

func diffEntries(scheduleID, scheduleName string, old, new []StoredEntry, audits []pd.ScheduleAuditRecord) []Change {
	type entryKey struct {
		Start string
		End   string
	}

	oldMap := make(map[entryKey]StoredEntry)
	for _, e := range old {
		oldMap[entryKey{e.Start, e.End}] = e
	}

	newMap := make(map[entryKey]StoredEntry)
	for _, e := range new {
		newMap[entryKey{e.Start, e.End}] = e
	}

	now := time.Now()
	changedBy := latestActor(audits)

	var changes []Change

	// Check for modified or removed entries
	for key, oldEntry := range oldMap {
		if newEntry, ok := newMap[key]; ok {
			if oldEntry.UserID != newEntry.UserID {
				start, _ := time.Parse(time.RFC3339, key.Start)
				end, _ := time.Parse(time.RFC3339, key.End)
				changes = append(changes, Change{
					ScheduleID:   scheduleID,
					ScheduleName: scheduleName,
					DetectedAt:   now,
					ChangeType:   "modified",
					Description:  fmt.Sprintf("%s replaced by %s", oldEntry.UserName, newEntry.UserName),
					OriginalUser: oldEntry.UserName,
					NewUser:      newEntry.UserName,
					Start:        start,
					End:          end,
					ChangedBy:    changedBy,
				})
			}
		} else {
			start, _ := time.Parse(time.RFC3339, key.Start)
			end, _ := time.Parse(time.RFC3339, key.End)
			changes = append(changes, Change{
				ScheduleID:   scheduleID,
				ScheduleName: scheduleName,
				DetectedAt:   now,
				ChangeType:   "removed",
				Description:  fmt.Sprintf("Shift removed: %s", oldEntry.UserName),
				OriginalUser: oldEntry.UserName,
				Start:        start,
				End:          end,
				ChangedBy:    changedBy,
			})
		}
	}

	// Check for added entries
	for key, newEntry := range newMap {
		if _, ok := oldMap[key]; !ok {
			start, _ := time.Parse(time.RFC3339, key.Start)
			end, _ := time.Parse(time.RFC3339, key.End)
			changes = append(changes, Change{
				ScheduleID:   scheduleID,
				ScheduleName: scheduleName,
				DetectedAt:   now,
				ChangeType:   "added",
				Description:  fmt.Sprintf("Shift added: %s", newEntry.UserName),
				NewUser:      newEntry.UserName,
				Start:        start,
				End:          end,
				ChangedBy:    changedBy,
			})
		}
	}

	return changes
}

func detectOverrideChanges(scheduleID, scheduleName string, overrides []pd.ScheduleOverride, rendered []pd.ScheduleEntry) []Change {
	// Build a set of rendered user assignments by time range to see who the
	// override replaced. This is best-effort — we only detect overrides that
	// replace a different user.
	type timeRange struct{ Start, End time.Time }
	renderedMap := make(map[timeRange]string)
	for _, e := range rendered {
		renderedMap[timeRange{e.Start, e.End}] = e.UserName
	}

	// We don't record override changes here because the schedule diff already
	// catches the net effect. Overrides are tracked via the schedule snapshot
	// diff above — keeping this function as a hook for future enrichment.
	return nil
}

func latestActor(audits []pd.ScheduleAuditRecord) string {
	if len(audits) == 0 {
		return ""
	}
	latest := audits[0]
	for _, a := range audits[1:] {
		if a.ExecutionTime.After(latest.ExecutionTime) {
			latest = a
		}
	}
	return latest.ActorName
}
