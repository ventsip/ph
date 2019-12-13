var requestCfg = new XMLHttpRequest();
requestCfg.open('GET', '/config', true);

function limitsList(limits) {
    var c = document.createElement('table')
    c.classList.add("w3-table", "w3-bordered")

    Object.keys(limits).forEach(key => {
        var tr = document.createElement('tr')
        var td = document.createElement('td')
        td.classList.add("w3-right-align")
        td.innerText = key
        tr.appendChild(td)

        td = document.createElement('td')
        td.innerText = limits[key]
        tr.appendChild(td)

        c.appendChild(tr)
    })

    return c
}

function processList(processes) {
    var c = document.createElement('div')

    processes.forEach(proc => {
        var p = document.createElement('p')
        p.classList.add("w3-round", "w3-bar-item", "w3-margin-right", "w3-tag")
        p.innerText = proc

        c.appendChild(p)
    })

    return c
}

function ConfigCard(dtl) {
    var c = document.createElement('div')
    c.classList.add("w3-card", "w3-margin")
    c.style.float = "left"

    var e = document.createElement('header')
    e.classList.add("w3-container", "w3-blue", "w3-bar")
    e.appendChild(processList(dtl.processes))
    c.appendChild(e)

    e = document.createElement('div')
    e.style.float = "left"
    e.appendChild(limitsList(dtl.limits))
    c.appendChild(e)

    return c
}

requestCfg.onload = function () {
    const config = document.getElementById('ph_config');
    config.innerHTML = "" // wipe out the element
    var dtls = JSON.parse(this.response);
    if (requestCfg.status >= 200 && requestCfg.status < 400) {
        dtls.forEach(dtl => {
            config.appendChild(ConfigCard(dtl));
        });
    } else {
        config.innerText = `Error retreiving configuration`;
    };
}
requestCfg.send();