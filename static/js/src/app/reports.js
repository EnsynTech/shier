function formatFraction(rate) {
    return (Math.round((rate || 0) * 1000) / 10) + "%"
}

function reportApiError(xhr) {
    var msg = xhr && xhr.responseJSON && xhr.responseJSON.message
    var map = {
        "brief only includes completed campaigns": "reports.err_not_completed",
        "compare requires 2 to 4 campaigns": "reports.err_compare_count",
        "campaign not found": "reports.err_not_found",
        "ids are required": "reports.err_ids",
        "too many campaigns": "reports.err_too_many",
        "invalid campaign id": "reports.err_invalid_id"
    }
    errorFlash(map[msg] ? t(map[msg]) : (msg || t("reports.err_not_found")))
}

function selectedReportIds() {
    var ids = []
    $("#reportCampaignList input:checked").each(function () {
        ids.push($(this).val())
    })
    return ids
}

function renderReportCampaignList(campaigns) {
    var host = $("#reportCampaignList")
    if (!host.length) {
        return
    }
    var rows = (campaigns || []).slice().sort(function (a, b) {
        return moment.utc(b.launch_date || b.created_date).valueOf() - moment.utc(a.launch_date || a.created_date).valueOf()
    })
    if (!rows.length) {
        $("#reportCampaignEmpty").show()
        return
    }
    host.html(rows.map(function (c) {
        var when = formatDateTime(c.launch_date || c.created_date, false)
        return "<label class='shier-check'>" +
            "<input type='checkbox' value='" + c.id + "' data-status='" + escapeHtml(c.status || "") + "'>" +
            "<span><strong>" + escapeHtml(c.name || ("#" + c.id)) + "</strong>" +
            "<span class='text-muted'> · " + escapeHtml(statusLabel(c.status)) +
            " · " + escapeHtml(sendingTrustLabel(c.sending_trust)) +
            (c.scenario_pack_id ? " · " + escapeHtml(c.scenario_pack_id) : "") +
            (when ? " · " + escapeHtml(when) : "") +
            "</span></span></label>"
    }).join(""))
}

