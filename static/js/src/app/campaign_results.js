var map = null
var doPoll = true;

function resultCountsAsClicked(status) {
    return status === "Clicked Link" || status === "Submitted Data"
}

function resultCountsAsDelivered(status) {
    return status === "Email Sent" || status === "Email Opened" || status === "Clicked Link" || status === "Submitted Data"
}

function personName(result) {
    var last = (result.last_name || "").trim()
    var first = (result.first_name || "").trim()
    if (window.GophishLang === "zh") {
        return (last + first).trim()
    }
    return (first + " " + last).trim()
}

function formatPct(num, den) {
    if (!den) {
        return "0%"
    }
    return (Math.round((num / den) * 1000) / 10) + "%"
}

function techniqueDisplay(tech) {
    var key = "scenario_library.technique_" + tech
    var label = t(key)
    return label === key ? tech : label
}

function renderCampaignTagBadges(c) {
    var host = $("#campaignTagBadges")
    if (!host.length) {
        return
    }
    var badges = ""
    if (c.difficulty) {
        badges += " <span class='label label-warning'>" + escapeHtml(t("results.difficulty")) +
            " " + escapeHtml(String(c.difficulty)) + "</span>"
    }
    badges += " <span class='label label-success'>" + escapeHtml(sendingTrustLabel(c.sending_trust)) + "</span>"
    $.each(c.techniques || [], function (i, tech) {
        badges += " <span class='label label-info'>" + escapeHtml(techniqueDisplay(tech)) + "</span>"
    })
    $.each(c.industries || [], function (i, ind) {
        badges += " <span class='label label-default'>" + escapeHtml(ind) + "</span>"
    })
    host.html(badges)
}

function renderDepartmentBreakdown(results) {
    var host = $("#departmentBreakdownTable tbody")
    if (!host.length) {
        return
    }
    var buckets = {}
    $.each(results, function (i, result) {
        var dept = (result.department || "").trim() || t("results.department_empty")
        if (!buckets[dept]) {
            buckets[dept] = { delivered: 0, clicked: 0, submitted: 0, reported: 0 }
        }
        if (!resultCountsAsDelivered(result.status)) {
            return
        }
        buckets[dept].delivered++
        if (resultCountsAsClicked(result.status)) {
            buckets[dept].clicked++
        }
        if (result.status === "Submitted Data") {
            buckets[dept].submitted++
        }
        if (result.reported) {
            buckets[dept].reported++
        }
    })
    var rows = Object.keys(buckets).map(function (dept) {
        var b = buckets[dept]
        b.department = dept
        b.sample_small = b.delivered < 5
        b.click_rate = b.delivered ? b.clicked / b.delivered : 0
        return b
    }).sort(function (a, b) {
        if (a.sample_small !== b.sample_small) {
            return a.sample_small ? 1 : -1
        }
        if (a.click_rate !== b.click_rate) {
            return b.click_rate - a.click_rate
        }
        return a.department.localeCompare(b.department)
    })
    var html = rows.map(function (b) {
        var rate = b.delivered ? formatPct(b.clicked, b.delivered) : "—"
        return "<tr><td>" + escapeHtml(b.department) + "</td><td>" + b.delivered + "</td><td>" +
            b.clicked + "</td><td>" + b.submitted + "</td><td>" + b.reported + "</td><td>" +
            rate + "</td><td>" + (b.sample_small ? escapeHtml(t("results.sample_small")) : "") + "</td></tr>"
    }).join("")
    if (!html) {
        html = "<tr><td colspan='7' class='text-muted'>" + escapeHtml(t("results.department_empty_table")) + "</td></tr>"
    }
    host.html(html)
}

function renderHighRisk(results) {
    var host = $("#highRiskTable tbody")
    if (!host.length) {
        return
    }
    var people = (results || []).filter(function (result) {
        return resultCountsAsClicked(result.status)
    }).sort(function (a, b) {
        var aSubmit = a.status === "Submitted Data" ? 0 : 1
        var bSubmit = b.status === "Submitted Data" ? 0 : 1
        if (aSubmit !== bSubmit) {
            return aSubmit - bSubmit
        }
        return (a.email || "").localeCompare(b.email || "")
    })
    if (!people.length) {
        host.html("<tr><td colspan='5' class='text-muted'>" + escapeHtml(t("results.high_risk_empty")) + "</td></tr>")
        return
    }
    host.html(people.map(function (result) {
        return "<tr><td>" + escapeHtml(personName(result)) + "</td><td>" +
            escapeHtml(result.email || "") + "</td><td>" +
            escapeHtml((result.department || "").trim() || t("results.department_empty")) + "</td><td>" +
            escapeHtml(statusLabel(result.status)) + "</td><td>" +
            escapeHtml(result.reported ? t("common.yes") : t("common.no")) + "</td></tr>"
    }).join(""))
}

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
        icon: "fa-envelope-open",
        point: "ct-point-opened"
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
    //not a status, but is used for the campaign timeline and user timeline
    "Email Reported": {
        color: "#45d6ef",
        label: "label-info",
        icon: "fa-bullhorn",
        point: "ct-point-reported"
    },
    "Email Replied": {
        color: "#9b59b6",
        label: "label-info",
        icon: "fa-reply",
        point: "ct-point-replied"
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
    "Retrying": {
        color: "#6c7a89",
        label: "label-default",
        icon: "fa-clock-o",
        point: "ct-point-error"
    },
    "Scheduled": {
        color: "#428bca",
        label: "label-primary",
        icon: "fa-clock-o",
        point: "ct-point-sending"
    },
    "Campaign Created": {
        color: "#1abc9c",
        label: "label-success",
        icon: "fa-rocket",
        point: "ct-point-sent"
    },
    "Campaign Saved as Draft": {
        color: "#95a5a6",
        label: "label-default",
        icon: "fa-file-text-o",
        point: "ct-point-error"
    },
    "Campaign Launched": {
        color: "#1abc9c",
        label: "label-success",
        icon: "fa-rocket",
        point: "ct-point-sent"
    }
}

