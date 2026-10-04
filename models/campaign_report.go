package models

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

const (
	maxBriefCampaigns = 30
	// CompareNarrativeRetest means every selected campaign shares one scenario pack and sending trust.
	CompareNarrativeRetest = "retest"
	// CompareNarrativeExposure means the selected campaigns are not the same scenario at the same trust tier.
	CompareNarrativeExposure = "exposure"
)

var (
	// ErrReportIDsRequired indicates ids were missing.
	ErrReportIDsRequired = errors.New("ids are required")
	// ErrReportCampaignID indicates an id was not a positive integer.
	ErrReportCampaignID = errors.New("invalid campaign id")
	// ErrBriefNotCompleted indicates a brief included a campaign that is not completed.
	ErrBriefNotCompleted = errors.New("brief only includes completed campaigns")
	// ErrCompareCount indicates compare was not given 2 to 4 campaigns.
	ErrCompareCount = errors.New("compare requires 2 to 4 campaigns")
	// ErrTooManyCampaigns indicates a brief selected more campaigns than the page can hold.
	ErrTooManyCampaigns = errors.New("too many campaigns")
)

// BriefCampaign is one completed campaign on the leadership brief.
type BriefCampaign struct {
	Id             int64      `json:"id"`
	Name           string     `json:"name"`
	LaunchDate     time.Time  `json:"launch_date"`
	ScenarioPackId string     `json:"scenario_pack_id"`
	Difficulty     int        `json:"difficulty"`
	SendingTrust   string     `json:"sending_trust"`
	Techniques     StringList `json:"techniques"`
	Delivered      int64      `json:"delivered"`
	Opened         int64      `json:"opened"`
	Clicked        int64      `json:"clicked"`
	Submitted      int64      `json:"submitted"`
	Reported       int64      `json:"reported"`
	PhishProne     float64    `json:"phish_prone"`
	SubmitRate     float64    `json:"submit_rate"`
	ReportRate     float64    `json:"report_rate"`
}

// BriefCoverage is who the selected campaigns reached.
type BriefCoverage struct {
	CampaignCount int   `json:"campaign_count"`
	Delivered     int64 `json:"delivered"`
	Departments   int   `json:"departments"`
}

// BriefOverall is the pooled rate across the selected campaigns.
type BriefOverall struct {
	Delivered  int64   `json:"delivered"`
	Opened     int64   `json:"opened"`
	Clicked    int64   `json:"clicked"`
	Submitted  int64   `json:"submitted"`
	Reported   int64   `json:"reported"`
	PhishProne float64 `json:"phish_prone"`
	SubmitRate float64 `json:"submit_rate"`
	ReportRate float64 `json:"report_rate"`
}

// BriefTrustRow separates customer-domain and lookalike drills.
type BriefTrustRow struct {
	SendingTrust string  `json:"sending_trust"`
	Delivered    int64   `json:"delivered"`
	Clicked      int64   `json:"clicked"`
	PhishProne   float64 `json:"phish_prone"`
}

// BriefAdvice is the data the page turns into the next-step sentence.
type BriefAdvice struct {
	WeakestDepartment string  `json:"weakest_department,omitempty"`
	WeakestClickRate  float64 `json:"weakest_click_rate,omitempty"`
	NoReports         bool    `json:"no_reports"`
}

// CampaignBrief is the one-page leadership summary for selected completed campaigns.
type CampaignBrief struct {
	Campaigns          []BriefCampaign   `json:"campaigns"`
	Coverage           BriefCoverage     `json:"coverage"`
	Overall            BriefOverall      `json:"overall"`
	BySendingTrust     []BriefTrustRow   `json:"by_sending_trust"`
	WeakestDepartments []BreakdownBucket `json:"weakest_departments"`
	ByTechnique        []BreakdownBucket `json:"by_technique"`
	Trend              []BriefCampaign   `json:"trend"`
	Advice             BriefAdvice       `json:"advice"`
}

