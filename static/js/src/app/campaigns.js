// labels is a map of campaign statuses to
// CSS classes
var labels = {
    "In progress": "label-primary",
    "Queued": "label-info",
    "Completed": "label-success",
    "Emails Sent": "label-success",
    "Error": "label-danger"
}

var campaigns = []
var trashCampaigns = []
var campaign = {}
var campaignStepRows = []
var campaignTemplateOptions = []
var campaignPageOptions = []

function addCampaignStep(step) {
    step = step || { template_name: "", delay_minutes: 60, trigger: "after_sent", page_id: "" }
    campaignStepRows.push(step)
    renderCampaignSteps()
}

function removeCampaignStep(idx) {
    campaignStepRows.splice(idx, 1)
    renderCampaignSteps()
}

function renderCampaignSteps() {
    var html = ""
    $.each(campaignStepRows, function (i, step) {
        html += "<div class='well well-sm' style='margin-bottom:8px;' data-step-idx='" + i + "'>" +
            "<div class='row'>" +
            "<div class='col-sm-4'>" +
            "<label>" + t("campaigns.step_template") + " #" + (i + 1) + "</label>" +
            "<select class='form-control campaign-step-template' id='campaign_step_template_" + i + "'></select>" +
            "</div>" +
            "<div class='col-sm-3'>" +
            "<label>" + t("campaigns.step_page") + "</label>" +
            "<select class='form-control campaign-step-page' id='campaign_step_page_" + i + "'><option value=''></option></select>" +
            "</div>" +
            "<div class='col-sm-2'>" +
            "<label>" + t("campaigns.step_delay") + "</label>" +
            "<input type='number' min='0' class='form-control campaign-step-delay' value='" + (i === 0 ? 0 : (step.delay_minutes || 60)) + "'" + (i === 0 ? " disabled" : "") + ">" +
            "</div>" +
            "<div class='col-sm-2'>" +
            "<label>" + t("campaigns.step_trigger") + "</label>" +
            "<select class='form-control campaign-step-trigger'>" +
            "<option value='after_sent'" + (step.trigger !== "on_reply" ? " selected" : "") + ">" + t("campaigns.trigger_after_sent") + "</option>" +
            "<option value='on_reply'" + (step.trigger === "on_reply" ? " selected" : "") + ">" + t("campaigns.trigger_on_reply") + "</option>" +
            "</select>" +
            "</div>" +
            "<div class='col-sm-1' style='padding-top:24px;'>" +
            (i > 0 ? "<button type='button' class='btn btn-danger btn-sm' onclick='removeCampaignStep(" + i + ")' title='" + t("js.delete") + "' aria-label='" + t("js.delete") + "'><i class='fa fa-trash'></i></button>" : "") +
            "</div></div>" +
            "<p class='help-block' style='margin:6px 0 0;'>" + t("campaigns.trigger_on_reply_note") + " " + t("campaigns.step_page_help") + "</p>" +
            "</div>"
    })
    $("#campaignStepsList").html(html)
    $(".campaign-step-template").each(function (i) {
        var $el = $(this)
        $el.select2({
            placeholder: t("campaigns.step_template"),
            data: campaignTemplateOptions
        })
        if (campaignStepRows[i] && campaignStepRows[i].template_id) {
            $el.val(campaignStepRows[i].template_id.toString()).trigger("change.select2")
        } else if (i === 0 && $("#template").val()) {
            $el.val($("#template").val()).trigger("change.select2")
        }
    })
    $(".campaign-step-page").each(function (i) {
        var $el = $(this)
        var pageData = [{ id: "", text: "" }].concat(campaignPageOptions || [])
        $el.select2({
            placeholder: t("campaigns.step_page"),
            allowClear: true,
            data: pageData
        })
        if (campaignStepRows[i] && campaignStepRows[i].page_id) {
            $el.val(campaignStepRows[i].page_id.toString()).trigger("change.select2")
        }
    })
}

