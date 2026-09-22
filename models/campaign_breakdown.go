package models

import (
	"encoding/json"
	"strings"
)

const (
	// BreakdownEmptyDepartment is the bucket label for blank department values.
	BreakdownEmptyDepartment = "未填写"
	// BreakdownEmptyTechnique is the bucket label for campaigns without technique tags.
	BreakdownEmptyTechnique = "未标注"
)

// BreakdownBucket is an aggregated click-rate slice.
type BreakdownBucket struct {
	Department string  `json:"department,omitempty"`
	Technique  string  `json:"technique,omitempty"`
	Total      int64   `json:"total"`
	Clicked    int64   `json:"clicked"`
	ClickRate  float64 `json:"click_rate"`
}

// CampaignBreakdown is the cross-campaign department × technique report.
type CampaignBreakdown struct {
	ByDepartment         []BreakdownBucket `json:"by_department"`
	ByTechnique          []BreakdownBucket `json:"by_technique"`
	DepartmentXTechnique []BreakdownBucket `json:"department_x_technique"`
}

type breakdownResultRow struct {
	Department string
	Status     string
	Techniques string
}

type bucketAcc struct {
	total   int64
	clicked int64
}

func resultCountsAsClicked(status string) bool {
	return status == EventClicked || status == EventDataSubmit
}

func parseTechniquesColumn(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" || raw == "null" {
		return []string{BreakdownEmptyTechnique}
	}
	var techs []string
	if err := json.Unmarshal([]byte(raw), &techs); err != nil || len(techs) == 0 {
		return []string{BreakdownEmptyTechnique}
	}
	out := make([]string, 0, len(techs))
	for _, t := range techs {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		out = append(out, t)
	}
	if len(out) == 0 {
		return []string{BreakdownEmptyTechnique}
	}
	return out
}

func normalizeDepartment(dept string) string {
	dept = strings.TrimSpace(dept)
	if dept == "" {
		return BreakdownEmptyDepartment
	}
	return dept
}

func finalizeBuckets(acc map[string]*bucketAcc, withDept, withTech bool) []BreakdownBucket {
	out := make([]BreakdownBucket, 0, len(acc))
	for key, a := range acc {
		b := BreakdownBucket{
			Total:   a.total,
			Clicked: a.clicked,
		}
		if a.total > 0 {
			b.ClickRate = float64(a.clicked) / float64(a.total)
		}
		parts := strings.SplitN(key, "\x00", 2)
		if withDept {
			b.Department = parts[0]
		}
		if withTech {
			if withDept && len(parts) > 1 {
				b.Technique = parts[1]
			} else {
				b.Technique = parts[0]
			}
		}
		out = append(out, b)
	}
	// Stable-ish order: department then technique then total desc via simple insertion sort.
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if breakdownLess(out[j], out[i]) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func breakdownLess(a, b BreakdownBucket) bool {
	if a.Department != b.Department {
		return a.Department < b.Department
	}
	if a.Technique != b.Technique {
		return a.Technique < b.Technique
	}
	return a.Total > b.Total
}

func addBucket(acc map[string]*bucketAcc, key string, clicked bool) {
	a, ok := acc[key]
	if !ok {
		a = &bucketAcc{}
		acc[key] = a
	}
	a.total++
	if clicked {
		a.clicked++
	}
}

// GetCampaignBreakdown aggregates click rates by department and technique for a user.
// Campaigns with multiple techniques contribute the same result row to each technique bucket.
func GetCampaignBreakdown(uid int64) (*CampaignBreakdown, error) {
	rows := []breakdownResultRow{}
	query := db.Table("results").
		Select("results.department as department, results.status as status, campaigns.techniques as techniques").
		Joins("JOIN campaigns ON campaigns.id = results.campaign_id").
		Where("results.user_id = ?", uid)
	err := whereJoinedCampaignNotDeleted(query).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	byDept := map[string]*bucketAcc{}
	byTech := map[string]*bucketAcc{}
	byBoth := map[string]*bucketAcc{}

	for _, row := range rows {
		dept := normalizeDepartment(row.Department)
		clicked := resultCountsAsClicked(row.Status)
		techs := parseTechniquesColumn(row.Techniques)

		addBucket(byDept, dept, clicked)
		for _, tech := range techs {
			addBucket(byTech, tech, clicked)
			addBucket(byBoth, dept+"\x00"+tech, clicked)
		}
	}

	return &CampaignBreakdown{
		ByDepartment:         finalizeBuckets(byDept, true, false),
		ByTechnique:          finalizeBuckets(byTech, false, true),
		DepartmentXTechnique: finalizeBuckets(byBoth, true, true),
	}, nil
}