function timelineStatus(message) {
    var style = statuses[message]
    if (style && style.label) {
        return style
    }
    return {
        color: "#6c7a89",
        label: "label-default",
        icon: "fa-question",
        point: "ct-point-error"
    }
}

var statusMapping = {
    "Email Sent": "sent",
    "Email Opened": "opened",
    "Clicked Link": "clicked",
    "Submitted Data": "submitted_data",
    "Email Reported": "reported",
    "Email Replied": "replied",
}

// This is an underwhelming attempt at an enum
// until I have time to refactor this appropriately.
var progressListing = [
    "Email Sent",
    "Email Opened",
    "Clicked Link",
    "Submitted Data"
]

var campaign = {}
var latestTimelineData = []
var bubbles = []

function dismiss() {
    $("#modal\\.flashes").empty()
    $("#modal").modal('hide')
    $("#resultsTable").dataTable().DataTable().clear().draw()
}

// Deletes a campaign after prompting the user
function deleteCampaign() {
    Swal.fire({
        title: t("js.confirm_delete"),
        text: t("js.confirm_delete_campaign"),
        type: "warning",
        animation: false,
        showCancelButton: true,
        confirmButtonText: t("js.delete"),
        confirmButtonColor: "#428bca",
        reverseButtons: true,
        allowOutsideClick: false,
        showLoaderOnConfirm: true,
        preConfirm: function () {
            return new Promise(function (resolve, reject) {
                api.campaignId.delete(campaign.id)
                    .success(function (msg) {
                        resolve()
                    })
                    .error(function (data) {
                        reject(data.responseJSON.message)
                    })
            })
        }
    }).then(function (result) {
        if (result.value) {
            Swal.fire(
                t("js.campaign_moved_to_trash"),
                t("js.campaign_moved_to_trash_detail"),
                "success"
            ).then(function () {
                location.href = "/campaigns"
            })
        }
    })
}

// Completes a campaign after prompting the user
function completeCampaign() {
    Swal.fire({
        title: t("js.confirm_delete"),
        text: t("js.confirm_complete_campaign"),
        type: "warning",
        animation: false,
        showCancelButton: true,
        confirmButtonText: t("results.complete"),
        confirmButtonColor: "#428bca",
        reverseButtons: true,
        allowOutsideClick: false,
        showLoaderOnConfirm: true,
        preConfirm: function () {
            return new Promise(function (resolve, reject) {
                api.campaignId.complete(campaign.id)
                    .success(function (msg) {
                        resolve()
                    })
                    .error(function (data) {
                        reject(data.responseJSON.message)
                    })
            })
        }
    }).then(function (result) {
        if (result.value){
            Swal.fire(
                t("js.campaign_completed_title"),
                t("js.campaign_completed_text"),
                'success'
            );
            $('#complete_button')[0].disabled = true;
            $('#complete_button').html('<i class="fa fa-flag-checkered"></i> ' + t("js.status_completed"))
            doPoll = false;
        }
    })
}

