(async function() {
    // Clear login redirect guard — we made it to the dashboard
    sessionStorage.removeItem('login-redirect-attempted');

    // ===== Auth guard — must pass before anything else =====
    const token = await getAccessToken();
    if (!token) {
        window.location.href = '/login.html';
        return;
    }
    // Set cookie for SSR on next page load
    try {
        const { data: { session } } = await initSupabase().auth.getSession();
        if (session) {
            document.cookie = 'sb-access-token=' + session.access_token + '; path=/; max-age=' + session.expires_in + '; SameSite=Lax' + (location.protocol === 'https:' ? '; Secure' : '');
        }
    } catch (e) {
        // Supabase unreachable — continue with token from localStorage
    }

    // ===== Dashboard state =====
    const currentDashboardId = document.getElementById('main-grid')?.dataset?.dashboardId || null;

    // ===== Logout button =====
    const logoutBtn = document.getElementById('btn-logout');
    if (logoutBtn) {
        logoutBtn.addEventListener('click', async function() {
            await signOut();
        });
    }

    // ===== Focus Google search on load =====
    document.getElementById('search-input').focus();

    // ===== Date display =====
    const now = new Date();
    document.getElementById('date-weekday').textContent = now.toLocaleDateString('es-ES', { weekday: 'long' }).toUpperCase();
    document.getElementById('date-numeric').textContent = now.toLocaleDateString('es-ES', { day: '2-digit', month: 'short', year: 'numeric' }).toUpperCase();

    // ===== Weather widget =====
    async function fetchWeather() {
        const lat = 37.34, lon = -5.84;
        const url = 'https://api.open-meteo.com/v1/forecast?latitude=' + lat + '&longitude=' + lon + '&current=temperature_2m,relative_humidity_2m,weather_code,wind_speed_10m&timezone=Europe%2FMadrid';
        try {
            const res = await fetch(url);
            const data = await res.json();
            const c = data.current;
            document.getElementById('weather-temp').textContent = Math.round(c.temperature_2m) + '°';
            document.getElementById('weather-humidity').textContent = '\u{1F4A7} ' + c.relative_humidity_2m + '%';
            document.getElementById('weather-wind').textContent = '\u{1F4A8} ' + Math.round(c.wind_speed_10m) + ' km/h';
            const { icon, desc } = getWeatherInfo(c.weather_code);
            document.getElementById('weather-icon').textContent = icon;
            document.getElementById('weather-desc').textContent = desc;
        } catch {
            document.getElementById('weather-desc').textContent = 'Sin conexión';
        }
    }

    function getWeatherInfo(code) {
        const map = {
            0: { icon: '\u2600\uFE0F', desc: 'Despejado' },
            1: { icon: '\u{1F324}\uFE0F', desc: 'Mayormente despejado' },
            2: { icon: '\u26C5', desc: 'Parcialmente nublado' },
            3: { icon: '\u2601\uFE0F', desc: 'Nublado' },
            45: { icon: '\u{1F32B}\uFE0F', desc: 'Niebla' },
            48: { icon: '\u{1F32B}\uFE0F', desc: 'Niebla helada' },
            51: { icon: '\u{1F326}\uFE0F', desc: 'Llovizna ligera' },
            53: { icon: '\u{1F326}\uFE0F', desc: 'Llovizna' },
            55: { icon: '\u{1F327}\uFE0F', desc: 'Llovizna intensa' },
            61: { icon: '\u{1F327}\uFE0F', desc: 'Lluvia ligera' },
            63: { icon: '\u{1F327}\uFE0F', desc: 'Lluvia moderada' },
            65: { icon: '\u{1F327}\uFE0F', desc: 'Lluvia intensa' },
            71: { icon: '\u{1F328}\uFE0F', desc: 'Nevada ligera' },
            73: { icon: '\u{1F328}\uFE0F', desc: 'Nevada moderada' },
            75: { icon: '\u2744\uFE0F', desc: 'Nevada intensa' },
            80: { icon: '\u{1F326}\uFE0F', desc: 'Chubascos ligeros' },
            81: { icon: '\u{1F327}\uFE0F', desc: 'Chubascos moderados' },
            82: { icon: '\u26C8\uFE0F', desc: 'Chubascos intensos' },
            95: { icon: '\u26C8\uFE0F', desc: 'Tormenta' },
            96: { icon: '\u26C8\uFE0F', desc: 'Tormenta con granizo' },
            99: { icon: '\u26C8\uFE0F', desc: 'Tormenta con granizo fuerte' },
        };
        return map[code] || { icon: '\u{1F321}\uFE0F', desc: 'Desconocido' };
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
            const url = 'https://speed.cloudflare.com/__down?bytes=' + bytes + '&ckSize=25000000&measId=' + Date.now();
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

    // ===== Dashboard navigation =====
    function switchDashboard(id) {
        window.location.href = '/dashboard/' + id;
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

    // ===== Dashboard tabs: add dashboard =====
    function openCreateDashboardModal() {
        modalTitle.textContent = 'Nuevo dashboard';
        modalFields.innerHTML = '<input type="text" id="input-dash-name" placeholder="Nombre del dashboard" autocomplete="off">';
        currentModalAction = async () => {
            const name = document.getElementById('input-dash-name').value.trim();
            if (!name) return;
            try {
                const d = await authFetch('POST', '/api/v1/dashboards', { name });
                switchDashboard(d.id);
            } catch (e) { alert('Error al crear dashboard'); }
        };
        openModal();
        setTimeout(() => document.getElementById('input-dash-name').focus(), 50);
    }

    const btnAddDash = document.getElementById('btn-add-dashboard');
    if (btnAddDash) btnAddDash.addEventListener('click', openCreateDashboardModal);

    // Dashboard tabs: rename on double-click
    document.getElementById('dashboard-tabs')?.addEventListener('dblclick', function(e) {
        const tab = e.target.closest('.tab[data-dashboard-id]');
        if (!tab) return;
        const id = tab.dataset.dashboardId;
        const currentName = tab.textContent.trim();
        modalTitle.textContent = 'Renombrar dashboard';
        modalFields.innerHTML = '<input type="text" id="input-dash-name" value="' + currentName.replace(/"/g, '&quot;') + '" autocomplete="off">';
        currentModalAction = async () => {
            const name = document.getElementById('input-dash-name').value.trim();
            if (!name) return;
            try {
                await authFetch('PUT', '/api/v1/dashboards/' + id, { name });
                location.reload();
            } catch (e) { alert('Error al renombrar'); }
        };
        openModal();
        setTimeout(() => { const inp = document.getElementById('input-dash-name'); inp.focus(); inp.select(); }, 50);
    });

    // Dashboard tabs: delete on right-click
    document.getElementById('dashboard-tabs')?.addEventListener('contextmenu', function(e) {
        const tab = e.target.closest('.tab[data-dashboard-id]');
        if (!tab) return;
        e.preventDefault();
        const id = tab.dataset.dashboardId;
        const name = tab.textContent.trim();
        if (!confirm('¿Eliminar el dashboard "' + name + '" y todo su contenido?')) return;
        authFetch('DELETE', '/api/v1/dashboards/' + id).then(() => {
            const remaining = document.querySelector('.tab[data-dashboard-id]:not([data-dashboard-id="' + id + '"])');
            if (remaining) switchDashboard(remaining.dataset.dashboardId);
            else window.location.href = '/';
        }).catch(() => alert('Error al eliminar'));
    });

    // ===== CRUD: Add link =====
    function openAddLinkModal(categoryEl) {
        const catId = categoryEl.dataset.categoryId;
        const catName = categoryEl.querySelector('.category-title').textContent.trim();
        modalTitle.textContent = 'Añadir enlace a ' + catName;
        modalFields.innerHTML = '\n            <input type="text" id="input-link-name" placeholder="Nombre (ej: GitHub)" autocomplete="off">\n            <input type="url" id="input-link-url" placeholder="URL (ej: https://github.com)" autocomplete="off" style="margin-top:0.5rem">\n        ';
        currentModalAction = async () => {
            const name = document.getElementById('input-link-name').value.trim();
            const url = document.getElementById('input-link-url').value.trim();
            if (!name || !url) return;
            try { new URL(url); } catch { return; }
            try {
                await authFetch('POST', '/api/v1/links', { name, url, categoryId: catId });
                location.reload();
            } catch (e) {
                alert('Error al crear enlace');
            }
        };
        openModal();
        setTimeout(() => document.getElementById('input-link-name').focus(), 50);
    }
    window.openAddLinkModal = openAddLinkModal;

    // ===== CRUD: Delete link =====
    async function deleteLink(id) {
        if (!confirm('¿Eliminar este enlace?')) return;
        try {
            await authFetch('DELETE', '/api/v1/links/' + id);
            location.reload();
        } catch (e) {
            alert('Error al eliminar');
        }
    }
    window.deleteLink = deleteLink;

    // ===== CRUD: Add category =====
    function openAddCategoryModal() {
        modalTitle.textContent = 'Añadir categoría';
        modalFields.innerHTML = '\n            <input type="text" id="input-cat-name" placeholder="Nombre de categoría" autocomplete="off">\n            <input type="text" id="input-cat-icon" placeholder="Emoji icono (ej: \uD83D\uDD27)" autocomplete="off" style="margin-top:0.5rem" maxlength="2">\n        ';
        currentModalAction = async () => {
            const name = document.getElementById('input-cat-name').value.trim();
            const icon = document.getElementById('input-cat-icon').value.trim() || '\uD83D\uDCC1';
            if (!name) return;
            try {
                await authFetch('POST', '/api/v1/categories', { name, icon, dashboardId: currentDashboardId });
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
            await authFetch('DELETE', '/api/v1/categories/' + id);
            location.reload();
        } catch (e) {
            alert('Error al eliminar');
        }
    }
    window.deleteCategory = deleteCategory;

    document.getElementById('btn-add-category').addEventListener('click', openAddCategoryModal);


})();
