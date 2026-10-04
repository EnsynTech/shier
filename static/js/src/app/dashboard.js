var campaigns = []
// statuses is a helper map to point result statuses to ui classes
var statuses = {
    "Email Sent": {
        color: "#1abc9c",
        label: "label-success",
        icon: "fa-envelope",
        point: "ct-point-sent"
    },
    "Emails Sent": {
        color: "#1abc9c",
        label: "label-success",
        icon: "fa-envelope",
        point: "ct-point-sent"
    },
    "In progress": {
        label: "label-primary"
    },
    "Queued": {
        label: "label-info"
    },
    "Completed": {
        label: "label-success"
    },
    "Email Opened": {
        color: "#f9bf3b",
        label: "label-warning",
        icon: "fa-envelope",
        point: "ct-point-opened"
    },
    "Email Reported": {
        color: "#45d6ef",
        label: "label-warning",
        icon: "fa-bullhorne",
        point: "ct-point-reported"
    },
    "Email Replied": {
        color: "#9b59b6",
        label: "label-info",
        icon: "fa-reply",
        point: "ct-point-replied"
    },
    "Clicked Link": {
        color: "#F39C12",
        label: "label-clicked",
        icon: "fa-mouse-pointer",
        point: "ct-point-clicked"
    },
    "Success": {
        color: "#f05b4f",
        label: "label-danger",
        icon: "fa-exclamation",
        point: "ct-point-clicked"
    },
    "Error": {
        color: "#6c7a89",
        label: "label-default",
        icon: "fa-times",
        point: "ct-point-error"
    },
    "Error Sending Email": {
        color: "#6c7a89",
        label: "label-default",
        icon: "fa-times",
        point: "ct-point-error"
    },
    "Submitted Data": {
        color: "#f05b4f",
        label: "label-danger",
        icon: "fa-exclamation",
        point: "ct-point-clicked"
    },
    "Unknown": {
        color: "#6c7a89",
        label: "label-default",
        icon: "fa-question",
        point: "ct-point-error"
    },
    "Sending": {
        color: "#428bca",
        label: "label-primary",
        icon: "fa-spinner",
        point: "ct-point-sending"
    },
    "Campaign Created": {
        label: "label-success",
        icon: "fa-rocket"
    }
}

function deleteCampaign(idx) {
    if (confirm(t("js.confirm_delete_campaign") + "\n" + campaigns[idx].name)) {
        api.campaignId.delete(campaigns[idx].id)
            .success(function (data) {
                successFlash(data.message)
                location.reload()
            })
    }
}

function pct(num, den) {
    if (!den) {
        return 0
    }
    return Math.round((num / den) * 1000) / 10
}

function renderWorkbenchHero(campaigns) {
    var sorted = campaigns.slice().sort(function (a, b) {
        return moment.utc(b.created_date).valueOf() - moment.utc(a.created_date).valueOf()
    })
    var windowCampaigns = sorted.slice(0, 8)
    var sent = 0
    var clicked = 0
    var reported = 0
    var active = 0
    $.each(windowCampaigns, function (i, c) {
        sent += (c.stats && c.stats.sent) || 0
        clicked += (c.stats && c.stats.clicked) || 0
        reported += (c.stats && c.stats.email_reported) || 0
    })
    $.each(campaigns, function (i, c) {
        if (c.status === "In progress" || c.status === "Queued" || c.status === "Emails Sent") {
            active += 1
        }
    })
    var prone = pct(clicked, sent)
    $("#phishProneRate").text(prone + "%")
    $("#activeCampaignCount").text(active)
    $("#reportRate").text(pct(reported, sent) + "%")
    if (sorted.length) {
        $("#recentCampaignHint").text(sorted[0].name || "")
    }
    if (prone >= 25) {
        $("#phishProneVerdict").text(t("dashboard.verdict_attention"))
    } else if (prone > 0) {
        $("#phishProneVerdict").text(t("dashboard.verdict_improve"))
    } else {
        $("#phishProneVerdict").text(t("dashboard.verdict_default"))
    }
}