// Exports campaign results as a CSV file
function exportAsCSV(scope) {
    exportHTML = $("#exportButton").html()
    var csvScope = null
    var filename = campaign.name + ' - ' + capitalize(scope) + '.csv'
    switch (scope) {
        case "results":
            csvScope = (campaign.results || []).map(function (result) {
                var row = {}
                row[t("results.csv_name")] = personName(result)
                row[t("common.email")] = result.email || ""
                row[t("common.department")] = result.department || ""
                row[t("common.position")] = result.position || ""
                row[t("results.csv_status")] = statusLabel(result.status)
                row[t("results.csv_reported")] = result.reported ? t("common.yes") : t("common.no")
                row[t("results.csv_replied")] = result.replied ? t("common.yes") : t("common.no")
                row[t("results.csv_sent_at")] = formatDateTime(result.send_date)
                return row
            })
            break;
        case "events":
            csvScope = campaign.timeline
            break;
    }
    if (!csvScope) {
        return
    }
    $("#exportButton").html('<i class="fa fa-spinner fa-spin"></i>')
    var csvString = Papa.unparse(csvScope, {
        'escapeFormulae': true
    })
    var csvData = new Blob([csvString], {
        type: 'text/csv;charset=utf-8;'
    });
    if (navigator.msSaveBlob) {
        navigator.msSaveBlob(csvData, filename);
    } else {
        var csvURL = window.URL.createObjectURL(csvData);
        var dlLink = document.createElement('a');
        dlLink.href = csvURL;
        dlLink.setAttribute('download', filename)
        document.body.appendChild(dlLink)
        dlLink.click();
        document.body.removeChild(dlLink)
    }
    $("#exportButton").html(exportHTML)
}

function replay(event_idx) {
    request = campaign.timeline[event_idx]
    details = JSON.parse(request.details)
    url = null
    form = $('<form>').attr({
        method: 'POST',
        target: '_blank',
    })
    /* Create a form object and submit it */
    $.each(Object.keys(details.payload), function (i, param) {
        if (param == "rid") {
            return true;
        }
        if (param == "__original_url") {
            url = details.payload[param];
            return true;
        }
        $('<input>').attr({
            name: param,
        }).val(details.payload[param]).appendTo(form);
    })
    /* Ensure we know where to send the user */
    // Prompt for the URL
    Swal.fire({
        title: 'Where do you want the credentials submitted to?',
        input: 'text',
        showCancelButton: true,
        inputPlaceholder: "http://example.com/login",
        inputValue: url || "",
        inputValidator: function (value) {
            return new Promise(function (resolve, reject) {
                if (value) {
                    resolve();
                } else {
                    reject('Invalid URL.');
                }
            });
        }
    }).then(function (result) {
        if (result.value){
            url = result.value
            submitForm()
        }
    })
    return
    submitForm()

    function submitForm() {
        form.attr({
            action: url
        })
        form.appendTo('body').submit().remove()
    }
}

/**
 * Returns an HTML string that displays the OS and browser that clicked the link
 * or submitted credentials.
 * 
 * @param {object} event_details - The "details" parameter for a campaign
 *  timeline event
 * 
 */
var renderDevice = function (event_details) {
    var ua = UAParser(details.browser['user-agent'])
    var detailsString = '<div class="timeline-device-details">'

    var deviceIcon = 'laptop'
    if (ua.device.type) {
        if (ua.device.type == 'tablet' || ua.device.type == 'mobile') {
            deviceIcon = ua.device.type
        }
    }

    var deviceVendor = ''
    if (ua.device.vendor) {
        deviceVendor = ua.device.vendor.toLowerCase()
        if (deviceVendor == 'microsoft') deviceVendor = 'windows'
    }

    var deviceName = 'Unknown'
    if (ua.os.name) {
        deviceName = ua.os.name
        if (deviceName == "Mac OS") {
            deviceVendor = 'apple'
        } else if (deviceName == "Windows") {
            deviceVendor = 'windows'
        }
        if (ua.device.vendor && ua.device.model) {
            deviceName = ua.device.vendor + ' ' + ua.device.model
        }
    }

    if (ua.os.version) {
        deviceName = deviceName + ' (OS Version: ' + ua.os.version + ')'
    }

    deviceString = '<div class="timeline-device-os"><span class="fa fa-stack">' +
        '<i class="fa fa-' + escapeHtml(deviceIcon) + ' fa-stack-2x"></i>' +
        '<i class="fa fa-vendor-icon fa-' + escapeHtml(deviceVendor) + ' fa-stack-1x"></i>' +
        '</span> ' + escapeHtml(deviceName) + '</div>'

    detailsString += deviceString

    var deviceBrowser = 'Unknown'
    var browserIcon = 'info-circle'
    var browserVersion = ''

    if (ua.browser && ua.browser.name) {
        deviceBrowser = ua.browser.name
        // Handle the "mobile safari" case
        deviceBrowser = deviceBrowser.replace('Mobile ', '')
        if (deviceBrowser) {
            browserIcon = deviceBrowser.toLowerCase()
            if (browserIcon == 'ie') browserIcon = 'internet-explorer'
        }
        browserVersion = '(Version: ' + ua.browser.version + ')'
    }

    var browserString = '<div class="timeline-device-browser"><span class="fa fa-stack">' +
        '<i class="fa fa-' + escapeHtml(browserIcon) + ' fa-stack-1x"></i></span> ' +
        deviceBrowser + ' ' + browserVersion + '</div>'

    detailsString += browserString
    detailsString += '</div>'
    return detailsString
}

