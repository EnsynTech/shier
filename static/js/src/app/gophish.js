function errorFlash(message) {
    $("#flashes").empty()
    $("#flashes").append("<div style=\"text-align:center\" class=\"alert alert-danger\">\
        <i class=\"fa fa-exclamation-circle\"></i> " + message + "</div>")
}

function successFlash(message) {
    $("#flashes").empty()
    $("#flashes").append("<div style=\"text-align:center\" class=\"alert alert-success\">\
        <i class=\"fa fa-check-circle\"></i> " + message + "</div>")
}

// Fade message after n seconds
function errorFlashFade(message, fade) {
    $("#flashes").empty()
    $("#flashes").append("<div style=\"text-align:center\" class=\"alert alert-danger\">\
        <i class=\"fa fa-exclamation-circle\"></i> " + message + "</div>")
    setTimeout(function(){ 
        $("#flashes").empty() 
    }, fade * 1000);
}
// Fade message after n seconds
function successFlashFade(message, fade) {  
    $("#flashes").empty()
    $("#flashes").append("<div style=\"text-align:center\" class=\"alert alert-success\">\
        <i class=\"fa fa-check-circle\"></i> " + message + "</div>")
    setTimeout(function(){ 
        $("#flashes").empty() 
    }, fade * 1000);

}

function modalError(message) {
    $("#modal\\.flashes").empty().append("<div style=\"text-align:center\" class=\"alert alert-danger\">\
        <i class=\"fa fa-exclamation-circle\"></i> " + message + "</div>")
}

function query(endpoint, method, data, async) {
    return $.ajax({
        url: "/api" + endpoint,
        async: async,
        method: method,
        data: JSON.stringify(data),
        dataType: "json",
        contentType: "application/json",
        beforeSend: function (xhr) {
            xhr.setRequestHeader('Authorization', 'Bearer ' + user.api_key);
        }
    })
}

function escapeHtml(text) {
    return $("<div/>").text(text).html()
}
window.escapeHtml = escapeHtml

/** Translate a UI string key; falls back to the key if missing. */
function t(key) {
    if (window.GophishTranslations && window.GophishTranslations[key]) {
        return window.GophishTranslations[key]
    }
    return key
}
window.t = t

// Display timestamps follow the page language. Moment in this build has no
// Chinese locale, so Chinese uses numeric tokens that do not need one.
function dateTimeFormat(withSeconds) {
    if (window.GophishLang === "zh") {
        return withSeconds === false ? "YYYY年M月D日 HH:mm" : "YYYY年M月D日 HH:mm:ss"
    }
    return withSeconds === false ? "MMMM Do YYYY, h:mm a" : "MMMM Do YYYY, h:mm:ss a"
}

function formatDateTime(value, withSeconds) {
    if (value == null || value === "") {
        return ""
    }
    var m = moment.isMoment(value) ? value.clone() : moment(value)
    if (!m.isValid()) {
        return ""
    }
    return m.format(dateTimeFormat(withSeconds))
}
window.dateTimeFormat = dateTimeFormat
window.formatDateTime = formatDateTime

/** Translate API status strings for display. Keys stay English for matching. */
function sendingTrustLabel(value) {
    if (value === "customer_domain") {
        return t("campaigns.trust_customer")
    }
    if (value === "lookalike") {
        return t("campaigns.trust_lookalike")
    }
    return t("campaigns.trust_unspecified")
}
window.sendingTrustLabel = sendingTrustLabel

function statusLabel(status) {
    var map = {
        "Email Sent": "js.status_email_sent",
        "Emails Sent": "js.email_sent",
        "Email Opened": "js.status_email_opened",
        "Clicked Link": "js.status_clicked",
        "Submitted Data": "js.status_submitted",
        "Email Reported": "js.status_reported",
        "Email Replied": "js.status_replied",
        "Error": "js.status_error",
        "Scheduled": "js.status_scheduled",
        "In progress": "js.status_in_progress",
        "Completed": "js.status_completed",
        "Queued": "js.status_queued",
        "Sending": "js.status_sending",
        "Retrying": "js.status_retrying"
    }
    if (map[status]) {
        return t(map[status])
    }
    return status
}
window.statusLabel = statusLabel

