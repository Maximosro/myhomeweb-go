// ===== Focus Google search on load =====
document.getElementById('search-input').focus();

// ===== Date display =====
const now = new Date();
document.getElementById('date-weekday').textContent = now.toLocaleDateString('es-ES', { weekday: 'long' }).toUpperCase();
document.getElementById('date-numeric').textContent = now.toLocaleDateString('es-ES', { day: '2-digit', month: 'short', year: 'numeric' }).toUpperCase();

// ===== App status checker =====
async function checkStatus(entry) {
    const url = entry.dataset.url;
    const dot = entry.querySelector('.status-indicator');
    const label = entry.querySelector('.status-label');
    try {
        const ctrl = new AbortController();
        const t = setTimeout(() => ctrl.abort(), 5000);
        await fetch(url, { mode: 'no-cors', signal: ctrl.signal });
        clearTimeout(t);
        dot.classList.add('online');
        label.textContent = 'Online';
        label.classList.add('online');
    } catch {
        dot.classList.add('offline');
        label.textContent = 'Offline';
        label.classList.add('offline');
    }
}

document.querySelectorAll('.app-entry[data-url]').forEach(checkStatus);

setInterval(() => {
    document.querySelectorAll('.app-entry[data-url]').forEach(el => {
        const dot = el.querySelector('.status-indicator');
        const label = el.querySelector('.status-label');
        dot.className = 'status-indicator';
        label.className = 'status-label';
        label.textContent = 'Checking...';
        checkStatus(el);
    });
}, 30000);

// ===== Weather widget =====
async function fetchWeather() {
    const lat = 37.34, lon = -5.84;
    const url = `https://api.open-meteo.com/v1/forecast?latitude=${lat}&longitude=${lon}&current=temperature_2m,relative_humidity_2m,weather_code,wind_speed_10m&timezone=Europe%2FMadrid`;
    try {
        const res = await fetch(url);
        const data = await res.json();
        const c = data.current;
        document.getElementById('weather-temp').textContent = Math.round(c.temperature_2m) + '°';
        document.getElementById('weather-humidity').textContent = '💧 ' + c.relative_humidity_2m + '%';
        document.getElementById('weather-wind').textContent = '💨 ' + Math.round(c.wind_speed_10m) + ' km/h';
        const { icon, desc } = getWeatherInfo(c.weather_code);
        document.getElementById('weather-icon').textContent = icon;
        document.getElementById('weather-desc').textContent = desc;
    } catch {
        document.getElementById('weather-desc').textContent = 'Sin conexión';
    }
}

function getWeatherInfo(code) {
    const map = {
        0: { icon: '☀️', desc: 'Despejado' },
        1: { icon: '🌤️', desc: 'Mayormente despejado' },
        2: { icon: '⛅', desc: 'Parcialmente nublado' },
        3: { icon: '☁️', desc: 'Nublado' },
        45: { icon: '🌫️', desc: 'Niebla' },
        48: { icon: '🌫️', desc: 'Niebla helada' },
        51: { icon: '🌦️', desc: 'Llovizna ligera' },
        53: { icon: '🌦️', desc: 'Llovizna' },
        55: { icon: '🌧️', desc: 'Llovizna intensa' },
        61: { icon: '🌧️', desc: 'Lluvia ligera' },
        63: { icon: '🌧️', desc: 'Lluvia moderada' },
        65: { icon: '🌧️', desc: 'Lluvia intensa' },
        71: { icon: '🌨️', desc: 'Nevada ligera' },
        73: { icon: '🌨️', desc: 'Nevada moderada' },
        75: { icon: '❄️', desc: 'Nevada intensa' },
        80: { icon: '🌦️', desc: 'Chubascos ligeros' },
        81: { icon: '🌧️', desc: 'Chubascos moderados' },
        82: { icon: '⛈️', desc: 'Chubascos intensos' },
        95: { icon: '⛈️', desc: 'Tormenta' },
        96: { icon: '⛈️', desc: 'Tormenta con granizo' },
        99: { icon: '⛈️', desc: 'Tormenta con granizo fuerte' },
    };
    return map[code] || { icon: '🌡️', desc: 'Desconocido' };
}