function renderTimeline(data) {
    record = {
        "id": data[0],
        "first_name": data[2],
        "last_name": data[3],
        "email": data[4],
        "position": data[5],
        "department": data[6],
        "status": data[7],
        "reported": data[8],
        "send_date": data[10]
    }
    results = '<div class="timeline col-sm-12 well well-lg">' +
        '<h6>Timeline for ' + escapeHtml(record.first_name) + ' ' + escapeHtml(record.last_name) +
        '</h6><span class="subtitle">Email: ' + escapeHtml(record.email) +
        '<br>Result ID: ' + escapeHtml(record.id) + '</span>' +
        '<div class="timeline-graph col-sm-6">'
    $.each(campaign.timeline, function (i, event) {
        if (!event.email || event.email == record.email) {
            // Add the event
            results += '<div class="timeline-entry">' +
                '    <div class="timeline-bar"></div>'
            var eventStyle = timelineStatus(event.message)
            results +=
                '    <div class="timeline-icon ' + eventStyle.label + '">' +
                '    <i class="fa ' + eventStyle.icon + '"></i></div>' +
                '    <div class="timeline-message">' + escapeHtml(event.message) +
                '    <span class="timeline-date">' + formatDateTime(moment.utc(event.time).local()) + '</span>'
            if (event.details) {
                details = JSON.parse(event.details)
                if (event.message == "Clicked Link" || event.message == "Submitted Data") {
                    deviceView = renderDevice(details)
                    if (deviceView) {
                        results += deviceView
                    }
                }
                if (event.message == "Submitted Data") {
                    results += '<div class="timeline-replay-button"><button onclick="replay(' + i + ')" class="btn btn-success">'
                    results += '<i class="fa fa-refresh"></i> Replay Credentials</button></div>'
                    results += '<div class="timeline-event-details"><i class="fa fa-caret-right"></i> View Details</div>'
                }
                if (details.payload) {
                    results += '<div class="timeline-event-results">'
                    results += '    <table class="table table-condensed table-bordered table-striped">'
                    results += '        <thead><tr><th>Parameter</th><th>Value(s)</tr></thead><tbody>'
                    $.each(Object.keys(details.payload), function (i, param) {
                        if (param == "rid") {
                            return true;
                        }
                        results += '    <tr>'
                        results += '        <td>' + escapeHtml(param) + '</td>'
                        results += '        <td>' + escapeHtml(details.payload[param]) + '</td>'
                        results += '    </tr>'
                    })
                    results += '       </tbody></table>'
                    results += '</div>'
                }
                if (details.error) {
                    var reasonLabel = sendErrorLabel(details.reason)
                    if (reasonLabel) {
                        results += '<div>' + escapeHtml(reasonLabel) + '</div>'
                    }
                    results += '<div class="timeline-event-details"><i class="fa fa-caret-right"></i> ' + escapeHtml(t("results.view_raw_error")) + '</div>'
                    results += '<div class="timeline-event-results">' + escapeHtml(details.error) + '</div>'
                }
                if (event.message == "Email Reported") {
                    var reportLines = []
                    var sourceLabels = {
                        forward: t("results.report_source_forward"),
                        complaint_reply: t("results.report_source_complaint"),
                        attachment: t("results.report_source_attachment"),
                        manual: t("results.report_source_manual")
                    }
                    if (details.source && sourceLabels[details.source]) {
                        reportLines.push(sourceLabels[details.source])
                    }
                    if (details.from) {
                        reportLines.push(details.from)
                    }
                    if (details.subject) {
                        reportLines.push(details.subject)
                    }
                    if (details.message_id) {
                        reportLines.push(details.message_id)
                    }
                    if (reportLines.length) {
                        results += '<div class="timeline-event-details"><i class="fa fa-caret-right"></i> View Details</div>'
                        results += '<div class="timeline-event-results">' + escapeHtml(reportLines.join(" · ")) + '</div>'
                    }
                }
            }
            results += '</div></div>'
        }
    })
    // Add the scheduled send event at the bottom
    if (record.status == "Scheduled" || record.status == "Retrying") {
        results += '<div class="timeline-entry">' +
            '    <div class="timeline-bar"></div>'
        var scheduleStyle = timelineStatus(record.status)
        results +=
            '    <div class="timeline-icon ' + scheduleStyle.label + '">' +
            '    <i class="fa ' + scheduleStyle.icon + '"></i></div>' +
            '    <div class="timeline-message">' + t("js.scheduled_to_send") + record.send_date + '</span>'
    }
    results += '</div></div>'
    return results
}