function sendErrorLabel(reason) {
    if (!reason) {
        return ""
    }
    var key = "send_error." + reason
    var label = t(key)
    if (label === key) {
        return ""
    }
    return label
}
window.sendErrorLabel = sendErrorLabel

function formatSendFailure(data) {
    var body = (data && data.responseJSON) || {}
    var reason = body.data && body.data.reason
    var raw = body.message || ""
    var label = sendErrorLabel(reason)
    var html = ""
    if (label) {
        html += "<div>" + escapeHtml(label) + "</div>"
    }
    if (raw) {
        html += "<div class='text-muted' style='margin-top:6px;'>" + escapeHtml(raw) + "</div>"
    }
    if (!html) {
        html = escapeHtml(t("send_error.unknown"))
    }
    return html
}
window.formatSendFailure = formatSendFailure

function unescapeHtml(html) {
    return $("<div/>").html(html).text()
}

/**
 * 
 * @param {string} string - The input string to capitalize
 * 
 */
var capitalize = function (string) {
    return string.charAt(0).toUpperCase() + string.slice(1);
}

/*
Define our API Endpoints
*/
var api = {
    // campaigns contains the endpoints for /campaigns
    campaigns: {
        // get() - Queries the API for GET /campaigns
        get: function () {
            return query("/campaigns/", "GET", {}, false)
        },
        // post() - Posts a campaign to POST /campaigns
        post: function (data) {
            return query("/campaigns/", "POST", data, false)
        },
        // summary() - Queries the API for GET /campaigns/summary
        summary: function () {
            return query("/campaigns/summary", "GET", {}, false)
        },
        // breakdown() - Queries the API for GET /campaigns/breakdown
        breakdown: function () {
            return query("/campaigns/breakdown", "GET", {}, false)
        },
        brief: function (ids) {
            return query("/campaigns/brief?ids=" + encodeURIComponent(ids), "GET", {}, false)
        },
        compare: function (ids) {
            return query("/campaigns/compare?ids=" + encodeURIComponent(ids), "GET", {}, false)
        },
        // trash() - Queries the API for GET /campaigns/trash
        trash: function () {
            return query("/campaigns/trash", "GET", {}, false)
        }
    },
    // campaignId contains the endpoints for /campaigns/:id
    campaignId: {
        // get() - Queries the API for GET /campaigns/:id
        get: function (id) {
            return query("/campaigns/" + id, "GET", {}, true)
        },
        // delete() - Soft-deletes a campaign at DELETE /campaigns/:id
        delete: function (id) {
            return query("/campaigns/" + id, "DELETE", {}, false)
        },
        // restore() - Restores a campaign from trash at POST /campaigns/:id/restore
        restore: function (id) {
            return query("/campaigns/" + id + "/restore", "POST", {}, false)
        },
        // purge() - Permanently deletes a trashed campaign at DELETE /campaigns/:id/purge
        purge: function (id) {
            return query("/campaigns/" + id + "/purge", "DELETE", { confirm: "purge" }, false)
        },
        // results() - Queries the API for GET /campaigns/:id/results
        results: function (id) {
            return query("/campaigns/" + id + "/results", "GET", {}, true)
        },
        // complete() - Completes a campaign at POST /campaigns/:id/complete
        complete: function (id) {
            return query("/campaigns/" + id + "/complete", "GET", {}, true)
        },
        // summary() - Queries the API for GET /campaigns/summary
        summary: function (id) {
            return query("/campaigns/" + id + "/summary", "GET", {}, true)
        },
        // report() - Records a manual report at POST /campaigns/:id/results/:rid/report
        report: function (id, rid) {
            return query("/campaigns/" + id + "/results/" + rid + "/report", "POST", {}, false)
        }
    },
    // groups contains the endpoints for /groups
    groups: {
        // get() - Queries the API for GET /groups
        get: function () {
            return query("/groups/", "GET", {}, false)
        },
        // post() - Posts a group to POST /groups
        post: function (group) {
            return query("/groups/", "POST", group, false)
        },
        // summary() - Queries the API for GET /groups/summary
        summary: function () {
            return query("/groups/summary", "GET", {}, true)
        }
    },
    // groupId contains the endpoints for /groups/:id
    groupId: {
        // get() - Queries the API for GET /groups/:id
        get: function (id) {
            return query("/groups/" + id, "GET", {}, false)
        },
        // put() - Puts a group to PUT /groups/:id
        put: function (group) {
            return query("/groups/" + group.id, "PUT", group, false)
        },
        // delete() - Deletes a group at DELETE /groups/:id
        delete: function (id) {
            return query("/groups/" + id, "DELETE", {}, false)
        }
    },
    // templates contains the endpoints for /templates
    templates: {
        // get() - Queries the API for GET /templates
        get: function () {
            return query("/templates/", "GET", {}, false)
        },
        // post() - Posts a template to POST /templates
        post: function (template) {
            return query("/templates/", "POST", template, false)
        }
    },
    // templateId contains the endpoints for /templates/:id
    templateId: {
        // get() - Queries the API for GET /templates/:id
        get: function (id) {
            return query("/templates/" + id, "GET", {}, false)
        },
        // put() - Puts a template to PUT /templates/:id
        put: function (template) {
            return query("/templates/" + template.id, "PUT", template, false)
        },
        // delete() - Deletes a template at DELETE /templates/:id
        delete: function (id) {
            return query("/templates/" + id, "DELETE", {}, false)
        }
    },
    // pages contains the endpoints for /pages
    pages: {
        // get() - Queries the API for GET /pages
        get: function () {
            return query("/pages/", "GET", {}, false)
        },
        // post() - Posts a page to POST /pages
        post: function (page) {
            return query("/pages/", "POST", page, false)
        }
    },
    // scenarioPacks contains the endpoints for /scenario_packs
    scenarioPacks: {
        get: function () {
            return query("/scenario_packs/", "GET", {}, false)
        },
        import: function (id) {
            return query("/scenario_packs/" + encodeURIComponent(id) + "/import", "POST", {}, false)
        }
    },
    // pageId contains the endpoints for /pages/:id
    pageId: {
        // get() - Queries the API for GET /pages/:id
        get: function (id) {
            return query("/pages/" + id, "GET", {}, false)
        },
        // put() - Puts a page to PUT /pages/:id
        put: function (page) {
            return query("/pages/" + page.id, "PUT", page, false)
        },
        // delete() - Deletes a page at DELETE /pages/:id
        delete: function (id) {
            return query("/pages/" + id, "DELETE", {}, false)
        }
    },
    // SMTP contains the endpoints for /smtp
    SMTP: {
        // get() - Queries the API for GET /smtp
        get: function () {
            return query("/smtp/", "GET", {}, false)
        },
        // post() - Posts a SMTP to POST /smtp
        post: function (smtp) {
            return query("/smtp/", "POST", smtp, false)
        }
    },
    // SMTPId contains the endpoints for /smtp/:id
    SMTPId: {
        // get() - Queries the API for GET /smtp/:id
        get: function (id) {
            return query("/smtp/" + id, "GET", {}, false)
        },
        // put() - Puts a SMTP to PUT /smtp/:id
        put: function (smtp) {
            return query("/smtp/" + smtp.id, "PUT", smtp, false)
        },
        // delete() - Deletes a SMTP at DELETE /smtp/:id
        delete: function (id) {
            return query("/smtp/" + id, "DELETE", {}, false)
        }
    },
    sendingChannels: {
        get: function () {
            return query("/sending_channels/", "GET", {}, false)
        },
        post: function (channel) {
            return query("/sending_channels/", "POST", channel, false)
        },
        test: function (payload) {
            return query("/sending_channels/test", "POST", payload, true)
        }
    },
    sendingChannelId: {
        get: function (id) {
            return query("/sending_channels/" + id, "GET", {}, false)
        },
        put: function (channel) {
            return query("/sending_channels/" + channel.id, "PUT", channel, false)
        },
        delete: function (id) {
            return query("/sending_channels/" + id, "DELETE", {}, false)
        },
        test: function (id, payload) {
            return query("/sending_channels/" + id + "/test", "POST", payload, true)
        }
    },
    // IMAP containts the endpoints for /imap/
    IMAP: {
        get: function() {
            return query("/imap/", "GET", {}, !1)
        },
        post: function(e) {
            return query("/imap/", "POST", e, !1)
        },
        validate: function(e) {
            return query("/imap/validate", "POST", e, true)
        }
    },
    // users contains the endpoints for /users
    users: {
        // get() - Queries the API for GET /users
        get: function () {
            return query("/users/", "GET", {}, true)
        },
        // post() - Posts a user to POST /users
        post: function (user) {
            return query("/users/", "POST", user, true)
        }
    },
    // userId contains the endpoints for /users/:id
    userId: {
        // get() - Queries the API for GET /users/:id
        get: function (id) {
            return query("/users/" + id, "GET", {}, true)
        },
        // put() - Puts a user to PUT /users/:id
        put: function (user) {
            return query("/users/" + user.id, "PUT", user, true)
        },
        // delete() - Deletes a user at DELETE /users/:id
        delete: function (id) {
            return query("/users/" + id, "DELETE", {}, true)
        }
    },
    webhooks: {
        get: function() {
            return query("/webhooks/", "GET", {}, false)
        },
        post: function(webhook) {
            return query("/webhooks/", "POST", webhook, false)
        },
    },
    webhookId: {
        get: function(id) {
            return query("/webhooks/" + id, "GET", {}, false)
        },
        put: function(webhook) {
            return query("/webhooks/" + webhook.id, "PUT", webhook, true)
        },
        delete: function(id) {
            return query("/webhooks/" + id, "DELETE", {}, false)
        },
        ping: function(id) {
            return query("/webhooks/" + id + "/validate", "POST", {}, true)
        },
    },
    // import handles all of the "import" functions in the api
    import_email: function (req) {
        return query("/import/email", "POST", req, false)
    },
    // clone_site handles importing a site by url
    clone_site: function (req) {
        return query("/import/site", "POST", req, false)
    },
    // send_test_email sends an email to the specified email address
    send_test_email: function (req) {
        return query("/util/send_test_email", "POST", req, true)
    },
    smtp_check: function (req) {
        return query("/util/smtp_check", "POST", req, true)
    },
    reset: function () {
        return query("/reset", "POST", {}, true)
    }
}
window.api = api