function generateTimelineChart(campaigns) {
    var sorted = campaigns.slice().sort(function (a, b) {
        return moment.utc(a.created_date).valueOf() - moment.utc(b.created_date).valueOf()
    })
    var recent = sorted.slice(-10)
    var categories = []
    var values = []
    var ids = []
    $.each(recent, function (i, campaign) {
        var delivered = (campaign.stats && campaign.stats.sent) || 0
        var clicked = (campaign.stats && campaign.stats.clicked) || 0
        categories.push(campaign.name || ("#" + campaign.id))
        values.push(pct(clicked, delivered))
        ids.push(campaign.id)
    })
    Highcharts.chart("overview_chart", {
        chart: {
            type: "areaspline",
            backgroundColor: "transparent",
            height: 280
        },
        title: { text: null },
        xAxis: {
            categories: categories,
            labels: { rotation: -20 }
        },
        yAxis: {
            min: 0,
            max: 100,
            title: { text: "%" }
        },
        tooltip: {
            formatter: function () {
                return this.x + "<br>" + t("dashboard.phish_prone") + ": <b>" + this.y + "%</b>"
            }
        },
        legend: { enabled: false },
        plotOptions: {
            series: {
                marker: { enabled: true, radius: 3 },
                cursor: "pointer",
                point: {
                    events: {
                        click: function () {
                            var id = ids[this.index]
                            if (id) {
                                window.location.href = "/campaigns/" + id
                            }
                        }
                    }
                }
            }
        },
        credits: { enabled: false },
        series: [{
            data: values,
            color: "#1b7a55",
            fillOpacity: 0.15
        }]
    })
}

function generateDepartmentChart(rows) {
    var data = (rows || []).slice(0, 10)
    if (!$("#dept_chart").length) {
        return
    }
    if (!data.length) {
        $("#dept_chart").html("<p class='text-muted'>" + escapeHtml(t("dashboard.dept_technique_empty")) + "</p>")
        return
    }
    Highcharts.chart("dept_chart", {
        chart: {
            type: "bar",
            backgroundColor: "transparent",
            height: 280
        },
        title: { text: null },
        credits: { enabled: false },
        legend: { enabled: false },
        xAxis: {
            categories: data.map(function (r) {
                var name = r.department
                if (r.sample_small) {
                    name += " (" + t("results.sample_small") + ")"
                }
                return name
            }),
            title: { text: null }
        },
        yAxis: {
            min: 0,
            max: 100,
            title: { text: "%" }
        },
        series: [{
            name: t("results.dept_click_rate"),
            data: data.map(function (r) { return Math.round((r.click_rate || 0) * 1000) / 10 }),
            color: "#1b7a55"
        }]
    })
}

$(document).ready(function () {
    Highcharts.setOptions({
        global: {
            useUTC: false
        }
    })
    api.campaigns.summary()
        .success(function (data) {
            $("#loading").hide()
            campaigns = data.campaigns
            if (campaigns.length > 0) {
                $("#dashboard").show()
                // Create the overview chart data
                campaignTable = $("#campaignTable").DataTable({
                    columnDefs: [{
                            orderable: false,
                            targets: "no-sort"
                        },
                        {
                            className: "color-sent",
                            targets: [2]
                        },
                        {
                            className: "color-opened",
                            targets: [3]
                        },
                        {
                            className: "color-clicked",
                            targets: [4]
                        },
                        {
                            className: "color-success",
                            targets: [5]
                        },
                        {
                            className: "color-reported",
                            targets: [6]
                        }
                    ],
                    order: [
                        [1, "desc"]
                    ]
                });
                campaignRows = []
                $.each(campaigns, function (i, campaign) {
                    var campaign_date = formatDateTime(campaign.created_date)
                    var label = statuses[campaign.status].label || "label-default";
                    //section for tooltips on the status of a campaign to show some quick stats
                    var launchDate;
                    if (moment(campaign.launch_date).isAfter(moment())) {
                        launchDate = t("js.scheduled_to_start") + formatDateTime(campaign.launch_date)
                        var quickStats = launchDate + "<br><br>" + "Number of recipients: " + campaign.stats.total
                    } else {
                        launchDate = t("js.launch_date_at") + formatDateTime(campaign.launch_date)
                        var quickStats = launchDate + "<br><br>" + "Number of recipients: " + campaign.stats.total + "<br><br>" + "Emails opened: " + campaign.stats.opened + "<br><br>" + "Emails clicked: " + campaign.stats.clicked + "<br><br>" + "Submitted Credentials: " + campaign.stats.submitted_data + "<br><br>" + "Errors : " + campaign.stats.error + "<br><br>" + "Reported : " + campaign.stats.email_reported + "<br><br>" + "Replied : " + (campaign.stats.email_replied || 0)
                    }
                    // Add it to the list
                    campaignRows.push([
                        escapeHtml(campaign.name),
                        campaign_date,
                        campaign.stats.sent,
                        campaign.stats.opened,
                        campaign.stats.clicked,
                        campaign.stats.submitted_data,
                        campaign.stats.email_reported,
                        "<span class=\"label " + label + "\" data-toggle=\"tooltip\" data-placement=\"right\" data-html=\"true\" title=\"" + quickStats + "\">" + escapeHtml(statusLabel(campaign.status)) + "</span>",
                        "<div class='pull-right'><a class='btn btn-primary' href='/campaigns/" + campaign.id + "' data-toggle='tooltip' data-placement='left' title='View Results'>\
                    <i class='fa fa-bar-chart'></i>\
                    </a>\
                    <button class='btn btn-danger' onclick='deleteCampaign(" + i + ")' data-toggle='tooltip' data-placement='left' title='Delete Campaign'>\
                    <i class='fa fa-trash-o'></i>\
                    </button></div>"
                    ])
                    $('[data-toggle="tooltip"]').tooltip()
                })
                campaignTable.rows.add(campaignRows).draw()
                renderWorkbenchHero(campaigns)
                generateTimelineChart(campaigns)
                loadDepartmentTechniqueMatrix()
            } else {
                $("#emptyMessage").show()
            }
        })
        .error(function () {
            errorFlash("Error fetching campaigns")
        })
})