var renderTimelineChart = function (chartopts) {
    return Highcharts.chart('timeline_chart', {
        chart: {
            zoomType: 'x',
            type: 'line',
            height: "200px"
        },
        title: {
            text: 'Campaign Timeline'
        },
        xAxis: {
            type: 'datetime',
            dateTimeLabelFormats: window.GophishLang === "zh" ? {
                second: '%H:%M:%S',
                minute: '%H:%M',
                hour: '%H:%M',
                day: '%Y年%m月%d日',
                week: '%Y年%m月%d日',
                month: '%Y年%m月'
            } : {
                second: '%l:%M:%S',
                minute: '%l:%M',
                hour: '%l:%M',
                day: '%b %d, %Y',
                week: '%b %d, %Y',
                month: '%b %Y'
            }
        },
        yAxis: {
            min: 0,
            max: 2,
            visible: false,
            tickInterval: 1,
            labels: {
                enabled: false
            },
            title: {
                text: ""
            }
        },
        tooltip: {
            formatter: function () {
                return Highcharts.dateFormat('%A, %b %d %l:%M:%S %P', new Date(this.x)) +
                    '<br>Event: ' + this.point.message + '<br>Email: <b>' + this.point.email + '</b>'
            }
        },
        legend: {
            enabled: false
        },
        plotOptions: {
            series: {
                marker: {
                    enabled: true,
                    symbol: 'circle',
                    radius: 3
                },
                cursor: 'pointer',
            },
            line: {
                states: {
                    hover: {
                        lineWidth: 1
                    }
                }
            }
        },
        credits: {
            enabled: false
        },
        series: [{
            data: chartopts['data'],
            dashStyle: "shortdash",
            color: "#cccccc",
            lineWidth: 1,
            turboThreshold: 0
        }]
    })
}

/* Renders a pie chart using the provided chartops */
var renderPieChart = function (chartopts) {
    return Highcharts.chart(chartopts['elemId'], {
        chart: {
            type: 'pie',
            events: {
                load: function () {
                    var chart = this,
                        rend = chart.renderer,
                        pie = chart.series[0],
                        left = chart.plotLeft + pie.center[0],
                        top = chart.plotTop + pie.center[1];
                    this.innerText = rend.text(chartopts['data'][0].count, left, top).
                    attr({
                        'text-anchor': 'middle',
                        'font-size': '24px',
                        'font-weight': 'bold',
                        'fill': chartopts['colors'][0],
                        'font-family': 'Helvetica,Arial,sans-serif'
                    }).add();
                },
                render: function () {
                    this.innerText.attr({
                        text: chartopts['data'][0].count
                    })
                }
            }
        },
        title: {
            text: chartopts['title']
        },
        plotOptions: {
            pie: {
                innerSize: '80%',
                dataLabels: {
                    enabled: false
                }
            }
        },
        credits: {
            enabled: false
        },
        tooltip: {
            formatter: function () {
                if (this.key == undefined) {
                    return false
                }
                return '<span style="color:' + this.color + '">\u25CF</span>' + this.point.name + ': <b>' + this.y + '%</b><br/>'
            }
        },
        series: [{
            data: chartopts['data'],
            colors: chartopts['colors'],
        }]
    })
}

function buildEmailSeriesData(results) {
    var email_series_data = {}
    Object.keys(statusMapping).forEach(function (k) {
        email_series_data[k] = 0
    })
    $.each(results || [], function (i, result) {
        email_series_data[result.status]++
        if (result.reported) {
            email_series_data["Email Reported"]++
        }
        if (result.replied) {
            email_series_data["Email Replied"]++
        }
        var step = progressListing.indexOf(result.status)
        for (var j = 0; j < step; j++) {
            email_series_data[progressListing[j]]++
        }
    })
    return email_series_data
}

function renderSendErrorSummary(results, events) {
    var failed = {}
    $.each(results || [], function (_, result) {
        if (result.status === "Error" && result.email) {
            failed[result.email] = "unknown"
        }
    })
    var ordered = (events || []).slice().sort(function (a, b) {
        return new Date(a.time) - new Date(b.time)
    })
    $.each(ordered, function (_, event) {
        if (event.message !== "Error Sending Email" || !Object.prototype.hasOwnProperty.call(failed, event.email)) {
            return
        }
        var details = {}
        try {
            details = JSON.parse(event.details || "{}")
        } catch (e) {
            return
        }
        if (details.reason) {
            failed[event.email] = details.reason
        }
    })
    var counts = {}
    $.each(failed, function (_, reason) {
        counts[reason] = (counts[reason] || 0) + 1
    })
    var reasonOrder = [
        "auth_failed", "tls_cert", "dns", "connect", "rate_limited",
        "mailbox_rejected", "policy_rejected", "gateway_status",
        "invalid_request", "max_attempts", "unknown"
    ]
    var parts = []
    $.each(reasonOrder, function (_, reason) {
        if (!counts[reason]) {
            return
        }
        var label = sendErrorLabel(reason) || reason
        parts.push(escapeHtml(t("results.send_error_count").replace("{n}", counts[reason])) + " " + escapeHtml(label))
    })
    return parts.join(escapeHtml(t("results.send_error_join")))
}