/**
 * Opens the scenario pack library modal and loads pack metadata.
 * @param {function} [onImported] - Optional callback after a successful import.
 */
function openScenarioLibrary(onImported) {
    window._scenarioLibraryOnImported = onImported
    $("#scenarioLibraryList").html(
        "<div class='text-center'><i class='fa fa-spinner fa-spin fa-2x'></i></div>"
    )
    $("#scenarioLibraryModal").modal("show")
    api.scenarioPacks.get()
        .success(function (packs) {
            window._scenarioLibraryPacks = packs || []
            renderScenarioLibraryFilters()
            renderScenarioLibrary(window._scenarioLibraryPacks)
        })
        .error(function (data) {
            var msg = (data.responseJSON && data.responseJSON.message) || t("scenario_library.load_error")
            $("#scenarioLibraryList").html(
                "<div class='alert alert-danger'>" + escapeHtml(msg) + "</div>"
            )
        })
}
window.openScenarioLibrary = openScenarioLibrary

function techniqueLabel(tech) {
    var key = "scenario_library.technique_" + tech
    var label = t(key)
    return label === key ? tech : label
}

function categoryLabel(category) {
    var key = "scenario_library.category_" + category
    var label = t(key)
    return label === key ? category : label
}

function renderScenarioLibraryFilters() {
    var packs = window._scenarioLibraryPacks || []
    var industries = {}
    $.each(packs, function (i, pack) {
        $.each(pack.industries || [], function (j, ind) {
            industries[ind] = true
        })
    })
    var industryOptions = "<option value=''>" + escapeHtml(t("scenario_library.filter_all_industries")) + "</option>"
    $.each(Object.keys(industries).sort(), function (i, ind) {
        industryOptions += "<option value='" + escapeHtml(ind) + "'>" + escapeHtml(ind) + "</option>"
    })
    var techOptions = [
        "", "link", "attachment", "qr", "sms", "im", "calendar", "reply"
    ].map(function (tech) {
        if (!tech) {
            return "<option value=''>" + escapeHtml(t("scenario_library.filter_all_techniques")) + "</option>"
        }
        return "<option value='" + tech + "'>" + escapeHtml(techniqueLabel(tech)) + "</option>"
    }).join("")
    var difficultyOptions = "<option value=''>" + escapeHtml(t("scenario_library.filter_all_difficulty")) + "</option>"
    for (var d = 1; d <= 5; d++) {
        difficultyOptions += "<option value='" + d + "'>" + d + "</option>"
    }
    var html = "<div class='row' style='margin-bottom:12px;' id='scenarioLibraryFilters'>" +
        "<div class='col-sm-4'><label class='control-label'>" + escapeHtml(t("scenario_library.filter_technique")) +
        "</label><select class='form-control input-sm' id='scenarioFilterTechnique' onchange='filterScenarioLibrary()'>" +
        techOptions + "</select></div>" +
        "<div class='col-sm-4'><label class='control-label'>" + escapeHtml(t("scenario_library.filter_industry")) +
        "</label><select class='form-control input-sm' id='scenarioFilterIndustry' onchange='filterScenarioLibrary()'>" +
        industryOptions + "</select></div>" +
        "<div class='col-sm-4'><label class='control-label'>" + escapeHtml(t("scenario_library.filter_difficulty")) +
        "</label><select class='form-control input-sm' id='scenarioFilterDifficulty' onchange='filterScenarioLibrary()'>" +
        difficultyOptions + "</select></div></div>"
    $("#scenarioLibraryFiltersHost").html(html)
}
window.renderScenarioLibraryFilters = renderScenarioLibraryFilters

