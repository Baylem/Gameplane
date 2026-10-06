package main

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"
)

// validRanges are the dashboard ranges in days (FR-022); anything else falls
// back to defaultRange.
var validRanges = []int{7, 30, 90, 365}

const (
	defaultRange = 30
	// topVersions is how many versions are listed before "Other" (FR-024).
	topVersions = 10
	// Labels of the two aggregate version rows. Stored rows use the lower-case
	// forms written by versionLabel.
	versionOtherLabel   = "Other"
	versionInvalidLabel = "Invalid"
	versionOtherStored  = "other"
	versionInvalidStore = "invalid"
)

// fleetBandBounds are the inclusive upper bounds of the fleet-size bands
// (data-model.md); a final "250+" band takes everything above the last bound.
var fleetBandBounds = []int{0, 1, 2, 5, 10, 25, 50, 100, 250}

// fleetBandOver is the label of the last, open-ended fleet band.
const fleetBandOver = "250+"

// Views is the dashboard view model, the body of GET /api/v1/views
// (contracts/receiver-http.md). Extended stays nil until US5 fills it.
type Views struct {
	Range    int            `json:"range"`
	AsOf     string         `json:"asOf"`
	Empty    bool           `json:"empty"`
	Basic    *BasicViews    `json:"basic"`
	Extended *ExtendedViews `json:"extended"`
}

// ExtendedViews is the extended block, added by US5.
type ExtendedViews struct{}

// BasicViews is the basic block (FR-024).
type BasicViews struct {
	ReportsPerDay []DayReports `json:"reportsPerDay"`
	Versions      []VersionRow `json:"versions"`
	Fleet         FleetViews   `json:"fleet"`
	LatestDay     LatestDayRow `json:"latestDay"`
}

// DayReports is one day of the reports-per-day trend.
type DayReports struct {
	Day     string `json:"day"`
	Reports int64  `json:"reports"`
}

// VersionRow is one version-adoption bar.
type VersionRow struct {
	Label   string `json:"label"`
	Reports int64  `json:"reports"`
}

// FleetViews holds the fleet-size distributions.
type FleetViews struct {
	Servers   FleetDistribution `json:"servers"`
	Templates FleetDistribution `json:"templates"`
}

// FleetDistribution is a banded histogram plus the exact median. A median of
// 1001 means "more than 1000".
type FleetDistribution struct {
	Bands  []FleetBand `json:"bands"`
	Median float64     `json:"median"`
}

// FleetBand is one histogram band; Le is the inclusive upper bound ("250+"
// for the open band).
type FleetBand struct {
	Le      string `json:"le"`
	Reports int64  `json:"reports"`
}

// LatestDayRow holds the totals for the as-of day.
type LatestDayRow struct {
	ServersTotal   int64 `json:"serversTotal"`
	TemplatesTotal int64 `json:"templatesTotal"`
}

// normalizeRange returns days when it is a valid range, else defaultRange.
func normalizeRange(days int) int {
	for _, r := range validRanges {
		if days == r {
			return days
		}
	}
	return defaultRange
}

// BuildViews computes the basic dashboard views over the rangeDays ending
// on the latest complete UTC day, which is yesterday relative to now. An
// invalid rangeDays falls back to 30. When no daily_basic row exists in the
// range the result is the empty state: Empty is true and Basic is nil.
func BuildViews(ctx context.Context, st *store, rangeDays int, now time.Time) (Views, error) {
	rangeDays = normalizeRange(rangeDays)
	asOf := now.UTC().Truncate(24*time.Hour).AddDate(0, 0, -1)
	from := asOf.AddDate(0, 0, 1-rangeDays)
	v := Views{Range: rangeDays, AsOf: dayString(asOf)}
	fromDay, toDay := dayString(from), dayString(asOf)

	perDay, latest, err := queryDailyBasic(ctx, st.db, fromDay, toDay, v.AsOf)
	if err != nil {
		return Views{}, err
	}
	if len(perDay) == 0 {
		v.Empty = true
		return v, nil
	}
	b := &BasicViews{LatestDay: latest}
	b.ReportsPerDay = denseDays(from, rangeDays, perDay)
	if b.Versions, err = queryVersions(ctx, st.db, fromDay, toDay); err != nil {
		return Views{}, err
	}
	if b.Fleet.Servers, err = queryFleet(ctx, st.db, fleetMetricServers, fromDay, toDay); err != nil {
		return Views{}, err
	}
	if b.Fleet.Templates, err = queryFleet(ctx, st.db, fleetMetricTemplates, fromDay, toDay); err != nil {
		return Views{}, err
	}
	v.Basic = b
	return v, nil
}

