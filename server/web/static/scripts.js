"use strict";
const refreshPeriod = 60000;

var dataConfig = {}; // loaded data

function editConfig() {
    $('#phid_edit_config').css({
        display: 'block'
    }).find('textarea').val(JSON.stringify(dataConfig, null, 4))
}

function saveConfig() {
    $.ajax({
        url: '/config',
        type: 'PUT',
        contentType: 'application/json',
        data: $('#phid_edit_config').css({
            display: 'block'
        }).find('textarea').val(),
        success: (r, s) => {
            $('#phid_edit_config').css({
                display: 'none'
            });
            requestCfg();
            requestProcessGroupBalance();
            requestProcessBalance();
        },
        error: (x, s, r) => {
            alert(r + ":\n" + x.responseText);
        }
    })
}

function requestData(ep, rootID, processData) {
    $.getJSON(ep, (d, s) => {
        if (s == "success") {
            processData(d, $('#' + $.escapeSelector(rootID)).html(""));
        } else {
            $('#' + $.escapeSelector(rootID)).text("Error retrieving data");
        }
    });
}

function genLimits(limits) {
    let t = $('<table class="w3-table w3-bordered"></table>');
    t.append($('<th>Day limits</th>'))

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

function genDowntime(dnts) {
    let t = $('<table class="w3-table w3-bordered"></table>');
    t.append($('<th>Downtime</th>'));

    if (dnts) {
        Object.keys(dnts).forEach(key => {
            t.append(
                $('<tr></tr>').append(
                    $('<td class="w3-right-align"></td>').text(key),
                    $('<td></td>').text(dnts[key])
                )
            );
        });
    }

    return t;
}

function registerHover(e, name) {
    const h = "w3-blue-grey";

    let cn = "ph-" + btoa(name);
    let s = '.' + $.escapeSelector(cn);
    e.addClass(cn).hover(
        () => {
            $(s).addClass(h);
        },
        () => {
            $(s).removeClass(h);
        }
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
                $('<div class="w3-margin" style="float:left"></div>').append(genLimits(dtl.limits)),
                $('<div class="w3-margin" style="float:left"></div>').append(genDowntime(dtl.downtime))
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

function genLimitAndBalance(l, d, b) {
    let c = $('<div class="w3-grey"></div>')
    let lnmb = toSeconds(l);

    if (d && lnmb > 0) {
        let progress = Math.min(100, 100 * toSeconds(b) / lnmb);
        let clr = "w3-light-green";

        if (progress > 90) {
            clr = "w3-red";
        } else if (progress > 75) {
            clr = "w3-orange";
        } else if (progress > 50) {
            clr = "w3-yellow";
        }

        c = c.append(
            $('<div></div>')
                .addClass(clr, "w3-center")
                .width(progress + "%")
                .text(b + "/" + l)
        );
    } else {
        c.text('No limit')
    }
    return c;
}

function genDowntimeLine(dnts, ts) {
    let c = $('<div class="w3-light-green"></div>');
    c.css({
        position: 'relative'
    });
    c.text('\xa0'); // non-breaking space 

    if (dnts) {
        // regex for hh:mm..hh:mm format
        const regex = /^(([0-9]|0[0-9]|1[0-9]|2[0-3])\:([0-5][0-9]))?\.\.(([0-9]|0[0-9]|1[0-9]|2[0-3])\:([0-5][0-9]))?$/;
        dnts.forEach(dnt => {
            if (regex.test(dnt)) {
                const m = dnt.match(regex) // see regex grouping for group indices
                const h1 = parseInt(m[2] || '00')
                const m1 = parseInt(m[3] || '00')
                const h2 = parseInt(m[5] || '24')
                const m2 = parseInt(m[6] || '00')
                const start = 100.0 * (h1 + m1 / 60.0) / 24.0
                const end = 100.0 * (h2 + m2 / 60.0) / 24.0
                c.append(
                    $('<div class="w3-red w3-tooltip"></div>')
                        .css({
                            left: start + "%",
                            top: 0,
                            position: 'absolute'
                        })
                        .width(end - start + "%")
                        .text('\xa0')
                        .append(
                            $('<span class="w3-center w3-text w3-tag" style="position:absolute;left:0%;bottom:100%"></span>')
                                .text(h1.toString().padStart(2, '0') + ':' +
                                    m1.toString().padStart(2, '0') + '..' +
                                    h2.toString().padStart(2, '0') + ':' +
                                    m2.toString().padStart(2, '0'))
                        )
                );
            }
        });
    }

    if (ts) {
        // regex for hh:mm format
        const regex = /^([0-9]|0[0-9]|1[0-9]|2[0-3])\:([0-5][0-9])$/;
        if (regex.test(ts)) {
            const m = ts.match(regex) // see regex grouping for group indices
            const hr = parseInt(m[1] || '00')
            const mn = parseInt(m[2] || '00')

            const pos = 100.0 * (hr + mn / 60.0) / 24.0

            c.append(
                $('<div class="w3-black"></div>')
                    .css({
                        left: pos + "%",
                        top: "0",
                        position: 'absolute'
                    })
                    .width(2)
                    .text('\xa0')
            );
        }
    }

    return c;
}


function processPGB(data, root) {
    data.forEach(pgb => {
        root.append(
            $('<div class="w3-card w3-margin" style="float:left"></div>').append(
                $('<header class="w3-container w3-light-blue w3-bar"></header>').append(processList(pgb.processes)),
                $('<div class="w3-container w3-margin"></div>').append(genLimitAndBalance(pgb.limit, pgb.limit_defined, pgb.balance)),
                $('<div class="w3-container w3-margin"></div>').append(genDowntimeLine(pgb.downtime, pgb.timestamp))
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

function testNotification() {
    if (Notification.permission !== 'granted')
        Notification.requestPermission();
    else {
        let notification = new Notification('Notification title', {
            icon: 'http://localhost:8080/favicon.ico',
            body: 'Hey there! You\'ve been notified! Go to ph',
        });
        notification.onclick = function () {
            window.open('http://localhost:8080');
        };
    }
}

function requestNotificationPermission() {
    if (typeof Notification === 'undefined') {
        alert('Desktop notifications not available in your browser. Try Chromium.');
        return;
    }

    if (Notification.permission !== 'granted')
        Notification.requestPermission();
}

$(document).ready(
    () => {
        requestNotificationPermission();

        $("#phid_version").load("/version");

        requestCfg();
        requestProcessGroupBalance();
        requestProcessBalance();

        setInterval("requestCfg();", refreshPeriod);
        setInterval("requestProcessGroupBalance();", refreshPeriod);
        setInterval("requestProcessBalance();", refreshPeriod);
    }
);