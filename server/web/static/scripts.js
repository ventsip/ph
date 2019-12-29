"use strict";
const refreshPeriod = 60000;

var dataConfig = {}; // loaded data

function editConfig() {
    $('#phid_edit_config').css({ display: 'block' }).find('textarea').val(JSON.stringify(dataConfig, null, 4))
}

function saveConfig() {

    let d = $('#phid_edit_config').css({ display: 'block' }).find('textarea').val()

    $.ajax({
        url: '/config',
        type: 'PUT',
        contentType: 'application/json',
        data: d,
        success: (r, s) => { requestCfg() },
        error: (x, s, r) => { alert(r + ":\n" + x.responseText) }
    })
    $('#phid_edit_config').css({ display: 'none' })
}

function requestData(ep, rootID, processData) {
    $.getJSON(ep, (d, s) => {
        if (s == "success") {
            processData(d, $('#' + $.escapeSelector(rootID)).html(""));
        }
        else {
            $('#' + $.escapeSelector(rootID)).text("Error retreiving data");
        }
    });
}

function genLimits(limits) {
    let t = $('<table class="w3-table w3-bordered"></table>')
    Object.keys(limits).forEach(key => {
        t.append(
            $('<tr></tr>').append(
                $('<td class="w3-right-align"></td>').text(key),
                $('<td></td>').text(limits[key])
            )
        );
    });

    return t;
}

function registerHover(e, name) {
    const h = "w3-blue-grey";

    let cn = "ph-" + name;
    let s = '.' + $.escapeSelector(cn)
    e.addClass(cn).hover(
        () => { $(s).addClass(h) },
        () => { $(s).removeClass(h) }
    );

    return e;
}

function processList(processes) {
    let c = $('<div></div>')

    processes.forEach(proc => {
        c.append(registerHover($('<p class="w3-round w3-bar-item w3-margin w3-tag"></p>').text(proc), proc));
    });

    return c;
}

function processConfig(data, root) {
    dataConfig = data;
    data.forEach(dtl => {
        root.append(
            $('<div class="w3-card w3-margin" style="float:left"></div>').append(
                $('<header class="w3-container w3-blue w3-bar"></header>').append(processList(dtl.processes)),
                $('<div class="w3-margin" style="float:left"></div>').append(genLimits(dtl.limits))
            )
        );
    });
}

function requestCfg() {
    requestData('/config', 'phid_config', processConfig);
}

function toSeconds(d) {
    // regex for xxHxxMxxS format
    const regex = /^(\d+h)?(\d+m)?(\d+(\.\d*)?s)?$/i;
    if (regex.test(d)) {
        return parseInt(d.match(/\d+h/i) || '0') * 60 * 60 +
            parseInt(d.match(/\d+m/i) || '0') * 60 +
            parseFloat(d.match(/\d+(\.\d*)?s/i) || '0');
    } else {
        return 0;
    }
}

function genLimitAndBalance(l, b) {

    let progress = 100; // in case limit is 0
    let lnmb = toSeconds(l);
    if (lnmb > 0) {
        progress = Math.min(100, 100 * toSeconds(b) / toSeconds(l));
    }

    let clr = "w3-light-green";

    if (progress > 90) {
        clr = "w3-red";
    } else if (progress > 75) {
        clr = "w3-orange";
    } else if (progress > 50) {
        clr = "w3-yellow";
    }

    return $('<div class="w3-dark-grey w3-round-xlarge"></div>').append(
        $('<div class="w3-container w3-round-xlarge"></div>')
            .addClass(clr, "w3-center")
            .width(progress + "%")
            .text(b + "/" + l)
    );
}

function processPGB(data, root) {
    data.forEach(pgb => {
        root.append(
            $('<div class="w3-card w3-margin" style="float:left"></div>').append(
                $('<header class="w3-container w3-light-blue w3-bar"></header>').append(processList(pgb.processes)),
                $('<div class="w3-container w3-margin"></div>').append(genLimitAndBalance(pgb.limit, pgb.balance))
            )
        );
    });
}

function requestProcessGroupBalance() {
    requestData('/groupbalance', 'phid_groupbalance', processPGB);
}

function processProcB(data, root) {
    let t = $('<table class="w3-table w3-bordered"></table>')

    Object.keys(data).forEach(key => {
        t.append(
            $('<tr></tr>').append(
                registerHover($('<td class="w3-right-align w3-round w3-tag" style="float:right"></td>').text(key), key),
                $('<td></td>').text(data[key])
            )
        );
    });

    root.append(
        $('<div class="w3-card w3-margin" style="float:left"></div>').append(
            $('<div class="w3-margin"></div>').append(t)
        )
    );
}

function requestProcessBalance() {
    requestData('/processbalance', 'phid_processbalance', processProcB);
}

$(document).ready(
    () => {
        $("#phid_version").load("/version");

        requestCfg();
        requestProcessGroupBalance();
        requestProcessBalance();

        setInterval("requestCfg();", refreshPeriod);
        setInterval("requestProcessGroupBalance();", refreshPeriod);
        setInterval("requestProcessBalance();", refreshPeriod);
    }
);