fetchWeather();
setInterval(fetchWeather, 600000);

// ===== Bandwidth measurement =====
async function measureBandwidth() {
    const valEl = document.getElementById('bw-value');
    const barEl = document.getElementById('bw-bar');
    const statusEl = document.getElementById('bw-status');
    const btnEl = document.getElementById('bw-refresh');
    btnEl.classList.add('spinning');
    valEl.textContent = '--';
    barEl.style.width = '0%';
    statusEl.textContent = 'Midiendo...';
    try {
        const bytes = 25000000;
        const url = `https://speed.cloudflare.com/__down?bytes=${bytes}&ckSize=25000000&measId=${Date.now()}`;
        const startTime = performance.now();
        const response = await fetch(url, { cache: 'no-store' });
        const reader = response.body.getReader();
        let totalBytes = 0;
        while (true) {
            const { done, value } = await reader.read();
            if (done) break;
            totalBytes += value.length;
        }
        const endTime = performance.now();
        const durationSec = (endTime - startTime) / 1000;
        const speedMbps = ((totalBytes * 8) / durationSec / 1000000).toFixed(1);
        valEl.textContent = speedMbps;
        const pct = Math.min((speedMbps / 500) * 100, 100);
        barEl.style.width = pct + '%';
        if (speedMbps >= 200) {
            barEl.style.background = 'linear-gradient(90deg, #00e676, #00c853)';
            statusEl.textContent = 'Excelente';
            statusEl.style.color = '#00e676';
        } else if (speedMbps >= 100) {
            barEl.style.background = 'linear-gradient(90deg, #ffeb3b, #ffc107)';
            statusEl.textContent = 'Buena';
            statusEl.style.color = '#ffc107';
        } else if (speedMbps >= 30) {
            barEl.style.background = 'linear-gradient(90deg, #ff9800, #f57c00)';
            statusEl.textContent = 'Regular';
            statusEl.style.color = '#ff9800';
        } else {
            barEl.style.background = 'linear-gradient(90deg, #f44336, #d32f2f)';
            statusEl.textContent = 'Lenta';
            statusEl.style.color = '#f44336';
        }
    } catch {
        valEl.textContent = '?';
        statusEl.textContent = 'Error de conexión';
        statusEl.style.color = '#f44336';
    }
    btnEl.classList.remove('spinning');
}

document.getElementById('bw-refresh').addEventListener('click', measureBandwidth);
measureBandwidth();

// ===== API helpers =====
const API_BASE = '/api/v1';

async function apiFetch(method, path, body) {
    const options = { method, headers: {} };
    if (body) {
        options.headers['Content-Type'] = 'application/json';
        options.body = JSON.stringify(body);
    }
    const res = await fetch(`${API_BASE}${path}`, options);
    if (!res.ok) {
        const err = new Error(`API error: ${res.status}`);
        err.status = res.status;
        throw err;
    }
    return res.status === 204 ? null : res.json();
}

// ===== Modal logic =====
const overlay = document.getElementById('modal-overlay');
const modalTitle = document.getElementById('modal-title');
const modalFields = document.getElementById('modal-fields');
const modalSave = document.getElementById('modal-save');
const modalCancel = document.getElementById('modal-cancel');
let currentModalAction = null;

function openModal() { overlay.classList.add('active'); }
function closeModal() { overlay.classList.remove('active'); currentModalAction = null; }

modalCancel.addEventListener('click', closeModal);
overlay.addEventListener('click', (e) => { if (e.target === overlay) closeModal(); });

document.getElementById('modal').addEventListener('keydown', (e) => {
    if (e.key === 'Enter' && currentModalAction) currentModalAction();
    if (e.key === 'Escape') closeModal();
});

modalSave.addEventListener('click', () => { if (currentModalAction) currentModalAction(); });