function renderConversionVisuals(results) {
    var total = (results || []).length || 1
    var series = buildEmailSeriesData(results)
    var steps = [
        { key: "Email Sent", label: t("results.sent"), color: "#1abc9c", width: 100 },
        { key: "Email Opened", label: t("results.opened"), color: "#f9bf3b", width: 88 },
        { key: "Clicked Link", label: t("results.clicked"), color: "#F39C12", width: 72 },
        { key: "Submitted Data", label: t("results.submitted"), color: "#f05b4f", width: 56 }
    ]
    var sentBase = series["Email Sent"] || 0
    var html = ""
    $.each(steps, function (i, step) {
        var count = series[step.key] || 0
        var rate = sentBase ? Math.round((count / sentBase) * 1000) / 10 : 0
        var width = sentBase ? (count / sentBase) * 100 : 0
        if (count > 0 && width < 16) {
            width = 16
        }
        html += "<div class='shier-funnel-step' style='width:" + width + "%;background:" + step.color + ";'>" +
            "<span>" + escapeHtml(step.label) + " · " + count + "</span>" +
            "<span class='pct'>" + rate + "%</span></div>"
    })
    $("#resultsFunnel").html(html)

    var sent = series["Email Sent"] || 0
    var opened = series["Email Opened"] || 0
    var clicked = series["Clicked Link"] || 0
    var submitted = series["Submitted Data"] || 0
    var reported = series["Email Reported"] || 0
    var replied = series["Email Replied"] || 0
    var errors = 0
    var retrying = 0
    $.each(results || [], function (i, result) {
        if (result.status === "Error") {
            errors++
        }
        if (result.status === "Retrying") {
            retrying++
        }
    })
    var finished = sent + errors
    if (finished > 0) {
        $("#verdictSendRate").text(formatPct(sent, finished))
    } else {
        $("#verdictSendRate").text("—")
    }
    var notes = []
    if (retrying > 0) {
        notes.push(escapeHtml(t("results.retrying_note").replace("{n}", retrying)))
    }
    var reasonSummary = renderSendErrorSummary(results, (campaign && campaign.timeline) || [])
    if (reasonSummary) {
        notes.push(reasonSummary)
    }
    $("#verdictSendNotes").html(notes.join("<br>"))
    var prone = sent ? Math.round((clicked / sent) * 1000) / 10 : 0
    $("#verdictPhishProne").text(prone + "%")
    $("#verdictSubmitRate").text(formatPct(submitted, sent))
    $("#verdictReportRate").text(formatPct(reported, sent))
    $("#verdictSent").text(sent)
    $("#verdictOpened").text(opened)
    $("#verdictClicked").text(clicked)
    $("#verdictSubmitted").text(submitted)
    $("#verdictReported").text(reported)
    $("#verdictReplied").text(replied)
    $("#verdictErrors").text(errors)
    if (prone >= 25) {
        $("#verdictText").text(t("dashboard.verdict_attention"))
    } else if (prone > 0) {
        $("#verdictText").text(t("dashboard.verdict_improve"))
    } else {
        $("#verdictText").text(t("dashboard.verdict_default"))
    }
}

function ensureResultsMap() {
    if (map) {
        return
    }
    map = new Datamap({
        element: document.getElementById("resultsMap"),
        responsive: true,
        fills: {
            defaultFill: "#ffffff",
            point: "#283F50"
        },
        geographyConfig: {
            highlightFillColor: "#1abc9c",
            borderColor: "#283F50"
        },
        bubblesConfig: {
            borderColor: "#283F50"
        }
    })
    updateMap(campaign.results || [])
}

/* Updates the bubbles on the map

@param {campaign.result[]} results - The campaign results to process
*/
var updateMap = function (results) {
    if (!map) {
        return
    }
    bubbles = []
    $.each(campaign.results, function (i, result) {
        // Check that it wasn't an internal IP
        if (result.latitude == 0 && result.longitude == 0) {
            return true;
        }
        newIP = true
        $.each(bubbles, function (i, bubble) {
            if (bubble.ip == result.ip) {
                bubbles[i].radius += 1
                newIP = false
                return false
            }
        })
        if (newIP) {
            bubbles.push({
                latitude: result.latitude,
                longitude: result.longitude,
                name: result.ip,
                fillKey: "point",
                radius: 2
            })
        }
    })
    map.bubbles(bubbles)
}

/**
 * Creates a status label for use in the results datatable
 * @param {string} status 
 * @param {moment(datetime)} send_date 
 */
function createStatusLabel(status, send_date) {
    var label = timelineStatus(status).label || "label-default";
    var text = escapeHtml(statusLabel(status))
    var statusColumn = "<span class=\"label " + label + "\">" + text + "</span>"
    // Add the tooltip if the email is scheduled to be sent
    if (status == "Scheduled" || status == "Retrying") {
        var sendDateMessage = escapeHtml(t("js.scheduled_to_send") + send_date)
        statusColumn = "<span class=\"label " + label + "\" data-toggle=\"tooltip\" data-placement=\"top\" data-html=\"true\" title=\"" + sendDateMessage + "\">" + text + "</span>"
    }
    return statusColumn
}