// CompareCampaign is one column of a scenario comparison.
type CompareCampaign struct {
	Id             int64             `json:"id"`
	Name           string            `json:"name"`
	Status         string            `json:"status"`
	LaunchDate     time.Time         `json:"launch_date"`
	ScenarioPackId string            `json:"scenario_pack_id"`
	Difficulty     int               `json:"difficulty"`
	SendingTrust   string            `json:"sending_trust"`
	Delivered      int64             `json:"delivered"`
	Opened         int64             `json:"opened"`
	Clicked        int64             `json:"clicked"`
	Submitted      int64             `json:"submitted"`
	Reported       int64             `json:"reported"`
	PhishProne     float64           `json:"phish_prone"`
	Departments    []BreakdownBucket `json:"departments"`
}

// CompareCell is one campaign's rate inside a shared department.
type CompareCell struct {
	CampaignId  int64   `json:"campaign_id"`
	Delivered   int64   `json:"delivered"`
	Clicked     int64   `json:"clicked"`
	ClickRate   float64 `json:"click_rate"`
	SampleSmall bool    `json:"sample_small"`
}

// SharedDepartment is a department present in every compared campaign.
type SharedDepartment struct {
	Department string        `json:"department"`
	Cells      []CompareCell `json:"cells"`
}

// CampaignCompare is a 2–4 campaign side-by-side view.
type CampaignCompare struct {
	Narrative         string             `json:"narrative"`
	Campaigns         []CompareCampaign  `json:"campaigns"`
	SharedDepartments []SharedDepartment `json:"shared_departments"`
}

type reportResultRow struct {
	CampaignId   int64
	Department   string
	Status       string
	Reported     bool
	Techniques   string
	SendingTrust string
}

func parseCampaignIDList(raw string) ([]int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, ErrReportIDsRequired
	}
	seen := map[int64]bool{}
	ids := []int64{}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil || id <= 0 {
			return nil, ErrReportCampaignID
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, ErrReportIDsRequired
	}
	return ids, nil
}

func loadOwnedCampaigns(uid int64, ids []int64) ([]Campaign, error) {
	found := []Campaign{}
	err := db.Where("user_id = ? AND id IN (?)", uid, ids).Where("deleted_at IS NULL").Find(&found).Error
	if err != nil {
		return nil, err
	}
	byID := map[int64]Campaign{}
	for _, c := range found {
		byID[c.Id] = c
	}
	if len(byID) != len(ids) {
		return nil, ErrCampaignNotFound
	}
	ordered := make([]Campaign, 0, len(ids))
	for _, id := range ids {
		ordered = append(ordered, byID[id])
	}
	return ordered, nil
}

func loadReportRows(uid int64, ids []int64) ([]reportResultRow, error) {
	rows := []reportResultRow{}
	err := db.Table("results").
		Select("results.campaign_id as campaign_id, results.department as department, results.status as status, results.reported as reported, campaigns.techniques as techniques, campaigns.sending_trust as sending_trust").
		Joins("JOIN campaigns ON campaigns.id = results.campaign_id").
		Where("results.user_id = ? AND results.campaign_id IN (?)", uid, ids).
		Scan(&rows).Error
	return rows, err
}

func briefCampaignFrom(c Campaign, stats CampaignStats) BriefCampaign {
	return BriefCampaign{
		Id:             c.Id,
		Name:           c.Name,
		LaunchDate:     c.LaunchDate,
		ScenarioPackId: c.ScenarioPackId,
		Difficulty:     c.Difficulty,
		SendingTrust:   c.SendingTrust,
		Techniques:     c.Techniques,
		Delivered:      stats.EmailsSent,
		Opened:         stats.OpenedEmail,
		Clicked:        stats.ClickedLink,
		Submitted:      stats.SubmittedData,
		Reported:       stats.EmailReported,
		PhishProne:     rateOf(stats.ClickedLink, stats.EmailsSent),
		SubmitRate:     rateOf(stats.SubmittedData, stats.EmailsSent),
		ReportRate:     rateOf(stats.EmailReported, stats.EmailsSent),
	}
}