function filterScenarioLibrary() {
    var packs = window._scenarioLibraryPacks || []
    var tech = $("#scenarioFilterTechnique").val() || ""
    var industry = $("#scenarioFilterIndustry").val() || ""
    var difficulty = $("#scenarioFilterDifficulty").val() || ""
    var filtered = packs.filter(function (pack) {
        if (tech && (!(pack.techniques || []).length || pack.techniques.indexOf(tech) === -1)) {
            return false
        }
        if (industry && (!(pack.industries || []).length || pack.industries.indexOf(industry) === -1)) {
            return false
        }
        if (difficulty && String(pack.difficulty || 0) !== String(difficulty)) {
            return false
        }
        return true
    })
    renderScenarioLibrary(filtered)
}
window.filterScenarioLibrary = filterScenarioLibrary

function renderScenarioLibrary(packs) {
    var categories = {
        link: { label: t("scenario_library.category_link"), items: [] },
        calendar: { label: t("scenario_library.category_calendar"), items: [] },
        qr: { label: t("scenario_library.category_qr"), items: [] },
        office: { label: t("scenario_library.category_office"), items: [] },
        conversation: { label: t("scenario_library.category_conversation"), items: [] },
        sms: { label: t("scenario_library.category_sms"), items: [] },
        im: { label: t("scenario_library.category_im"), items: [] }
    }
    $.each(packs, function (i, pack) {
        if (categories[pack.category]) {
            categories[pack.category].items.push(pack)
        } else {
            if (!categories.other) {
                categories.other = { label: pack.category, items: [] }
            }
            categories.other.items.push(pack)
        }
    })
    var html = ""
    $.each(categories, function (key, group) {
        if (!group.items.length) {
            return
        }
        html += "<h4 style='margin-top:16px;'>" + escapeHtml(group.label) + "</h4>"
        $.each(group.items, function (j, pack) {
            var badges = ""
            if (pack.difficulty) {
                badges += " <span class='label label-warning'>D" + escapeHtml(String(pack.difficulty)) + "</span>"
            }
            $.each(pack.techniques || [], function (k, tech) {
                badges += " <span class='label label-info'>" + escapeHtml(techniqueLabel(tech)) + "</span>"
            })
            if (pack.has_attachment) {
                var attBadge = "Att"
                if (pack.category === "calendar") {
                    attBadge = ".ics"
                } else if (pack.category === "office") {
                    attBadge = "Office"
                } else if (pack.category === "qr") {
                    attBadge = "QR file"
                }
                badges += " <span class='label label-default'>" + attBadge + "</span>"
            }
            if (pack.has_qr) {
                badges += " <span class='label label-primary'>QR</span>"
            }
            var industries = (pack.industries || []).join(", ")
            html += "<div class='panel panel-default' style='margin-bottom:10px;'>" +
                "<div class='panel-body' style='padding:12px 15px;'>" +
                "<div class='row'>" +
                "<div class='col-sm-9'>" +
                "<strong>" + escapeHtml(pack.title) + "</strong>" + badges +
                "<p class='text-muted' style='margin:6px 0 0;'>" + escapeHtml(pack.description) + "</p>" +
                (industries ? "<p class='text-muted' style='margin:4px 0 0;font-size:12px;'>" +
                    escapeHtml(t("scenario_library.industries")) + ": " + escapeHtml(industries) + "</p>" : "") +
                "</div>" +
                "<div class='col-sm-3 text-right' style='padding-top:8px;'>" +
                "<button type='button' class='btn btn-primary btn-sm' onclick='importScenarioPack(" +
                JSON.stringify(pack.id) + ")'>" + t("scenario_library.import") + "</button>" +
                "</div></div></div></div>"
        })
    })
    if (!html) {
        html = "<div class='alert alert-info'>" + t("scenario_library.empty") + "</div>"
    }
    $("#scenarioLibraryList").html(html)
}
window.renderScenarioLibrary = renderScenarioLibrary

function importScenarioPack(id) {
    api.scenarioPacks.import(id)
        .success(function (result) {
            $("#scenarioLibraryModal").modal("hide")
            var msg = t("scenario_library.import_success")
                .replace("{template}", result.template_name || "")
                .replace("{page}", result.page_name || "")
            successFlash(msg)
            if (typeof window._scenarioLibraryOnImported === "function") {
                window._scenarioLibraryOnImported(result)
            }
        })
        .error(function (data) {
            var msg = (data.responseJSON && data.responseJSON.message) || t("scenario_library.import_error")
            errorFlash(msg)
        })
}
window.importScenarioPack = importScenarioPack

// Register our moment.js datatables listeners
$(document).ready(function () {
    // Setup nav highlighting
    var path = location.pathname;
    $('.nav-sidebar li').each(function () {
        var $this = $(this);
        // if the current path is like this link, make it active
        if ($this.find("a").attr('href') === path) {
            $this.addClass('active');
        }
    })
    $.fn.dataTable.moment(dateTimeFormat(true));
    // Setup tooltips
    $('[data-toggle="tooltip"]').tooltip()
});