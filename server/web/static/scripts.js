function requestData(ep, rootID, processData) {
    let r = new XMLHttpRequest();
    r.open('GET', ep, true);

    r.onload = function () {
        const root = document.getElementById(rootID);
        root.innerHTML = "" // wipe out the element
        let d = JSON.parse(this.response);
        if (r.status >= 200 && r.status < 400) {
            processData(d, root)
        } else {
            root.innerText = `Error retreiving data`;
        };
    }
    r.send();
}

function limitsList(limits) {
    let t = document.createElement('table')
    t.classList.add("w3-table", "w3-bordered")

    Object.keys(limits).forEach(key => {
        let r = document.createElement('tr')

        let d = document.createElement('td')
        d.classList.add("w3-right-align")
        d.innerText = key
        r.appendChild(d)

        d = document.createElement('td')
        d.innerText = limits[key]
        r.appendChild(d)

        t.appendChild(r)
    })

    return t
}

function processList(processes) {
    let c = document.createElement('div')

    processes.forEach(proc => {
        let p = document.createElement('p')
        p.classList.add("w3-round", "w3-bar-item", "w3-margin", "w3-tag")
        p.innerText = proc

        c.appendChild(p)
    })

    return c
}

function configCard(dtl) {
    let c = document.createElement('div')
    c.classList.add("w3-card", "w3-margin")
    c.style.float = "left"

    let e = document.createElement('header')
    e.classList.add("w3-container", "w3-blue", "w3-bar")
    e.appendChild(processList(dtl.processes))
    c.appendChild(e)

    e = document.createElement('div')
    e.style.float = "left"
    e.appendChild(limitsList(dtl.limits))
    c.appendChild(e)

    return c
}

function processConfig(data, root) {
    data.forEach(dtl => {
        root.appendChild(configCard(dtl));
    })
}

function requestCfg() {
    requestData('/config', 'ph_config', processConfig)
}

requestCfg()

function toSeconds(d) {
    // regex for xxHxxMxxS format
    const regex = /^(\d{1,2}h)?(\d{1,2}m)?(\d{1,2}(\.\d*)?s)?$/i
    if (regex.test(d)) {
        return parseInt(d.match(/\d{1,2}h/i) || '0') * 60 * 60 +
            parseInt(d.match(/\d{1,2}m/i) || '0') * 60 +
            parseFloat(d.match(/\d{1,2}(\.\d*)?s/i) || '0')
    } else {
        return 0
    }
}

function limitAndBalance(l, b) {
    let pb = document.createElement('div')
    pb.classList.add("w3-dark-grey", "w3-round-xlarge")

    let p = document.createElement('div')
    p.classList.add("w3-container", "w3-round-xlarge")
    let progress = 100 // in case limit is 0
    let lnmb = toSeconds(l)
    if (lnmb > 0) {
        progress = Math.min(100, 100 * toSeconds(b) / toSeconds(l))
    }

    let clr = "w3-light-green"
    if (progress > 50) {
        if (progress > 75) {
            if (progress > 90) {
                clr = "w3-red"
            } else {
                clr = "w3-orange"
            }
        } else {
            clr = "w3-yellow"
        }
    }
    p.classList.add(clr)
    p.style.width = progress + "%"
    p.innerText = b + "/" + l

    pb.appendChild(p)

    return pb
}

function pgbCard(pgb) {
    let c = document.createElement('div')
    c.classList.add("w3-card", "w3-margin")
    c.style.float = "left"

    let e = document.createElement('header')
    e.classList.add("w3-container", "w3-light-blue", "w3-bar")
    e.appendChild(processList(pgb.processes))
    c.appendChild(e)

    e = document.createElement('div')
    e.classList.add("w3-container", "w3-margin")
    e.appendChild(limitAndBalance(pgb.limit, pgb.balance))
    c.appendChild(e)

    return c
}

function processPGB(data, root) {
    data.forEach(pgb => {
        root.appendChild(pgbCard(pgb));
    })
}

function requestProcessGroupBalance() {
    requestData('/groupbalance', 'ph_groupbalance', processPGB)
}

requestProcessGroupBalance()

function processProcB(data, root) {
    let t = document.createElement('table')
    t.classList.add("w3-card", "w3-margin", "w3-table", "w3-bordered")
    t.style.float = "left"

    Object.keys(data).forEach(key => {
        let r = document.createElement('tr')
        let d = document.createElement('td')
        d.classList.add("w3-right-align", "w3-round", "w3-tag")
        d.innerText = key
        r.appendChild(d)

        d = document.createElement('td')
        d.innerText = data[key]
        r.appendChild(d)

        t.appendChild(r)
    })

    root.appendChild(t)
}

function requestProcessBalance() {
    requestData('/processbalance', 'ph_processbalance', processProcB)
}

requestProcessBalance()