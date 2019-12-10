const config = document.getElementById('config');

const container = document.createElement('div');
container.setAttribute('class', 'container');
config.appendChild(container);

var requestCfg = new XMLHttpRequest();
requestCfg.open('GET', '/config', true);
requestCfg.onload = function () {
    var dtls = JSON.parse(this.response);
    tbl = document.createElement('table');
    if (requestCfg.status >= 200 && requestCfg.status < 400) {
        dtls.forEach(dtl => {
            tr = tbl.insertRow()
            td = tr.insertCell();
            tblInner = document.createElement('table')
            dtl.processes.forEach(n => {
                tblInner.insertRow().insertCell().innerHTML = n
            });
            td.appendChild(tblInner)

            td = tr.insertCell();
            tblInner = document.createElement('table')
            Object.keys(dtl.limits).forEach(day => {
                tr = tblInner.insertRow()
                tr.insertCell().innerHTML = day
                tr.insertCell().innerHTML = dtl.limits[day]
            });
            td.appendChild(tblInner)
        });
    } else {
        const e = document.createElement('div');
        e.textContent = `Error retreiving configuration`;
        config.appendChild(e);
    }

    config.appendChild(tbl)

}
requestCfg.send();