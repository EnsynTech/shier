var channels = []
var channelId = -1

function dismiss() {
    $("#modal\\.flashes").empty()
    $("#name").val("")
    $("#channel").val("sms")
    $("#endpoint_url").val("")
    $("#http_method").val("POST")
    $("#content_type").val("application/json")
    $("#auth_header_name").val("")
    $("#auth_header_value").val("")
    $("#success_status_codes").val("200,201,204")
    $("#body_template").val('{"to":"{{.Phone}}","text":"{{.Message}}"}')
    $("#ignore_cert_errors").prop("checked", false)
    $("#test_phone").val("")
    $("#test_email").val("")
    $("#test_message").val("")
    $("#modalSubmit").show()
    channelId = -1
}

function channelFormPayload() {
    return {
        name: $("#name").val(),
        channel: $("#channel").val(),
        endpoint_url: $("#endpoint_url").val(),
        http_method: $("#http_method").val(),
        content_type: $("#content_type").val(),
        auth_header_name: $("#auth_header_name").val(),
        auth_header_value: $("#auth_header_value").val(),
        success_status_codes: $("#success_status_codes").val(),
        body_template: $("#body_template").val(),
        ignore_cert_errors: $("#ignore_cert_errors").prop("checked")
    }
}

function edit(id) {
    channelId = id
    $("#modal\\.flashes").empty()
    if (id === -1) {
        $("#channelModalLabel").text(t("channels.modal_title"))
        dismiss()
        return
    }
    var ch = channels.find(function (x) {
        return x.id === id
    })
    if (!ch) {
        return
    }
    $("#channelModalLabel").text(t("channels.modal_title"))
    $("#name").val(ch.name)
    $("#channel").val(ch.channel)
    $("#endpoint_url").val(ch.endpoint_url)
    $("#http_method").val(ch.http_method || "POST")
    $("#content_type").val(ch.content_type || "application/json")
    $("#auth_header_name").val(ch.auth_header_name || "")
    $("#auth_header_value").val("")
    $("#success_status_codes").val(ch.success_status_codes || "200,201,204")
    $("#body_template").val(ch.body_template || "")
    $("#ignore_cert_errors").prop("checked", !!ch.ignore_cert_errors)
}

function save() {
    var payload = channelFormPayload()
    var method = channelId === -1 ? api.sendingChannels.post : api.sendingChannelId.put
    if (channelId !== -1) {
        payload.id = channelId
    }
    method(payload)
        .success(function () {
            dismiss()
            $("#modal").modal("hide")
            load()
        })
        .error(function (data) {
            modalError(data.responseJSON.message)
        })
}

function sendTest() {
    var payload = channelFormPayload()
    payload.phone = $("#test_phone").val()
    payload.email = $("#test_email").val()
    payload.message = $("#test_message").val()
    var req = channelId === -1 ? api.sendingChannels.test(payload) : api.sendingChannelId.test(channelId, payload)
    req.success(function (data) {
        $("#modal\\.flashes").empty().append("<div style=\"text-align:center\" class=\"alert alert-success\">\
        <i class=\"fa fa-check-circle\"></i> " + escapeHtml(data.message) + "</div>")
    }).error(function (data) {
        modalError(data.responseJSON.message)
    })
}

function deleteChannel(id) {
    var ch = channels.find(function (x) {
        return x.id === id
    })
    if (!ch) {
        return
    }
    Swal.fire({
        title: t("js.confirm_delete"),
        text: t("channels.confirm_delete"),
        type: "warning",
        animation: false,
        showCancelButton: true,
        confirmButtonText: t("common.delete") + " " + escapeHtml(ch.name),
        confirmButtonColor: "#428bca",
        reverseButtons: true,
        allowOutsideClick: false,
        preConfirm: function () {
            return new Promise(function (resolve, reject) {
                api.sendingChannelId.delete(id)
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
            load()
        }
    })
}

function load() {
    $("#channelTable").hide()
    $("#emptyMessage").hide()
    $("#loading").show()
    api.sendingChannels.get()
        .success(function (cs) {
            channels = cs
            $("#loading").hide()
            if (channels.length > 0) {
                $("#emptyMessage").hide()
                $("#channelTable").show()
                var table = $("#channelTable").DataTable({
                    destroy: true,
                    columnDefs: [{
                        orderable: false,
                        targets: "no-sort"
                    }]
                })
                table.clear()
                $.each(channels, function (i, ch) {
                    table.row.add([
                        escapeHtml(ch.name),
                        escapeHtml(ch.channel),
                        escapeHtml(ch.endpoint_url),
                        moment(ch.modified_date).format("MMMM Do YYYY, h:mm:ss a"),
                        "<div class='pull-right'><span data-toggle='modal' data-backdrop='static' data-target='#modal'><button class='btn btn-primary' onclick='edit(" + ch.id + ")' title='" + t("js.edit") + "' aria-label='" + t("js.edit") + "'>\
                    <i class='fa fa-pencil'></i>\
                    </button></span>\
                    <button class='btn btn-danger' onclick='deleteChannel(" + ch.id + ")' title='" + t("js.delete") + "' aria-label='" + t("js.delete") + "'>\
                    <i class='fa fa-trash-o'></i>\
                    </button></div>"
                    ]).draw()
                })
            } else {
                $("#emptyMessage").show()
            }
        })
        .error(function () {
            $("#loading").hide()
            errorFlash(t("js.error_loading"))
        })
}

$(document).ready(function () {
    load()
})