// GetCampaignBrief aggregates a printable summary for completed campaigns owned by the user.
func GetCampaignBrief(uid int64, rawIDs string) (*CampaignBrief, error) {
	ids, err := parseCampaignIDList(rawIDs)
	if err != nil {
		return nil, err
	}
	if len(ids) > maxBriefCampaigns {
		return nil, ErrTooManyCampaigns
	}
	camps, err := loadOwnedCampaigns(uid, ids)
	if err != nil {
		return nil, err
	}
	for _, c := range camps {
		if c.Status != CampaignComplete {
			return nil, ErrBriefNotCompleted
		}
	}
	rows, err := loadReportRows(uid, ids)
	if err != nil {
		return nil, err
	}

	items := make([]BriefCampaign, 0, len(camps))
	var overall BriefOverall
	trustAcc := map[string]*bucketAcc{}
	for _, c := range camps {
		stats, err := getCampaignStats(c.Id)
		if err != nil {
			return nil, err
		}
		item := briefCampaignFrom(c, stats)
		items = append(items, item)
		overall.Delivered += item.Delivered
		overall.Opened += item.Opened
		overall.Clicked += item.Clicked
		overall.Submitted += item.Submitted
		overall.Reported += item.Reported
		addDeliveredCounts(trustAcc, c.SendingTrust, item.Delivered, item.Clicked)
	}
	overall.PhishProne = rateOf(overall.Clicked, overall.Delivered)
	overall.SubmitRate = rateOf(overall.Submitted, overall.Delivered)
	overall.ReportRate = rateOf(overall.Reported, overall.Delivered)

	bdRows := make([]breakdownResultRow, 0, len(rows))
	depts := map[string]bool{}
	for _, row := range rows {
		bdRows = append(bdRows, breakdownResultRow{
			Department: row.Department,
			Status:     row.Status,
			Reported:   row.Reported,
			Techniques: row.Techniques,
		})
		if resultCountsAsDelivered(row.Status) {
			depts[normalizeDepartment(row.Department)] = true
		}
	}
	bd := breakdownFromRows(bdRows)
	weak := make([]BreakdownBucket, 0, 5)
	for _, b := range bd.ByDepartment {
		if b.SampleSmall || b.Delivered == 0 {
			continue
		}
		weak = append(weak, b)
		if len(weak) == 5 {
			break
		}
	}

	trend := append([]BriefCampaign(nil), items...)
	for i := 0; i < len(trend); i++ {
		for j := i + 1; j < len(trend); j++ {
			if trend[j].LaunchDate.Before(trend[i].LaunchDate) {
				trend[i], trend[j] = trend[j], trend[i]
			}
		}
	}

	advice := BriefAdvice{NoReports: overall.Reported == 0 && overall.Delivered > 0}
	if len(weak) > 0 {
		advice.WeakestDepartment = weak[0].Department
		advice.WeakestClickRate = weak[0].ClickRate
	}

	return &CampaignBrief{
		Campaigns: items,
		Coverage: BriefCoverage{
			CampaignCount: len(items),
			Delivered:     overall.Delivered,
			Departments:   len(depts),
		},
		Overall:            overall,
		BySendingTrust:     finalizeTrustRows(trustAcc),
		WeakestDepartments: weak,
		ByTechnique:        bd.ByTechnique,
		Trend:              trend,
		Advice:             advice,
	}, nil
}

func addDeliveredCounts(acc map[string]*bucketAcc, key string, delivered, clicked int64) {
	a, ok := acc[key]
	if !ok {
		a = &bucketAcc{}
		acc[key] = a
	}
	a.delivered += delivered
	a.clicked += clicked
}

func finalizeTrustRows(acc map[string]*bucketAcc) []BriefTrustRow {
	order := []string{SendingTrustCustomerDomain, SendingTrustLookalike, ""}
	seen := map[string]bool{}
	out := []BriefTrustRow{}
	appendRow := func(key string) {
		a, ok := acc[key]
		if !ok {
			return
		}
		seen[key] = true
		out = append(out, BriefTrustRow{
			SendingTrust: key,
			Delivered:    a.delivered,
			Clicked:      a.clicked,
			PhishProne:   rateOf(a.clicked, a.delivered),
		})
	}
	for _, key := range order {
		appendRow(key)
	}
	for key := range acc {
		if !seen[key] {
			appendRow(key)
		}
	}
	return out
}