function renderReportCharts() {
    api.campaigns.summary()
        .success(function (data) {
            var campaigns = (data && data.campaigns) ? data.campaigns.slice() : []
            renderReportCampaignList(campaigns)
            campaigns.sort(function (a, b) {
                return moment.utc(a.created_date).valueOf() - moment.utc(b.created_date).valueOf()
            })
            var recent = campaigns.slice(-8)
            var trendCats = []
            var trendVals = []
            $.each(recent, function (i, c) {
                var delivered = (c.stats && c.stats.sent) || 0
                var clicked = (c.stats && c.stats.clicked) || 0
                trendCats.push(c.name || ("#" + c.id))
                trendVals.push(delivered ? Math.round((clicked / delivered) * 1000) / 10 : 0)
            })
            if (trendCats.length && $("#reportTrendChart").length && window.Highcharts) {
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
        .error(function () {
            errorFlash(t("reports.err_not_found"))
        })

    if (!$("#reportDeptChart").length) {
        return
    }
    api.campaigns.breakdown()
        .success(function (data) {
            var rows = (data && data.by_department) ? data.by_department.slice(0, 8) : []
            if (!rows.length || !$("#reportDeptChart").length || !window.Highcharts) {
                return
            }
            Highcharts.chart("reportDeptChart", {
                chart: { type: "bar", backgroundColor: "transparent" },
                title: { text: null },
                credits: { enabled: false },
                legend: { enabled: false },
                xAxis: {
                    categories: rows.map(function (r) {
                        var name = r.department
                        if (r.sample_small) {
                            name += " (" + t("results.sample_small") + ")"
                        }
                        return name
                    }),
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

function briefAdviceText(advice) {
    advice = advice || {}
    var parts = []
    if (advice.weakest_department) {
        parts.push(t("reports.next_retest").replace("{department}", advice.weakest_department))
    } else {
        parts.push(t("reports.next_default"))
    }
    if (advice.no_reports) {
        parts.push(t("reports.next_reports"))
    }
    return parts.join(" ")
}

function renderBrief(data) {
    var root = $("#briefRoot")
    var coverage = data.coverage || {}
    var overall = data.overall || {}
    var trustRows = data.by_sending_trust || []
    var weak = data.weakest_departments || []
    var techs = data.by_technique || []
    var trend = data.trend || []
    var html = ""
    html += "<section class='shier-panel'><h2>" + escapeHtml(t("reports.coverage")) + "</h2>"
    html += "<div class='shier-stats'>"
    html += statBox(t("reports.coverage_campaigns"), coverage.campaign_count || 0)
    html += statBox(t("reports.coverage_delivered"), coverage.delivered || 0)
    html += statBox(t("reports.coverage_departments"), coverage.departments || 0)
    html += "</div>"
    html += "<ul class='shier-brief-list'>"
    $.each(data.campaigns || [], function (i, c) {
        html += "<li>" + escapeHtml(c.name || "") + " · " + escapeHtml(formatDateTime(c.launch_date, false)) +
            " · " + escapeHtml(sendingTrustLabel(c.sending_trust)) +
            (c.scenario_pack_id ? " · " + escapeHtml(c.scenario_pack_id) : "") + "</li>"
    })
    html += "</ul></section>"

    html += "<section class='shier-panel'><h2>" + escapeHtml(t("reports.overall")) + "</h2>"
    html += "<div class='shier-stats'>"
    html += statBox(t("results.phish_prone"), formatFraction(overall.phish_prone))
    html += statBox(t("results.submit_rate"), formatFraction(overall.submit_rate))
    html += statBox(t("results.report_rate"), formatFraction(overall.report_rate))
    html += "</div>"
    html += "<p class='verdict'>" + escapeHtml(briefAdviceText(data.advice)) + "</p></section>"

    html += "<section class='shier-panel'><h2>" + escapeHtml(t("reports.trend_title")) + "</h2>"
    html += "<div id='briefTrendChart' class='shier-chart-box'></div></section>"

    html += "<section class='shier-panel'><h2>" + escapeHtml(t("reports.weakest")) + "</h2>"
    if (!weak.length) {
        html += "<p class='text-muted'>" + escapeHtml(t("reports.weakest_empty")) + "</p>"
    } else {
        html += deptTable(weak)
    }
    html += "</section>"

    html += "<section class='shier-panel'><h2>" + escapeHtml(t("reports.techniques")) + "</h2>"
    html += "<p class='text-muted'>" + escapeHtml(t("reports.techniques_note")) + "</p>"
    html += techTable(techs)
    html += "</section>"

    html += "<section class='shier-panel'><h2>" + escapeHtml(t("reports.trust_rows")) + "</h2>"
    html += "<table class='table table-condensed'><thead><tr><th>" + escapeHtml(t("campaigns.sending_trust")) +
        "</th><th>" + escapeHtml(t("results.dept_delivered")) + "</th><th>" + escapeHtml(t("results.clicked")) +
        "</th><th>" + escapeHtml(t("results.phish_prone")) + "</th></tr></thead><tbody>"
    if (!trustRows.length) {
        html += "<tr><td colspan='4' class='text-muted'>—</td></tr>"
    }
    $.each(trustRows, function (i, row) {
        html += "<tr><td>" + escapeHtml(sendingTrustLabel(row.sending_trust)) + "</td><td>" +
            (row.delivered || 0) + "</td><td>" + (row.clicked || 0) + "</td><td>" +
            formatFraction(row.phish_prone) + "</td></tr>"
    })
    html += "</tbody></table></section>"
    root.html(html)

    if (trend.length && window.Highcharts) {
        Highcharts.chart("briefTrendChart", {
            chart: { type: "areaspline", backgroundColor: "transparent", height: 240 },
            title: { text: null },
            credits: { enabled: false },
            legend: { enabled: false },
            xAxis: {
                categories: trend.map(function (c) { return c.name || ("#" + c.id) }),
                labels: { rotation: -25 }
            },
            yAxis: { min: 0, max: 100, title: { text: "%" } },
            series: [{
                name: t("results.phish_prone"),
                data: trend.map(function (c) { return Math.round((c.phish_prone || 0) * 1000) / 10 }),
                color: "#1b7a55",
                fillOpacity: 0.15
            }]
        })
    }
}

function statBox(label, value) {
    return "<div class='shier-stat'><p class='k'>" + escapeHtml(String(label)) + "</p><p class='v'>" +
        escapeHtml(String(value)) + "</p></div>"
}

function deptTable(rows) {
    var html = "<table class='table table-condensed'><thead><tr><th>" + escapeHtml(t("common.department")) +
        "</th><th>" + escapeHtml(t("results.dept_delivered")) + "</th><th>" + escapeHtml(t("results.dept_clicked")) +
        "</th><th>" + escapeHtml(t("results.phish_prone")) + "</th></tr></thead><tbody>"
    $.each(rows, function (i, row) {
        html += "<tr><td>" + escapeHtml(row.department || "") + "</td><td>" + (row.delivered || 0) +
            "</td><td>" + (row.clicked || 0) + "</td><td>" + formatFraction(row.click_rate) + "</td></tr>"
    })
    return html + "</tbody></table>"
}

function techniqueLabel(tech) {
    var key = "scenario_library.technique_" + tech
    var label = t(key)
    return label === key ? tech : label
}

function techTable(rows) {
    if (!rows.length) {
        return "<p class='text-muted'>—</p>"
    }
    var html = "<table class='table table-condensed'><thead><tr><th>" + escapeHtml(t("reports.techniques")) +
        "</th><th>" + escapeHtml(t("results.dept_delivered")) + "</th><th>" + escapeHtml(t("results.phish_prone")) +
        "</th><th>" + escapeHtml(t("results.sample_small")) + "</th></tr></thead><tbody>"
    $.each(rows, function (i, row) {
        html += "<tr><td>" + escapeHtml(techniqueLabel(row.technique || "")) + "</td><td>" +
            (row.delivered || 0) + "</td><td>" + formatFraction(row.click_rate) + "</td><td>" +
            (row.sample_small ? escapeHtml(t("results.sample_small")) : "") + "</td></tr>"
    })
    return html + "</tbody></table>"
}

function renderCompare(data) {
    var narrative = data.narrative === "retest" ? "retest" : "exposure"
    $("#compareTitle").text(t(narrative === "retest" ? "reports.compare_retest" : "reports.compare_exposure"))
    var html = "<p class='text-muted'>" + escapeHtml(t(narrative === "retest" ? "reports.compare_retest_help" : "reports.compare_exposure_help")) + "</p>"
    html += "<p class='text-muted'>" + escapeHtml(t("reports.compare_note")) + "</p>"
    html += "<div class='table-responsive'><table class='table table-condensed'><thead><tr>"
    ;["reports.launch_date", "common.name", "reports.scenario_pack", "campaigns.sending_trust", "common.status",
        "results.dept_delivered", "results.opened", "results.clicked", "results.submitted", "results.reported", "results.phish_prone"
    ].forEach(function (key) {
        html += "<th>" + escapeHtml(t(key)) + "</th>"
    })
    html += "</tr></thead><tbody>"
    $.each(data.campaigns || [], function (i, c) {
        html += "<tr><td>" + escapeHtml(formatDateTime(c.launch_date, false)) + "</td><td>" + escapeHtml(c.name || "") +
            "</td><td>" + escapeHtml(c.scenario_pack_id || "—") + "</td><td>" + escapeHtml(sendingTrustLabel(c.sending_trust)) +
            "</td><td>" + escapeHtml(statusLabel(c.status)) + "</td><td>" + (c.delivered || 0) +
            "</td><td>" + (c.opened || 0) + "</td><td>" + (c.clicked || 0) + "</td><td>" + (c.submitted || 0) +
            "</td><td>" + (c.reported || 0) + "</td><td>" + formatFraction(c.phish_prone) + "</td></tr>"
    })
    html += "</tbody></table></div>"

    html += "<h2>" + escapeHtml(t("reports.shared_depts")) + "</h2>"
    var shared = data.shared_departments || []
    if (!shared.length) {
        html += "<p class='text-muted'>" + escapeHtml(t("reports.shared_empty")) + "</p>"
    } else {
        html += "<div class='table-responsive'><table class='table table-condensed'><thead><tr><th>" +
            escapeHtml(t("common.department")) + "</th>"
        $.each(data.campaigns || [], function (i, c) {
            html += "<th>" + escapeHtml(c.name || ("#" + c.id)) + "</th>"
        })
        html += "</tr></thead><tbody>"
        $.each(shared, function (i, row) {
            html += "<tr><td>" + escapeHtml(row.department || "") + "</td>"
            $.each(row.cells || [], function (j, cell) {
                var label = (cell.delivered || 0) ? (formatFraction(cell.click_rate) + " (" + cell.clicked + "/" + cell.delivered + ")") : "—"
                if (cell.sample_small) {
                    label += " · " + t("results.sample_small")
                }
                html += "<td>" + escapeHtml(label) + "</td>"
            })
            html += "</tr>"
        })
        html += "</tbody></table></div>"
    }
    $("#compareRoot").html(html)
}

function queryParam(name) {
    var params = new URLSearchParams(window.location.search)
    return params.get(name) || ""
}

$(function () {
    if ($("#reportCampaignList").length) {
        renderReportCharts()
        $("#buildBriefBtn").on("click", function () {
            var ids = selectedReportIds()
            if (!ids.length) {
                errorFlash(t("reports.need_one"))
                return
            }
            var incomplete = false
            $("#reportCampaignList input:checked").each(function () {
                if ($(this).attr("data-status") !== "Completed") {
                    incomplete = true
                }
            })
            if (incomplete) {
                errorFlash(t("reports.need_completed"))
                return
            }
            window.location = "/reports/brief?ids=" + ids.join(",")
        })
        $("#compareBtn").on("click", function () {
            var ids = selectedReportIds()
            if (ids.length < 2 || ids.length > 4) {
                errorFlash(t("reports.need_two"))
                return
            }
            window.location = "/reports/compare?ids=" + ids.join(",")
        })
    }
    if ($("#briefRoot").length) {
        api.campaigns.brief(queryParam("ids"))
            .success(renderBrief)
            .error(function (xhr) {
                $("#briefLoading").hide()
                reportApiError(xhr)
            })
    }
    if ($("#compareRoot").length) {
        api.campaigns.compare(queryParam("ids"))
            .success(renderCompare)
            .error(function (xhr) {
                $("#compareLoading").hide()
                reportApiError(xhr)
            })
    }
})