// ===== CRUD: Add link =====
function openAddLinkModal(categoryEl) {
    const catId = categoryEl.dataset.categoryId;
    const catName = categoryEl.querySelector('.category-title').textContent.trim();
    modalTitle.textContent = 'Añadir enlace a ' + catName;
    modalFields.innerHTML = `
        <input type="text" id="input-link-name" placeholder="Nombre (ej: GitHub)" autocomplete="off">
        <input type="url" id="input-link-url" placeholder="URL (ej: https://github.com)" autocomplete="off" style="margin-top:0.5rem">
    `;
    currentModalAction = async () => {
        const name = document.getElementById('input-link-name').value.trim();
        const url = document.getElementById('input-link-url').value.trim();
        if (!name || !url) return;
        try { new URL(url); } catch { return; }
        try {
            await apiFetch('POST', '/links', { name, url, categoryId: catId });
            location.reload();
        } catch (e) {
            alert('Error al crear enlace');
        }
    };
    openModal();
    setTimeout(() => document.getElementById('input-link-name').focus(), 50);
}

// ===== CRUD: Delete link =====
async function deleteLink(id) {
    if (!confirm('¿Eliminar este enlace?')) return;
    try {
        await apiFetch('DELETE', '/links/' + id);
        location.reload();
    } catch (e) {
        if (e.status === 403) alert('No se puede eliminar un enlace predefinido');
        else alert('Error al eliminar');
    }
}

// ===== CRUD: Add category =====
function openAddCategoryModal() {
    modalTitle.textContent = 'Añadir categoría';
    modalFields.innerHTML = `
        <input type="text" id="input-cat-name" placeholder="Nombre de categoría" autocomplete="off">
        <input type="text" id="input-cat-icon" placeholder="Emoji icono (ej: 🔧)" autocomplete="off" style="margin-top:0.5rem" maxlength="2">
    `;
    currentModalAction = async () => {
        const name = document.getElementById('input-cat-name').value.trim();
        const icon = document.getElementById('input-cat-icon').value.trim() || '📁';
        if (!name) return;
        try {
            await apiFetch('POST', '/categories', { name, icon });
            location.reload();
        } catch (e) {
            alert('Error al crear categoría');
        }
    };
    openModal();
    setTimeout(() => document.getElementById('input-cat-name').focus(), 50);
}

// ===== CRUD: Delete category =====
async function deleteCategory(id) {
    if (!confirm('¿Eliminar esta categoría y todos sus enlaces?')) return;
    try {
        await apiFetch('DELETE', '/categories/' + id);
        location.reload();
    } catch (e) {
        if (e.status === 403) alert('No se puede eliminar una categoría predefinida');
        else alert('Error al eliminar');
    }
}

document.getElementById('btn-add-category').addEventListener('click', openAddCategoryModal);

// ===== Export JSON =====
document.getElementById('btn-export-json').addEventListener('click', async () => {
    try {
        const data = await apiFetch('GET', '/export');
        const json = JSON.stringify(data, null, 2);
        const blob = new Blob([json], { type: 'application/json' });
        const a = document.createElement('a');
        a.href = URL.createObjectURL(blob);
        a.download = 'myapps-data.json';
        a.click();
        URL.revokeObjectURL(a.href);
    } catch (e) {
        alert('Error al exportar');
    }
});

// ===== Import JSON =====
document.getElementById('btn-import-json').addEventListener('click', () => {
    document.getElementById('json-file-input').click();
});

document.getElementById('json-file-input').addEventListener('change', async (e) => {
    const file = e.target.files[0];
    if (!file) return;
    try {
        const text = await file.text();
        const data = JSON.parse(text);
        const result = await apiFetch('POST', '/import', data);
        alert(`Importación completada: ${result.categoriesImported} categorías, ${result.linksImported} enlaces`);
        location.reload();
    } catch (err) {
        alert('Error al importar: ' + err.message);
    }
    e.target.value = '';
});