/* poll - Queries the API and updates the UI with the results
 *
 * Updates:
 * * Timeline Chart
 * * Email (Donut) Chart
 * * Map Bubbles
 * * Datatables
 */
function poll() {
    api.campaignId.results(campaign.id)
        .success(function (c) {
            campaign = c
            /* Update the timeline */
            var timeline_series_data = []
            $.each(campaign.timeline, function (i, event) {
                var event_date = moment.utc(event.time).local()
                timeline_series_data.push({
                    email: event.email,
                    message: event.message,
                    x: event_date.valueOf(),
                    y: 1,
                    marker: {
                        fillColor: timelineStatus(event.message).color
                    }
                })
            })
            latestTimelineData = timeline_series_data
            var timeline_chart = $("#timeline_chart").highcharts()
            if (timeline_chart) {
                timeline_chart.series[0].update({
                    data: timeline_series_data
                })
            }
            /* Update the results donut chart */
            var email_series_data = buildEmailSeriesData(campaign.results)
            renderConversionVisuals(campaign.results)
            renderDepartmentBreakdown(campaign.results || [])
            renderHighRisk(campaign.results || [])
            // Keep legacy chart ids in sync if Highcharts instances exist
            $.each(email_series_data, function (status, count) {
                if (!(status in statusMapping)) {
                    return true
                }
                var chart = $("#" + statusMapping[status] + "_chart").highcharts()
                if (!chart) {
                    return true
                }
                var email_data = [{
                    name: status,
                    y: Math.floor((count / campaign.results.length) * 100),
                    count: count
                }, {
                    name: '',
                    y: 100 - Math.floor((count / campaign.results.length) * 100)
                }]
                chart.series[0].update({
                    data: email_data
                })
            })

            /* Update the datatable */
            resultsTable = $("#resultsTable").DataTable()
            resultsTable.rows().every(function (i, tableLoop, rowLoop) {
                var row = this.row(i)
                var rowData = row.data()
                var rid = rowData[0]
                $.each(campaign.results, function (j, result) {
                    if (result.id == rid) {
                        rowData[10] = formatDateTime(result.send_date)
                        rowData[9] = result.replied
                        rowData[8] = result.reported
                        rowData[7] = result.status
                        rowData[6] = escapeHtml(result.department || "")
                        resultsTable.row(i).data(rowData)
                        if (row.child.isShown()) {
                            $(row.node()).find("#caret").removeClass("fa-caret-right")
                            $(row.node()).find("#caret").addClass("fa-caret-down")
                            row.child(renderTimeline(row.data()))
                        }
                        return false
                    }
                })
            })
            resultsTable.draw(false)
            /* Update the map information */
            updateMap(campaign.results)
            $('[data-toggle="tooltip"]').tooltip()
            $("#refresh_message").hide()
            $("#refresh_btn").show()
        })
}

