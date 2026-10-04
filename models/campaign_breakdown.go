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
	// MinReportSample is the delivered-recipient count below which a bucket is too small to rank.
	MinReportSample = 5
)

// BreakdownBucket is an aggregated click-rate slice.
// Total is the delivered count, which is also the click-rate denominator.
type BreakdownBucket struct {
	Department  string  `json:"department,omitempty"`
	Technique   string  `json:"technique,omitempty"`
	Total       int64   `json:"total"`
	Delivered   int64   `json:"delivered"`
	Clicked     int64   `json:"clicked"`
	Submitted   int64   `json:"submitted"`
	Reported    int64   `json:"reported"`
	ClickRate   float64 `json:"click_rate"`
	SampleSmall bool    `json:"sample_small"`
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
	Reported   bool
	Techniques string
}

type bucketAcc struct {
	delivered int64
	clicked   int64
	submitted int64
	reported  int64
}

func resultCountsAsClicked(status string) bool {
	return status == EventClicked || status == EventDataSubmit
}

func resultCountsAsDelivered(status string) bool {
	switch status {
	case EventSent, EventOpened, EventClicked, EventDataSubmit:
		return true
	default:
		return false
	}
}

func resultCountsAsSubmitted(status string) bool {
	return status == EventDataSubmit
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

func rateOf(num, den int64) float64 {
	if den <= 0 {
		return 0
	}
	return float64(num) / float64(den)
}

func finalizeBuckets(acc map[string]*bucketAcc, withDept, withTech bool) []BreakdownBucket {
	out := make([]BreakdownBucket, 0, len(acc))
	for key, a := range acc {
		b := BreakdownBucket{
			Total:       a.delivered,
			Delivered:   a.delivered,
			Clicked:     a.clicked,
			Submitted:   a.submitted,
			Reported:    a.reported,
			ClickRate:   rateOf(a.clicked, a.delivered),
			SampleSmall: a.delivered < MinReportSample,
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
	if a.SampleSmall != b.SampleSmall {
		return !a.SampleSmall
	}
	if a.ClickRate != b.ClickRate {
		return a.ClickRate > b.ClickRate
	}
	if a.Department != b.Department {
		return a.Department < b.Department
	}
	if a.Technique != b.Technique {
		return a.Technique < b.Technique
	}
	return a.Total > b.Total
}

func addDeliveredBucket(acc map[string]*bucketAcc, key string, clicked, submitted, reported bool) {
	a, ok := acc[key]
	if !ok {
		a = &bucketAcc{}
		acc[key] = a
	}
	a.delivered++
	if clicked {
		a.clicked++
	}
	if submitted {
		a.submitted++
	}
	if reported {
		a.reported++
	}
}

// GetCampaignBreakdown aggregates click rates by department and technique for a user.
// Only delivered recipients are counted. Campaigns with multiple techniques contribute
// the same result row to each technique bucket, and once to the department bucket.
func GetCampaignBreakdown(uid int64) (*CampaignBreakdown, error) {
	rows := []breakdownResultRow{}
	query := db.Table("results").
		Select("results.department as department, results.status as status, results.reported as reported, campaigns.techniques as techniques").
		Joins("JOIN campaigns ON campaigns.id = results.campaign_id").
		Where("results.user_id = ?", uid)
	err := whereJoinedCampaignNotDeleted(query).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return breakdownFromRows(rows), nil
}

func breakdownFromRows(rows []breakdownResultRow) *CampaignBreakdown {
	byDept := map[string]*bucketAcc{}
	byTech := map[string]*bucketAcc{}
	byBoth := map[string]*bucketAcc{}

	for _, row := range rows {
		if !resultCountsAsDelivered(row.Status) {
			continue
		}
		dept := normalizeDepartment(row.Department)
		clicked := resultCountsAsClicked(row.Status)
		submitted := resultCountsAsSubmitted(row.Status)
		techs := parseTechniquesColumn(row.Techniques)

		addDeliveredBucket(byDept, dept, clicked, submitted, row.Reported)
		for _, tech := range techs {
			addDeliveredBucket(byTech, tech, clicked, submitted, row.Reported)
			addDeliveredBucket(byBoth, dept+"\x00"+tech, clicked, submitted, row.Reported)
		}
	}

	return &CampaignBreakdown{
		ByDepartment:         finalizeBuckets(byDept, true, false),
		ByTechnique:          finalizeBuckets(byTech, false, true),
		DepartmentXTechnique: finalizeBuckets(byBoth, true, true),
	}
}