function collectCampaignSteps() {
    var steps = []
    $("#campaignStepsList .well").each(function (i) {
        var $row = $(this)
        var $tmpl = $row.find(".campaign-step-template")
        var tmplData = $tmpl.select2("data")
        var name = (tmplData && tmplData[0] && tmplData[0].text) || ""
        if (!name) {
            return
        }
        var step = {
            template: { name: name },
            delay_minutes: i === 0 ? 0 : (parseInt($row.find(".campaign-step-delay").val(), 10) || 0),
            trigger: $row.find(".campaign-step-trigger").val() || "after_sent"
        }
        var $page = $row.find(".campaign-step-page")
        var pageData = $page.select2("data")
        if (pageData && pageData[0] && pageData[0].text) {
            step.page = { name: pageData[0].text }
        }
        steps.push(step)
    })
    return steps
}

// Launch attempts to POST to /campaigns/
function launch() {
    Swal.fire({
        title: t("js.confirm_launch"),
        text: t("js.confirm_launch_text"),
        type: "question",
        animation: false,
        showCancelButton: true,
        confirmButtonText: t("js.launch"),
        confirmButtonColor: "#428bca",
        reverseButtons: true,
        allowOutsideClick: false,
        showLoaderOnConfirm: true,
        preConfirm: function () {
            return new Promise(function (resolve, reject) {
                groups = []
                $("#users").select2("data").forEach(function (group) {
                    groups.push({
                        name: group.text
                    });
                })
                // Validate our fields
                var send_by_date = $("#send_by_date").val()
                if (send_by_date != "") {
                    send_by_date = moment(send_by_date, "MMMM Do YYYY, h:mm a").utc().format()
                }
                var steps = collectCampaignSteps()
                var channel = $("#channel").val() || "email"
                campaign = {
                    name: $("#name").val(),
                    channel: channel,
                    template: {
                        name: $("#template").select2("data")[0].text
                    },
                    url: $("#url").val(),
                    page: {
                        name: $("#page").select2("data")[0].text
                    },
                    smtp: {
                        name: ""
                    },
                    sending_channel: {
                        name: ""
                    },
                    launch_date: moment($("#launch_date").val(), "MMMM Do YYYY, h:mm a").utc().format(),
                    send_by_date: send_by_date || null,
                    groups: groups,
                }
                if (channel === "email") {
                    campaign.smtp = {
                        name: $("#profile").select2("data")[0].text
                    }
                } else {
                    campaign.sending_channel = {
                        name: $("#sending_channel").select2("data")[0].text
                    }
                }
                if (steps.length > 1) {
                    campaign.steps = steps
                    campaign.template = steps[0].template
                } else if (steps.length === 1) {
                    // Single explicit step still works; omit steps for classic single-template campaigns.
                    campaign.template = steps[0].template
                }
                // Submit the campaign
                api.campaigns.post(campaign)
                    .success(function (data) {
                        resolve()
                        campaign = data
                    })
                    .error(function (data) {
                        $("#modal\\.flashes").empty().append("<div style=\"text-align:center\" class=\"alert alert-danger\">\
            <i class=\"fa fa-exclamation-circle\"></i> " + data.responseJSON.message + "</div>")
                        Swal.close()
                    })
            })
        }
    }).then(function (result) {
        if (result.value){
            Swal.fire(
                'Campaign Scheduled!',
                'This campaign has been scheduled for launch!',
                'success'
            );
        }
        $('button:contains("OK")').on('click', function () {
            window.location = "/campaigns/" + campaign.id.toString()
        })
    })
}

// Attempts to send a test email by POSTing to /campaigns/
function sendTestEmail() {
    var test_email_request = {
        template: {
            name: $("#template").select2("data")[0].text
        },
        first_name: $("input[name=to_first_name]").val(),
        last_name: $("input[name=to_last_name]").val(),
        email: $("input[name=to_email]").val(),
        position: $("input[name=to_position]").val(),
        department: $("input[name=to_department]").val(),
        url: $("#url").val(),
        page: {
            name: $("#page").select2("data")[0].text
        },
        smtp: {
            name: $("#profile").select2("data")[0].text
        }
    }
    btnHtml = $("#sendTestModalSubmit").html()
    $("#sendTestModalSubmit").html('<i class="fa fa-spinner fa-spin"></i> Sending')
    // Send the test email
    api.send_test_email(test_email_request)
        .success(function (data) {
            $("#sendTestEmailModal\\.flashes").empty().append("<div style=\"text-align:center\" class=\"alert alert-success\">\
            <i class=\"fa fa-check-circle\"></i> Email Sent!</div>")
            $("#sendTestModalSubmit").html(btnHtml)
        })
        .error(function (data) {
            $("#sendTestEmailModal\\.flashes").empty().append("<div style=\"text-align:center\" class=\"alert alert-danger\">\
            <i class=\"fa fa-exclamation-circle\"></i> " + data.responseJSON.message + "</div>")
            $("#sendTestModalSubmit").html(btnHtml)
        })
}

