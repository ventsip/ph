const config = document.getElementById('config');

const container = document.createElement('div');
container.setAttribute('class', 'container');
config.appendChild(container);

var requestCfg = new XMLHttpRequest();
requestCfg.open('GET', '/config', true);
requestCfg.onload = function () {
    var d = JSON.parse(this.response);
    if (requestCfg.status >= 200 && requestCfg.status < 400) {
        d.forEach(r => {
            r.processes.forEach(n => {
                const e = document.createElement('div');
                e.textContent = n;
                config.appendChild(e);
            });
            
            r.limits.forEach((v, k) => {
                const e = document.createElement('div');
                e.textContent = k;
                config.appendChild(e);
            });
        });
    } else {
        const e = document.createElement('div');
        e.textContent = `Error retreiving configuration`;
        config.appendChild(e);
    }
}
requestCfg.send();