function techniqueDisplayName(tech) {
    var key = "scenario_library.technique_" + tech
    var label = t(key)
    return label === key ? tech : label
}

function loadDepartmentTechniqueMatrix() {
    api.campaigns.breakdown()
        .success(function (data) {
            data = data || {}
            generateDepartmentChart(data.by_department)
            renderDepartmentTechniqueMatrix(data)
        })
        .error(function () {
            $("#dept_chart").html("<p class='text-muted'>" + escapeHtml(t("dashboard.dept_technique_error")) + "</p>")
            $("#deptTechniqueTable thead").html("")
            $("#deptTechniqueTable tbody").html(
                "<tr><td class='text-muted'>" + escapeHtml(t("dashboard.dept_technique_error")) + "</td></tr>"
            )
        })
}

function renderDepartmentTechniqueMatrix(data) {
    var cells = data.department_x_technique || []
    if (!cells.length) {
        $("#deptTechniqueTable thead").html("")
        $("#deptTechniqueTable tbody").html(
            "<tr><td class='text-muted'>" + escapeHtml(t("dashboard.dept_technique_empty")) + "</td></tr>"
        )
        return
    }
    var departments = []
    var techniques = []
    var deptSeen = {}
    var techSeen = {}
    var map = {}
    $.each(cells, function (i, cell) {
        if (!deptSeen[cell.department]) {
            deptSeen[cell.department] = true
            departments.push(cell.department)
        }
        if (!techSeen[cell.technique]) {
            techSeen[cell.technique] = true
            techniques.push(cell.technique)
        }
        map[cell.department + "\x00" + cell.technique] = cell
    })
    departments.sort()
    techniques.sort()
    var head = "<tr><th>" + escapeHtml(t("common.department")) + "</th>"
    $.each(techniques, function (i, tech) {
        head += "<th>" + escapeHtml(techniqueDisplayName(tech)) + "</th>"
    })
    head += "</tr>"
    var body = ""
    $.each(departments, function (i, dept) {
        body += "<tr><td>" + escapeHtml(dept) + "</td>"
        $.each(techniques, function (j, tech) {
            var cell = map[dept + "\x00" + tech]
            if (!cell || !cell.total) {
                body += "<td class='text-muted'>—</td>"
            } else {
                var rate = ((cell.click_rate || 0) * 100).toFixed(1) + "%"
                var note = cell.sample_small ? " " + t("results.sample_small") : ""
                body += "<td title='" + cell.clicked + "/" + (cell.delivered || cell.total) + note + "'>" + rate + "</td>"
            }
        })
        body += "</tr>"
    })
    $("#deptTechniqueTable thead").html(head)
    $("#deptTechniqueTable tbody").html(body)
}