function dismiss() {
    $("#modal\\.flashes").empty();
    $("#name").val("");
    $("#template").val("").change();
    $("#page").val("").change();
    $("#url").val("");
    $("#profile").val("").change();
    $("#users").val("").change();
    campaignStepRows = [];
    renderCampaignSteps();
    $("#modal").modal('hide');
}

function deleteCampaign(idx) {
    Swal.fire({
        title: t("js.confirm_delete"),
        text: t("js.confirm_delete_campaign"),
        type: "warning",
        animation: false,
        showCancelButton: true,
        confirmButtonText: t("js.delete") + " " + campaigns[idx].name,
        confirmButtonColor: "#428bca",
        reverseButtons: true,
        allowOutsideClick: false,
        preConfirm: function () {
            return new Promise(function (resolve, reject) {
                api.campaignId.delete(campaigns[idx].id)
                    .success(function () {
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
                location.reload()
            })
        }
    })
}

function archiveCampaign(idx) {
    Swal.fire({
        title: t("js.confirm_delete"),
        text: t("js.confirm_archive_campaign"),
        type: "warning",
        animation: false,
        showCancelButton: true,
        confirmButtonText: t("js.archive") + " " + campaigns[idx].name,
        confirmButtonColor: "#428bca",
        reverseButtons: true,
        allowOutsideClick: false,
        preConfirm: function () {
            return new Promise(function (resolve, reject) {
                api.campaignId.complete(campaigns[idx].id)
                    .success(function () {
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
                t("js.campaign_archived"),
                t("js.campaign_archived_detail"),
                "success"
            ).then(function () {
                location.reload()
            })
        }
    })
}

function restoreCampaign(id) {
    Swal.fire({
        title: t("js.confirm_delete"),
        text: t("js.confirm_restore_campaign"),
        type: "warning",
        animation: false,
        showCancelButton: true,
        confirmButtonText: t("js.restore"),
        confirmButtonColor: "#428bca",
        reverseButtons: true,
        allowOutsideClick: false,
        preConfirm: function () {
            return new Promise(function (resolve, reject) {
                api.campaignId.restore(id)
                    .success(function () {
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
                t("js.campaign_restored"),
                t("js.campaign_restored_detail"),
                "success"
            ).then(function () {
                location.reload()
            })
        }
    })
}

function purgeCampaign(id) {
    var name = ""
    $.each(trashCampaigns, function (_, campaign) {
        if (campaign.id === id) {
            name = campaign.name
        }
    })
    Swal.fire({
        title: t("js.confirm_delete"),
        text: t("js.confirm_purge_campaign"),
        type: "warning",
        animation: false,
        showCancelButton: true,
        confirmButtonText: t("js.purge") + " " + name,
        confirmButtonColor: "#d9534f",
        reverseButtons: true,
        allowOutsideClick: false,
        preConfirm: function () {
            return new Promise(function (resolve, reject) {
                api.campaignId.purge(id)
                    .success(function () {
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
                t("js.campaign_purged"),
                t("js.campaign_purged_detail"),
                "success"
            ).then(function () {
                location.reload()
            })
        }
    })
}

function loadTrashCampaigns() {
    api.campaigns.trash()
        .success(function (data) {
            trashCampaigns = data.campaigns || []
            var trashTable = $("#campaignTableTrash").DataTable({
                columnDefs: [{
                    orderable: false,
                    targets: "no-sort"
                }],
                order: [
                    [1, "desc"]
                ]
            })
            if (trashCampaigns.length === 0) {
                $("#emptyTrash").show()
                return
            }
            var rows = []
            $.each(trashCampaigns, function (_, campaign) {
                var deletedAt = campaign.deleted_at ? moment(campaign.deleted_at).format("MMMM Do YYYY, h:mm:ss a") : ""
                rows.push([
                    escapeHtml(campaign.name),
                    deletedAt,
                    "<span class=\"label label-default\">" + escapeHtml(campaign.status) + "</span>",
                    "<div class='pull-right'><button class='btn btn-primary' onclick='restoreCampaign(" + campaign.id + ")' data-toggle='tooltip' data-placement='left' title='" + t("js.restore") + "' aria-label='" + t("js.restore") + "'>\
                    <i class='fa fa-undo'></i>\
                    </button>\
                    <button class='btn btn-danger' onclick='purgeCampaign(" + campaign.id + ")' data-toggle='tooltip' data-placement='left' title='" + t("js.purge") + "' aria-label='" + t("js.purge") + "'>\
                    <i class='fa fa-trash-o'></i>\
                    </button></div>"
                ])
            })
            $("#campaignTableTrash").show()
            trashTable.rows.add(rows).draw()
            $('[data-toggle="tooltip"]').tooltip()
        })
        .error(function () {
            errorFlash("Error fetching trash")
        })
}

function setupOptions() {
    api.groups.summary()
        .success(function (summaries) {
            groups = summaries.groups
            if (groups.length == 0) {
                modalError("No groups found!")
                return false;
            } else {
                var group_s2 = $.map(groups, function (obj) {
                    obj.text = obj.name
                    obj.title = obj.num_targets + " targets"
                    return obj
                });
                console.log(group_s2)
                $("#users.form-control").select2({
                    placeholder: "Select Groups",
                    data: group_s2,
                });
            }
        });
    api.templates.get()
        .success(function (templates) {
            if (templates.length == 0) {
                modalError("No templates found!")
                return false
            } else {
                var template_s2 = $.map(templates, function (obj) {
                    obj.text = obj.name
                    return obj
                });
                campaignTemplateOptions = template_s2
                var template_select = $("#template.form-control")
                template_select.select2({
                    placeholder: "Select a Template",
                    data: template_s2,
                });
                if (templates.length === 1) {
                    template_select.val(template_s2[0].id)
                    template_select.trigger('change.select2')
                }
                renderCampaignSteps()
            }
        });
    api.pages.get()
        .success(function (pages) {
            if (pages.length == 0) {
                modalError("No pages found!")
                return false
            } else {
                var page_s2 = $.map(pages, function (obj) {
                    obj.text = obj.name
                    return obj
                });
                campaignPageOptions = page_s2
                var page_select = $("#page.form-control")
                page_select.select2({
                    placeholder: "Select a Landing Page",
                    data: page_s2,
                });
                if (pages.length === 1) {
                    page_select.val(page_s2[0].id)
                    page_select.trigger('change.select2')
                }
                renderCampaignSteps()
            }
        });
    api.SMTP.get()
        .success(function (profiles) {
            if (profiles.length == 0) {
                // Profiles are only required for email channel.
            } else {
                var profile_s2 = $.map(profiles, function (obj) {
                    obj.text = obj.name
                    return obj
                });
                var profile_select = $("#profile.form-control")
                profile_select.select2({
                    placeholder: "Select a Sending Profile",
                    data: profile_s2,
                }).select2("val", profile_s2[0]);
                if (profiles.length === 1) {
                    profile_select.val(profile_s2[0].id)
                    profile_select.trigger('change.select2')
                }
            }
        });
    api.sendingChannels.get()
        .success(function (chs) {
            var channel_s2 = $.map(chs || [], function (obj) {
                obj.text = obj.name
                return obj
            });
            window.campaignSendingChannels = channel_s2
            $("#sending_channel").select2({
                placeholder: t("campaigns.sending_channel"),
                data: channel_s2
            })
            updateCampaignChannelUI()
        })
}

function updateCampaignChannelUI() {
    var channel = $("#channel").val() || "email"
    if (channel === "email") {
        $("#smtpProfileGroup").show()
        $("#sendingChannelGroup").hide()
    } else {
        $("#smtpProfileGroup").hide()
        $("#sendingChannelGroup").show()
        var filtered = (window.campaignSendingChannels || []).filter(function (ch) {
            return ch.channel === channel
        })
        $("#sending_channel").empty().select2({
            placeholder: t("campaigns.sending_channel"),
            data: filtered
        })
        if (filtered.length === 1) {
            $("#sending_channel").val(filtered[0].id.toString()).trigger("change.select2")
        }
    }
}

function edit(campaign) {
    setupOptions();
    campaignStepRows = [];
    renderCampaignSteps();
    $("#channel").off("change.channel").on("change.channel", updateCampaignChannelUI)
    updateCampaignChannelUI()
}

function copy(idx) {
    setupOptions();
    // Set our initial values
    api.campaignId.get(campaigns[idx].id)
        .success(function (campaign) {
            $("#name").val("Copy of " + campaign.name)
            if (!campaign.template.id) {
                $("#template").val("").change();
                $("#template").select2({
                    placeholder: campaign.template.name
                });
            } else {
                $("#template").val(campaign.template.id.toString());
                $("#template").trigger("change.select2")
            }
            if (!campaign.page.id) {
                $("#page").val("").change();
                $("#page").select2({
                    placeholder: campaign.page.name
                });
            } else {
                $("#page").val(campaign.page.id.toString());
                $("#page").trigger("change.select2")
            }
            if (!campaign.smtp.id) {
                $("#profile").val("").change();
                $("#profile").select2({
                    placeholder: campaign.smtp.name
                });
            } else {
                $("#profile").val(campaign.smtp.id.toString());
                $("#profile").trigger("change.select2")
            }
            $("#channel").val(campaign.channel || "email")
            updateCampaignChannelUI()
            if (campaign.sending_channel && campaign.sending_channel.id) {
                $("#sending_channel").val(campaign.sending_channel.id.toString()).trigger("change.select2")
            }
            $("#url").val(campaign.url)
        })
        .error(function (data) {
            $("#modal\\.flashes").empty().append("<div style=\"text-align:center\" class=\"alert alert-danger\">\
            <i class=\"fa fa-exclamation-circle\"></i> " + data.responseJSON.message + "</div>")
        })
}

$(document).ready(function () {
    $("#launch_date").datetimepicker({
        "widgetPositioning": {
            "vertical": "bottom"
        },
        "showTodayButton": true,
        "defaultDate": moment(),
        "format": "MMMM Do YYYY, h:mm a"
    })
    $("#send_by_date").datetimepicker({
        "widgetPositioning": {
            "vertical": "bottom"
        },
        "showTodayButton": true,
        "useCurrent": false,
        "format": "MMMM Do YYYY, h:mm a"
    })
    // Setup multiple modals
    // Code based on http://miles-by-motorcycle.com/static/bootstrap-modal/index.html
    $('.modal').on('hidden.bs.modal', function (event) {
        $(this).removeClass('fv-modal-stack');
        $('body').data('fv_open_modals', $('body').data('fv_open_modals') - 1);
    });
    $('.modal').on('shown.bs.modal', function (event) {
        // Keep track of the number of open modals
        if (typeof ($('body').data('fv_open_modals')) == 'undefined') {
            $('body').data('fv_open_modals', 0);
        }
        // if the z-index of this modal has been set, ignore.
        if ($(this).hasClass('fv-modal-stack')) {
            return;
        }
        $(this).addClass('fv-modal-stack');
        // Increment the number of open modals
        $('body').data('fv_open_modals', $('body').data('fv_open_modals') + 1);
        // Setup the appropriate z-index
        $(this).css('z-index', 1040 + (10 * $('body').data('fv_open_modals')));
        $('.modal-backdrop').not('.fv-modal-stack').css('z-index', 1039 + (10 * $('body').data('fv_open_modals')));
        $('.modal-backdrop').not('fv-modal-stack').addClass('fv-modal-stack');
    });
    // Scrollbar fix - https://stackoverflow.com/questions/19305821/multiple-modals-overlay
    $(document).on('hidden.bs.modal', '.modal', function () {
        $('.modal:visible').length && $(document.body).addClass('modal-open');
    });
    $('#modal').on('hidden.bs.modal', function (event) {
        dismiss()
    });
    api.campaigns.summary()
        .success(function (data) {
            campaigns = data.campaigns || []
            $("#loading").hide()
            var tableOpts = {
                columnDefs: [{
                    orderable: false,
                    targets: "no-sort"
                }],
                order: [
                    [1, "desc"]
                ]
            }
            activeCampaignsTable = $("#campaignTable").DataTable(tableOpts)
            archivedCampaignsTable = $("#campaignTableArchive").DataTable(tableOpts)
            var rows = {
                active: [],
                archived: []
            }
            $.each(campaigns, function (i, campaign) {
                label = labels[campaign.status] || "label-default"

                var launchDate
                if (moment(campaign.launch_date).isAfter(moment())) {
                    launchDate = "Scheduled to start: " + moment(campaign.launch_date).format("MMMM Do YYYY, h:mm:ss a")
                    var quickStats = launchDate + "<br><br>" + "Number of recipients: " + campaign.stats.total
                } else {
                    launchDate = "Launch Date: " + moment(campaign.launch_date).format("MMMM Do YYYY, h:mm:ss a")
                    var quickStats = launchDate + "<br><br>" + "Number of recipients: " + campaign.stats.total + "<br><br>" + "Emails opened: " + campaign.stats.opened + "<br><br>" + "Emails clicked: " + campaign.stats.clicked + "<br><br>" + "Submitted Credentials: " + campaign.stats.submitted_data + "<br><br>" + "Errors : " + campaign.stats.error + "<br><br>" + "Reported : " + campaign.stats.email_reported + "<br><br>" + "Replied : " + (campaign.stats.email_replied || 0)
                }

                var actions = "<div class='pull-right'><a class='btn btn-primary' href='/campaigns/" + campaign.id + "' data-toggle='tooltip' data-placement='left' title='" + t("js.view") + "' aria-label='" + t("js.view") + "'>\
                    <i class='fa fa-bar-chart'></i>\
                    </a>\
            <span data-toggle='modal' data-backdrop='static' data-target='#modal'><button class='btn btn-primary' data-toggle='tooltip' data-placement='left' title='" + t("js.copy") + "' aria-label='" + t("js.copy") + "' onclick='copy(" + i + ")'>\
                    <i class='fa fa-copy'></i>\
                    </button></span>"
                if (campaign.status == "Completed") {
                    actions += "<button class='btn btn-danger' onclick='deleteCampaign(" + i + ")' data-toggle='tooltip' data-placement='left' title='" + t("js.delete") + "' aria-label='" + t("js.delete") + "'>\
                    <i class='fa fa-trash-o'></i>\
                    </button></div>"
                } else {
                    actions += "<button class='btn btn-primary' onclick='archiveCampaign(" + i + ")' data-toggle='tooltip' data-placement='left' title='" + t("js.archive") + "' aria-label='" + t("js.archive") + "'>\
                    <i class='fa fa-archive'></i>\
                    </button></div>"
                }
                var row = [
                    escapeHtml(campaign.name),
                    moment(campaign.created_date).format("MMMM Do YYYY, h:mm:ss a"),
                    "<span class=\"label " + label + "\" data-toggle=\"tooltip\" data-placement=\"right\" data-html=\"true\" title=\"" + quickStats + "\">" + campaign.status + "</span>",
                    actions
                ]
                if (campaign.status == "Completed") {
                    rows.archived.push(row)
                } else {
                    rows.active.push(row)
                }
            })
            if (rows.active.length > 0) {
                $("#campaignTable").show()
                activeCampaignsTable.rows.add(rows.active).draw()
            } else {
                $("#emptyActive").show()
            }
            if (rows.archived.length > 0) {
                $("#campaignTableArchive").show()
                archivedCampaignsTable.rows.add(rows.archived).draw()
            } else {
                $("#emptyArchived").show()
            }
            $('[data-toggle="tooltip"]').tooltip()
            loadTrashCampaigns()
        })
        .error(function () {
            $("#loading").hide()
            errorFlash("Error fetching campaigns")
        })
    // Select2 Defaults
    $.fn.select2.defaults.set("width", "100%");
    $.fn.select2.defaults.set("dropdownParent", $("#modal_body"));
    $.fn.select2.defaults.set("theme", "bootstrap");
    $.fn.select2.defaults.set("sorter", function (data) {
        return data.sort(function (a, b) {
            if (a.text.toLowerCase() > b.text.toLowerCase()) {
                return 1;
            }
            if (a.text.toLowerCase() < b.text.toLowerCase()) {
                return -1;
            }
            return 0;
        });
    })

    // Open create modal from /campaigns?new=1 or #new
    var params = new URLSearchParams(window.location.search || "")
    if (params.get("new") === "1" || window.location.hash === "#new") {
        edit("new")
        $("#modal").modal({
            backdrop: "static",
            show: true
        })
        if (window.history && window.history.replaceState) {
            window.history.replaceState({}, document.title, "/campaigns")
        }
    }
})
