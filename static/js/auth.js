// ===== Supabase Auth module for myhomeweb =====
const SUPABASE_URL = 'https://agtkcnxmlbccbwmsuxdz.supabase.co';
const SUPABASE_KEY = 'sb_publishable_W_Nc9hzh_M-uMNMbTPeE5w_6IKDgPd3';

let _supabase = null;

function initSupabase() {
    if (!_supabase) {
        _supabase = supabase.createClient(SUPABASE_URL, SUPABASE_KEY);
    }
    return _supabase;
}

async function getAccessToken() {
    const { data: { session } } = await initSupabase().auth.getSession();
    return session?.access_token || null;
}

function hasSession() {
    return !!localStorage.getItem('sb-agtkcnxmlbccbwmsuxdz-auth-token');
}

async function requireAuth() {
    const token = await getAccessToken();
    if (!token) {
        window.location.href = '/login.html';
        throw new Error('No session');
    }
    return token;
}

async function signOut() {
    await initSupabase().auth.signOut();
    document.cookie = 'sb-access-token=; path=/; max-age=0; SameSite=Lax' + (location.protocol === 'https:' ? '; Secure' : '');
    localStorage.removeItem('sb-agtkcnxmlbccbwmsuxdz-auth-token');
    window.location.href = '/login.html';
}

async function authFetch(method, path, body) {
    let token = await getAccessToken();
    if (!token) {
        // No token at all — try refreshing once before failing
        const { data, error } = await initSupabase().auth.refreshSession();
        if (data.session && !error) {
            token = data.session.access_token;
            document.cookie = 'sb-access-token=' + token + '; path=/; max-age=' + data.session.expires_in + '; SameSite=Lax' + (location.protocol === 'https:' ? '; Secure' : '');
        } else {
            signOut();
            throw new Error('No session');
        }
    }
    const options = {
        method,
        headers: {
            'Authorization': 'Bearer ' + token,
        },
    };
    if (body) {
        options.headers['Content-Type'] = 'application/json';
        options.body = JSON.stringify(body);
    }
    const res = await fetch(path, options);
    if (res.status === 401) {
        // Try refreshing session
        const { data, error } = await initSupabase().auth.refreshSession();
        if (data.session && !error) {
            token = data.session.access_token;
            document.cookie = 'sb-access-token=' + token + '; path=/; max-age=' + data.session.expires_in + '; SameSite=Lax' + (location.protocol === 'https:' ? '; Secure' : '');
            options.headers['Authorization'] = 'Bearer ' + token;
            const retryRes = await fetch(path, options);
            if (!retryRes.ok) {
                const err = new Error('API error: ' + retryRes.status);
                err.status = retryRes.status;
                throw err;
            }
            return retryRes.status === 204 ? null : retryRes.json();
        }
        signOut();
        throw new Error('Session expired');
    }
    if (!res.ok) {
        const err = new Error('API error: ' + res.status);
        err.status = res.status;
        throw err;
    }
    return res.status === 204 ? null : res.json();
}
