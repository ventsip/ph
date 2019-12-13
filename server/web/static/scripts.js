var requestCfg = new XMLHttpRequest();
requestCfg.open('GET', '/config', true);

function limitsList(limits) {
    var c = document.createElement('div')
    c.classList.add("w3-container", "w3-cell-row")

    Object.keys(limits).forEach(key => {
        var p = document.createElement('p')
        p.classList.add("w3-round", "w3-container", "w3-cell")
        p.innerText = key
        c.appendChild(p)

        p = document.createElement('p')
        p.classList.add("w3-round", "w3-container", "w3-cell")
        p.innerText = limits[key]
        c.appendChild(p)
    })

    return c
}

function processList(processes) {
    var c = document.createElement('div')
    c.classList.add("w3-container", "w3-cell-row")

    processes.forEach(proc => {
        var p = document.createElement('p')
        p.classList.add("w3-round", "w3-container", "w3-cell")
        p.innerText = proc

        c.appendChild(p)
    })

    return c
}

function ConfigCard(dtl) {
    var c = document.createElement('div')
    c.classList.add("w3-card-4")

    var e = document.createElement('header')
    e.classList.add("w3-container", "w3-blue", "w3-cell")
    e.appendChild(processList(dtl.processes))
    c.appendChild(e)

    e = document.createElement('div')
    e.classList.add("w3-container")
    e.appendChild(limitsList(dtl.limits))
    c.appendChild(e)

    return c
}

requestCfg.onload = function () {
    const config = document.getElementById('ph_config');
    var dtls = JSON.parse(this.response);
    if (requestCfg.status >= 200 && requestCfg.status < 400) {
        config.classList.add("w3-cell-row")
        dtls.forEach(dtl => {
            config.appendChild(ConfigCard(dtl));
        });
    } else {
        config.innerText = `Error retreiving configuration`;
    };
}
requestCfg.send();