function load() {
    campaign.id = window.location.pathname.split('/').slice(-1)[0]
    api.campaignId.results(campaign.id)
        .success(function (c) {
            campaign = c
            if (campaign) {
                $("title").text(c.name + " - " + t("brand.name"))
                $("#loading").hide()
                $("#campaignResults").show()
                // Set the title
                $("#page-title").text(t("js.results_for") + " " + c.name)
                if (c.status == "Completed") {
                    $('#complete_button')[0].disabled = true;
                    $('#complete_button').html('<i class="fa fa-flag-checkered"></i> ' + t("js.status_completed"));
                    doPoll = false;
                }
                // Setup viewing the details of a result
                $("#resultsTable").on("click", ".timeline-event-details", function () {
                    // Show the parameters
                    payloadResults = $(this).parent().find(".timeline-event-results")
                    if (payloadResults.is(":visible")) {
                        $(this).find("i").removeClass("fa-caret-down")
                        $(this).find("i").addClass("fa-caret-right")
                        payloadResults.hide()
                    } else {
                        $(this).find("i").removeClass("fa-caret-right")
                        $(this).find("i").addClass("fa-caret-down")
                        payloadResults.show()
                    }
                })
                // Setup the results table
                resultsTable = $("#resultsTable").DataTable({
                    destroy: true,
                    autoWidth: false,
                    scrollX: true,
                    "order": [
                        [2, "asc"]
                    ],
                    columnDefs: [{
                            orderable: false,
                            targets: "no-sort"
                        }, {
                            className: "details-control",
                            "targets": [1]
                        },                         {
                            "visible": false,
                            "targets": [0, 10]
                        },
                        {
                            "render": function (data, type, row) {
                                return createStatusLabel(data, row[10])
                            },
                            "targets": [7]
                        },
                        {
                            className: "text-center",
                            "render": function (reported, type, row) {
                                if (type == "display") {
                                    if (reported) {
                                        return "<i class='fa fa-check-circle text-center text-success'></i>"
                                    }
                                    return "<i role='button' class='fa fa-times-circle text-center text-muted' onclick='report_mail(\"" + row[0] + "\", \"" + campaign.id + "\");'></i>"
                                }
                                return reported
                            },
                            "targets": [8]
                        },
                        {
                            className: "text-center",
                            "render": function (replied, type, row) {
                                if (type == "display") {
                                    if (replied) {
                                        return "<i class='fa fa-check-circle text-center text-success'></i>"
                                    }
                                    return "<i class='fa fa-times-circle text-center text-muted'></i>"
                                }
                                return replied
                            },
                            "targets": [9]
                        }
                    ]
                });
                resultsTable.clear();
                var timeline_series_data = []
                $.each(campaign.results, function (i, result) {
                    resultsTable.row.add([
                        result.id,
                        "<i id=\"caret\" class=\"fa fa-caret-right\"></i>",
                        escapeHtml(result.first_name) || "",
                        escapeHtml(result.last_name) || "",
                        escapeHtml(result.email) || "",
                        escapeHtml(result.position) || "",
                        escapeHtml(result.department) || "",
                        result.status,
                        result.reported,
                        result.replied,
                        formatDateTime(result.send_date)
                    ])
                })
                resultsTable.draw();
                renderCampaignTagBadges(campaign)
                renderDepartmentBreakdown(campaign.results || [])
                renderHighRisk(campaign.results || [])
                renderConversionVisuals(campaign.results || [])
                // Setup tooltips
                $('[data-toggle="tooltip"]').tooltip()
                // Setup the individual timelines
                $('#resultsTable tbody').on('click', 'td.details-control', function () {
                    var tr = $(this).closest('tr');
                    var row = resultsTable.row(tr);
                    if (row.child.isShown()) {
                        // This row is already open - close it
                        row.child.hide();
                        tr.removeClass('shown');
                        $(this).find("i").removeClass("fa-caret-down")
                        $(this).find("i").addClass("fa-caret-right")
                    } else {
                        // Open this row
                        $(this).find("i").removeClass("fa-caret-right")
                        $(this).find("i").addClass("fa-caret-down")
                        row.child(renderTimeline(row.data())).show();
                        tr.addClass('shown');
                    }
                });
                // Setup the graphs
                $.each(campaign.timeline, function (i, event) {
                    if (event.message == "Campaign Created") {
                        return true
                    }
                    var event_date = moment.utc(event.time).local()
                    timeline_series_data.push({
                        email: event.email,
                        message: event.message,
                        x: event_date.valueOf(),
                        y: 1,
                        marker: {
                            fillColor: timelineStatus(event.message).color
                        }
                    })
                })
                latestTimelineData = timeline_series_data
                $("#resultsDetails").hide()
                $("#toggleResultsDetails").off("click").on("click", function () {
                    var $details = $("#resultsDetails")
                    if ($details.is(":visible")) {
                        $details.hide()
                        $(this).text(t("results.details_toggle"))
                        return
                    }
                    $details.show()
                    $(this).text(t("results.details_hide"))
                    if (!$("#timeline_chart").highcharts()) {
                        renderTimelineChart({ data: latestTimelineData })
                    } else {
                        $("#timeline_chart").highcharts().reflow()
                    }
                    if (resultsTable) {
                        resultsTable.columns.adjust()
                    }
                })

                $("#resultsMapContainer").hide()
                $("#toggleResultsMap").off("click").on("click", function () {
                    var $container = $("#resultsMapContainer")
                    if ($container.is(":visible")) {
                        $container.hide()
                        $(this).text(t("results.map_toggle"))
                    } else {
                        $container.show()
                        ensureResultsMap()
                        $(this).text(t("results.map_hide"))
                    }
                })
            }
        })
        .error(function () {
            $("#loading").hide()
            errorFlash(" Campaign not found!")
        })
}

var setRefresh

function refresh() {
    if (!doPoll) {
        return;
    }
    $("#refresh_message").show()
    $("#refresh_btn").hide()
    poll()
    clearTimeout(setRefresh)
    setRefresh = setTimeout(refresh, 60000)
};

function report_mail(rid, cid) {
    Swal.fire({
        title: t("js.confirm_report"),
        text: t("js.confirm_report_text"),
        type: "question",
        animation: false,
        showCancelButton: true,
        confirmButtonText: t("js.ok"),
        confirmButtonColor: "#428bca",
        reverseButtons: true,
        allowOutsideClick: false,
        showLoaderOnConfirm: true
    }).then(function (result) {
        if (result.value) {
            api.campaignId.report(cid, rid)
                .success(function () {
                    refresh();
                })
                .error(function () {
                    Swal.fire({
                        title: t("js.report_failed"),
                        type: "error",
                        confirmButtonText: "Close"
                    });
                });
        }
    })
}

$(document).ready(function () {
    Highcharts.setOptions({
        global: {
            useUTC: false
        }
    })
    load();

    // Start the polling loop
    setRefresh = setTimeout(refresh, 60000)
})
