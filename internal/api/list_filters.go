package api

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/pkg/task"
)

const listSortFieldOptions = "created, id, name, pack, progress, received, size, speed, status"

var validListStatuses = map[task.Status]struct{}{
	task.StatusWaiting:   {},
	task.StatusRunning:   {},
	task.StatusPaused:    {},
	task.StatusCompleted: {},
	task.StatusFailed:    {},
}

// ValidateListRequest checks sort, order, and status filter fields on a list request.
func ValidateListRequest(req *task.ListRequest) error {
	if req == nil {
		return nil
	}
	if req.Status != "" {
		if _, ok := validListStatuses[req.Status]; !ok {
			return fmt.Errorf("invalid status %q (valid options: waiting, running, paused, completed, failed)", req.Status)
		}
	}
	for _, st := range req.Statuses {
		if _, ok := validListStatuses[st]; !ok {
			return fmt.Errorf("invalid status %q (valid options: waiting, running, paused, completed, failed)", st)
		}
	}
	if s := strings.TrimSpace(req.Sort); s != "" {
		if canonicalSortField(s) == "" {
			return fmt.Errorf("invalid sort field %q (valid options: %s)", s, listSortFieldOptions)
		}
	}
	if o := strings.TrimSpace(req.Order); o != "" {
		if !isValidSortOrder(o) {
			return fmt.Errorf("invalid sort order %q (valid options: asc, desc)", o)
		}
	}
	return nil
}

func canonicalSortField(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "created", "createdat":
		return "created"
	case "id":
		return "id"
	case "name":
		return "name"
	case "pack", "packname":
		return "pack"
	case "progress":
		return "progress"
	case "received", "receivedbytes":
		return "received"
	case "size", "filesize":
		return "size"
	case "speed":
		return "speed"
	case "status":
		return "status"
	default:
		return ""
	}
}

func isValidSortOrder(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "asc", "desc":
		return true
	default:
		return false
	}
}

func resolveListSort(req *task.ListRequest) (field, order string, doSort bool) {
	sortRaw := strings.TrimSpace(req.Sort)
	orderRaw := strings.TrimSpace(req.Order)
	if sortRaw == "" && orderRaw == "" {
		return "", "", false
	}
	if sortRaw == "" {
		field = "created"
	} else {
		field = canonicalSortField(sortRaw)
	}
	if orderRaw == "" {
		order = "asc"
	} else {
		order = strings.ToLower(orderRaw)
	}
	return field, order, true
}

func taskMatchesStatus(st task.Status, req *task.ListRequest) bool {
	if len(req.Statuses) > 0 {
		for _, want := range req.Statuses {
			if st == want {
				return true
			}
		}
		return false
	}
	if req.Status != "" && st != req.Status {
		return false
	}
	return true
}

func taskMatchesPack(packName string, packs []string) bool {
	if len(packs) == 0 {
		return true
	}
	got := strings.ToLower(strings.TrimSpace(packName))
	for _, want := range packs {
		if got == strings.ToLower(strings.TrimSpace(want)) {
			return true
		}
	}
	return false
}

func normalizeFileExt(ext string) string {
	return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(ext), "."))
}

func taskMatchesExt(fileExt string, exts []string) bool {
	if len(exts) == 0 {
		return true
	}
	got := normalizeFileExt(fileExt)
	for _, want := range exts {
		norm := normalizeFileExt(want)
		if norm == "" {
			continue
		}
		if got == norm {
			return true
		}
	}
	return false
}

func taskMatchesQuery(t *task.Task, query string) bool {
	q := strings.TrimSpace(query)
	if q == "" {
		return true
	}
	q = strings.ToLower(q)
	for _, field := range []string{t.ID, t.Name, t.URL, t.PackName, t.FileExt} {
		if strings.Contains(strings.ToLower(field), q) {
			return true
		}
	}
	return false
}

func statusRank(s task.Status) int {
	switch s {
	case task.StatusWaiting:
		return 0
	case task.StatusRunning:
		return 1
	case task.StatusPaused:
		return 2
	case task.StatusCompleted:
		return 3
	case task.StatusFailed:
		return 4
	default:
		return 5
	}
}

func compareStatus(a, b task.Status) int {
	ra, rb := statusRank(a), statusRank(b)
	if ra != rb {
		return ra - rb
	}
	return strings.Compare(string(a), string(b))
}

func parseCreatedAtForSort(s string) (time.Time, bool) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func compareCreatedAt(a, b string) int {
	ta, okA := parseCreatedAtForSort(a)
	tb, okB := parseCreatedAtForSort(b)
	if okA && okB {
		switch {
		case ta.Before(tb):
			return -1
		case ta.After(tb):
			return 1
		default:
			return 0
		}
	}
	return strings.Compare(a, b)
}

func compareTasks(a, b *task.Task, field string) int {
	switch field {
	case "created":
		return compareCreatedAt(a.CreatedAt, b.CreatedAt)
	case "id":
		return strings.Compare(a.ID, b.ID)
	case "name":
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	case "pack":
		return strings.Compare(strings.ToLower(a.PackName), strings.ToLower(b.PackName))
	case "progress":
		switch {
		case a.Progress < b.Progress:
			return -1
		case a.Progress > b.Progress:
			return 1
		default:
			return 0
		}
	case "received":
		switch {
		case a.ReceivedBytes < b.ReceivedBytes:
			return -1
		case a.ReceivedBytes > b.ReceivedBytes:
			return 1
		default:
			return 0
		}
	case "size":
		switch {
		case a.FileSize < b.FileSize:
			return -1
		case a.FileSize > b.FileSize:
			return 1
		default:
			return 0
		}
	case "speed":
		switch {
		case a.Speed < b.Speed:
			return -1
		case a.Speed > b.Speed:
			return 1
		default:
			return 0
		}
	case "status":
		return compareStatus(a.Status, b.Status)
	default:
		return 0
	}
}

func sortTaskList(tasks []*task.Task, field, order string) {
	desc := order == "desc"
	sort.SliceStable(tasks, func(i, j int) bool {
		cmp := compareTasks(tasks[i], tasks[j], field)
		if desc {
			return cmp > 0
		}
		return cmp < 0
	})
}
