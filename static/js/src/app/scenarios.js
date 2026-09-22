$(function () {
    api.scenarioPacks.get()
        .success(function (packs) {
            $("#loading").hide()
            packs = packs || []
            if (!packs.length) {
                $("#emptyMessage").show()
                return
            }
            var html = ""
            $.each(packs, function (i, pack) {
                var tags = ""
                var seen = {}
                var addTag = function (label) {
                    if (!label || seen[label]) {
                        return
                    }
                    seen[label] = true
                    tags += "<span class='shier-tag'>" + escapeHtml(label) + "</span>"
                }
                addTag(pack.category)
                if (pack.difficulty) {
                    // Difficulty label is distinct from technique tags; never dedupe against category.
                    tags += "<span class='shier-tag'>" + escapeHtml(t("scenarios.difficulty") + " " + pack.difficulty) + "</span>"
                }
                if (pack.techniques && pack.techniques.length) {
                    $.each(pack.techniques.slice(0, 3), function (j, tech) {
                        addTag(tech)
                    })
                }
                html += "<article class='shier-scenario-card'>" +
                    "<div>" + tags + "</div>" +
                    "<h3>" + escapeHtml(pack.title || pack.id) + "</h3>" +
                    "<p>" + escapeHtml(pack.description || "") + "</p>" +
                    "<div>" +
                    "<a class='btn btn-primary btn-sm' href='/campaigns?new=1&scenario=" + encodeURIComponent(pack.id || "") + "'>" +
                    escapeHtml(t("scenarios.launch")) + "</a>" +
                    "</div></article>"
            })
            $("#scenarioGrid").html(html).show()
        })
        .error(function () {
            $("#loading").hide()
            errorFlash(t("scenarios.error"))
            $("#emptyMessage").show()
        })
})