func compareNarrative(camps []Campaign) string {
	if len(camps) < 2 {
		return CompareNarrativeExposure
	}
	pack := strings.TrimSpace(camps[0].ScenarioPackId)
	trust := strings.TrimSpace(camps[0].SendingTrust)
	if pack == "" || trust == "" {
		return CompareNarrativeExposure
	}
	for _, c := range camps[1:] {
		if strings.TrimSpace(c.ScenarioPackId) != pack || strings.TrimSpace(c.SendingTrust) != trust {
			return CompareNarrativeExposure
		}
	}
	return CompareNarrativeRetest
}

// GetCampaignCompare returns a side-by-side view of 2 to 4 campaigns.
// Incomplete campaigns are allowed. Deleted campaigns and other users' campaigns are not.
func GetCampaignCompare(uid int64, rawIDs string) (*CampaignCompare, error) {
	ids, err := parseCampaignIDList(rawIDs)
	if err != nil {
		return nil, err
	}
	if len(ids) < 2 || len(ids) > 4 {
		return nil, ErrCompareCount
	}
	camps, err := loadOwnedCampaigns(uid, ids)
	if err != nil {
		return nil, err
	}
	rows, err := loadReportRows(uid, ids)
	if err != nil {
		return nil, err
	}

	byCampaignDept := map[int64]map[string]*bucketAcc{}
	deptPresence := map[int64]map[string]bool{}
	for _, id := range ids {
		byCampaignDept[id] = map[string]*bucketAcc{}
		deptPresence[id] = map[string]bool{}
	}
	for _, row := range rows {
		dept := normalizeDepartment(row.Department)
		if deptPresence[row.CampaignId] == nil {
			continue
		}
		deptPresence[row.CampaignId][dept] = true
		if !resultCountsAsDelivered(row.Status) {
			continue
		}
		addDeliveredBucket(byCampaignDept[row.CampaignId], dept, resultCountsAsClicked(row.Status), resultCountsAsSubmitted(row.Status), row.Reported)
	}

	items := make([]CompareCampaign, 0, len(camps))
	for _, c := range camps {
		stats, err := getCampaignStats(c.Id)
		if err != nil {
			return nil, err
		}
		items = append(items, CompareCampaign{
			Id:             c.Id,
			Name:           c.Name,
			Status:         c.Status,
			LaunchDate:     c.LaunchDate,
			ScenarioPackId: c.ScenarioPackId,
			Difficulty:     c.Difficulty,
			SendingTrust:   c.SendingTrust,
			Delivered:      stats.EmailsSent,
			Opened:         stats.OpenedEmail,
			Clicked:        stats.ClickedLink,
			Submitted:      stats.SubmittedData,
			Reported:       stats.EmailReported,
			PhishProne:     rateOf(stats.ClickedLink, stats.EmailsSent),
			Departments:    finalizeBuckets(byCampaignDept[c.Id], true, false),
		})
	}

	sharedNames := []string{}
	if len(ids) > 0 {
		for dept := range deptPresence[ids[0]] {
			ok := true
			for _, id := range ids[1:] {
				if !deptPresence[id][dept] {
					ok = false
					break
				}
			}
			if ok {
				sharedNames = append(sharedNames, dept)
			}
		}
	}
	for i := 0; i < len(sharedNames); i++ {
		for j := i + 1; j < len(sharedNames); j++ {
			if sharedNames[j] < sharedNames[i] {
				sharedNames[i], sharedNames[j] = sharedNames[j], sharedNames[i]
			}
		}
	}
	shared := make([]SharedDepartment, 0, len(sharedNames))
	for _, dept := range sharedNames {
		cells := make([]CompareCell, 0, len(ids))
		for _, id := range ids {
			a := byCampaignDept[id][dept]
			cell := CompareCell{CampaignId: id}
			if a != nil {
				cell.Delivered = a.delivered
				cell.Clicked = a.clicked
				cell.ClickRate = rateOf(a.clicked, a.delivered)
				cell.SampleSmall = a.delivered < MinReportSample
			} else {
				cell.SampleSmall = true
			}
			cells = append(cells, cell)
		}
		shared = append(shared, SharedDepartment{Department: dept, Cells: cells})
	}

	return &CampaignCompare{
		Narrative:         compareNarrative(camps),
		Campaigns:         items,
		SharedDepartments: shared,
	}, nil
}