// queryDailyBasic returns reports per day that has a row in [from, to], and
// the totals of the asOf day (zero when it has no row).
func queryDailyBasic(ctx context.Context, db *sql.DB, from, to, asOf string) (map[string]int64, LatestDayRow, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT day, reports, servers_sum, templates_sum FROM daily_basic WHERE day >= ? AND day <= ?`, from, to)
	if err != nil {
		return nil, LatestDayRow{}, fmt.Errorf("query daily_basic: %w", err)
	}
	defer func() { _ = rows.Close() }()
	perDay := make(map[string]int64)
	var latest LatestDayRow
	for rows.Next() {
		var day string
		var reports, servers, templates int64
		if err := rows.Scan(&day, &reports, &servers, &templates); err != nil {
			return nil, LatestDayRow{}, fmt.Errorf("scan daily_basic: %w", err)
		}
		perDay[day] = reports
		if day == asOf {
			latest = LatestDayRow{ServersTotal: servers, TemplatesTotal: templates}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, LatestDayRow{}, fmt.Errorf("read daily_basic: %w", err)
	}
	return perDay, latest, nil
}

// denseDays lists every day of the range, oldest first, with 0 for days that
// have no row.
func denseDays(from time.Time, n int, perDay map[string]int64) []DayReports {
	out := make([]DayReports, 0, n)
	for i := range n {
		day := dayString(from.AddDate(0, 0, i))
		out = append(out, DayReports{Day: day, Reports: perDay[day]})
	}
	return out
}

// queryVersions sums daily_version over [from, to] and returns the top
// topVersions versions by reports (ties by label), then Other, then Invalid.
// Other and Invalid are left out when they have no reports.
func queryVersions(ctx context.Context, db *sql.DB, from, to string) ([]VersionRow, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT version, SUM(reports) FROM daily_version WHERE day >= ? AND day <= ? GROUP BY version`, from, to)
	if err != nil {
		return nil, fmt.Errorf("query daily_version: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var named []VersionRow
	var other, invalid int64
	for rows.Next() {
		var version string
		var n int64
		if err := rows.Scan(&version, &n); err != nil {
			return nil, fmt.Errorf("scan daily_version: %w", err)
		}
		switch version {
		case versionInvalidStore:
			invalid += n
		case versionOtherStored:
			other += n
		default:
			named = append(named, VersionRow{Label: version, Reports: n})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read daily_version: %w", err)
	}
	sort.Slice(named, func(i, j int) bool {
		if named[i].Reports != named[j].Reports {
			return named[i].Reports > named[j].Reports
		}
		return named[i].Label < named[j].Label
	})
	if len(named) > topVersions {
		for _, r := range named[topVersions:] {
			other += r.Reports
		}
		named = named[:topVersions]
	}
	out := append([]VersionRow{}, named...)
	if other > 0 {
		out = append(out, VersionRow{Label: versionOtherLabel, Reports: other})
	}
	if invalid > 0 {
		out = append(out, VersionRow{Label: versionInvalidLabel, Reports: invalid})
	}
	return out, nil
}

// queryFleet sums the exact-value histogram of one metric over [from, to],
// then derives the bands and the exact median.
func queryFleet(ctx context.Context, db *sql.DB, metric, from, to string) (FleetDistribution, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT value, SUM(reports) FROM daily_fleet WHERE metric = ? AND day >= ? AND day <= ?
		 GROUP BY value ORDER BY value`, metric, from, to)
	if err != nil {
		return FleetDistribution{}, fmt.Errorf("query daily_fleet %s: %w", metric, err)
	}
	defer func() { _ = rows.Close() }()
	var hist []fleetPoint
	for rows.Next() {
		var p fleetPoint
		if err := rows.Scan(&p.value, &p.reports); err != nil {
			return FleetDistribution{}, fmt.Errorf("scan daily_fleet: %w", err)
		}
		hist = append(hist, p)
	}
	if err := rows.Err(); err != nil {
		return FleetDistribution{}, fmt.Errorf("read daily_fleet: %w", err)
	}
	return FleetDistribution{Bands: fleetBands(hist), Median: fleetMedian(hist)}, nil
}

// fleetPoint is one exact histogram entry; points are sorted by value.
type fleetPoint struct {
	value   int
	reports int64
}

// fleetBands folds an exact histogram into the fixed bands. Every band is
// returned, including empty ones, so a histogram has a stable shape.
func fleetBands(hist []fleetPoint) []FleetBand {
	bands := make([]FleetBand, 0, len(fleetBandBounds)+1)
	for _, bound := range fleetBandBounds {
		bands = append(bands, FleetBand{Le: fmt.Sprint(bound)})
	}
	bands = append(bands, FleetBand{Le: fleetBandOver})
	for _, p := range hist {
		i := len(fleetBandBounds)
		for k, bound := range fleetBandBounds {
			if p.value <= bound {
				i = k
				break
			}
		}
		bands[i].Reports += p.reports
	}
	return bands
}

// fleetMedian returns the exact median of the histogram: the middle value,
// or the mean of the two middle values for an even count. It is 0 for an
// empty histogram. hist must be sorted by value.
func fleetMedian(hist []fleetPoint) float64 {
	var total int64
	for _, p := range hist {
		total += p.reports
	}
	if total == 0 {
		return 0
	}
	at := func(rank int64) int {
		var seen int64
		for _, p := range hist {
			seen += p.reports
			if rank < seen {
				return p.value
			}
		}
		return hist[len(hist)-1].value
	}
	lo, hi := at((total-1)/2), at(total/2)
	return float64(lo+hi) / 2
}
