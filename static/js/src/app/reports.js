function renderReportCharts() {
    api.campaigns.summary()
        .success(function (data) {
            var campaigns = (data && data.campaigns) ? data.campaigns.slice() : []
            campaigns.sort(function (a, b) {
                return moment.utc(a.created_date).valueOf() - moment.utc(b.created_date).valueOf()
            })
            var recent = campaigns.slice(-8)
            var trendCats = []
            var trendVals = []
            $.each(recent, function (i, c) {
                var total = (c.stats && c.stats.total) || 0
                var clicked = (c.stats && c.stats.clicked) || 0
                trendCats.push(c.name || ("#" + c.id))
                trendVals.push(total ? Math.round((clicked / total) * 1000) / 10 : 0)
            })
            if (trendCats.length && $("#reportTrendChart").length) {
                Highcharts.chart("reportTrendChart", {
                    chart: { type: "areaspline", backgroundColor: "transparent" },
                    title: { text: null },
                    credits: { enabled: false },
                    legend: { enabled: false },
                    xAxis: { categories: trendCats, labels: { rotation: -25 } },
                    yAxis: { min: 0, max: 100, title: { text: "%" } },
                    series: [{
                        name: t("results.phish_prone"),
                        data: trendVals,
                        color: "#1b7a55",
                        fillOpacity: 0.15
                    }]
                })
            }
        })

    api.campaigns.breakdown()
        .success(function (data) {
            var rows = (data && data.by_department) ? data.by_department.slice(0, 8) : []
            if (!rows.length || !$("#reportDeptChart").length) {
                return
            }
            Highcharts.chart("reportDeptChart", {
                chart: { type: "bar", backgroundColor: "transparent" },
                title: { text: null },
                credits: { enabled: false },
                legend: { enabled: false },
                xAxis: {
                    categories: rows.map(function (r) { return r.department }),
                    title: { text: null }
                },
                yAxis: { min: 0, max: 100, title: { text: "%" } },
                series: [{
                    name: t("results.dept_click_rate"),
                    data: rows.map(function (r) { return Math.round((r.click_rate || 0) * 1000) / 10 }),
                    color: "#1b7a55"
                }]
            })
        })
}

$(function () {
    renderReportCharts()
})
