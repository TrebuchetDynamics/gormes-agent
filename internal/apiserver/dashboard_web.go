package apiserver

import (
	"html/template"
	"net/http"
)

var dashboardTemplates = template.Must(template.New("").Parse(tmplDashboard))

func (s *Server) handleWebDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.renderDashboardPage(w)
}

func (s *Server) renderDashboardPage(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	dashboardTemplates.Execute(w, nil)
}

const tmplDashboard = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Gormes Dashboard</title>
<script src="https://unpkg.com/htmx.org@2.0.4"></script>
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #0d1117; color: #e6edf3; display: flex; min-height: 100vh; }
nav { width: 220px; background: #161b22; padding: 1rem; border-right: 1px solid #30363d; flex-shrink: 0; }
nav h1 { font-size: 1.1rem; margin-bottom: 1.5rem; color: #58a6ff; }
nav a { display: block; padding: 0.5rem 0.75rem; color: #8b949e; text-decoration: none; border-radius: 6px; margin-bottom: 0.25rem; cursor: pointer; font-size: 0.9rem; }
nav a:hover { background: #1f2937; color: #e6edf3; }
nav a.active { background: #1f2937; color: #f0f6fc; border-left: 3px solid #58a6ff; }
main { flex: 1; padding: 1.5rem; overflow-y: auto; max-height: 100vh; }
table { width: 100%; border-collapse: collapse; }
th, td { text-align: left; padding: 0.5rem 0.75rem; border-bottom: 1px solid #21262d; font-size: 0.9rem; }
th { color: #8b949e; font-weight: 600; font-size: 0.8rem; text-transform: uppercase; }
tr:hover td { background: #161b22; }
tr.clickable td { cursor: pointer; }
.header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.5rem; }
.header h2 { font-size: 1.3rem; }
.badge { background: #1f2937; padding: 0.2rem 0.6rem; border-radius: 12px; font-size: 0.8rem; color: #8b949e; }
code { background: #1f2937; padding: 0.1rem 0.3rem; border-radius: 3px; font-size: 0.85rem; }
.back { color: #58a6ff; text-decoration: none; font-size: 0.9rem; cursor: pointer; }
.back:hover { text-decoration: underline; }
.card { background: #161b22; border: 1px solid #30363d; border-radius: 8px; padding: 1rem; margin-bottom: 1rem; }
.card h3 { font-size: 1rem; margin-bottom: 0.75rem; color: #e6edf3; }
.card-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; }
.stat { display: flex; justify-content: space-between; padding: 0.4rem 0; border-bottom: 1px solid #21262d; font-size: 0.9rem; }
.stat:last-child { border-bottom: none; }
.stat-label { color: #8b949e; }
.stat-value { color: #e6edf3; }
.plugin-card { background: #0d1117; border: 1px solid #21262d; border-radius: 6px; padding: 0.75rem; margin-bottom: 0.5rem; }
.plugin-card .header-row { display: flex; justify-content: space-between; align-items: center; }
.plugin-card .name { font-weight: 600; }
.plugin-card .version { color: #8b949e; font-size: 0.85rem; }
.plugin-card .desc { color: #8b949e; font-size: 0.85rem; margin-top: 0.3rem; }
.plugin-card .meta { font-size: 0.8rem; margin-top: 0.3rem; }
.tag { display: inline-block; padding: 0.1rem 0.4rem; border-radius: 4px; font-size: 0.75rem; margin-right: 0.3rem; }
.tag-bundled { background: #1f3a5f; color: #58a6ff; }
.tag-user { background: #1a3a2a; color: #3fb950; }
.tag-disabled { background: #3d1f1f; color: #f85149; }
.tag-loaded { background: #1a3a2a; color: #3fb950; }
.tag-invalid { background: #3d1f1f; color: #f85149; }
.tag-p1 { background: #3d1f1f; color: #f85149; }
.tag-p2 { background: #1f3a5f; color: #58a6ff; }
.log-entry { padding: 0.3rem 0; border-bottom: 1px solid #21262d; font-family: monospace; font-size: 0.8rem; display: flex; gap: 0.75rem; }
.log-time { color: #8b949e; flex-shrink: 0; }
.log-level { flex-shrink: 0; width: 4.5rem; }
.log-level-info { color: #58a6ff; }
.log-level-warn { color: #d29922; }
.log-level-error { color: #f85149; }
.log-msg { color: #e6edf3; }
.env-card { background: #0d1117; border: 1px solid #21262d; border-radius: 6px; padding: 0.6rem; margin-bottom: 0.4rem; display: flex; justify-content: space-between; align-items: center; }
.env-key { font-family: monospace; font-size: 0.85rem; color: #e6edf3; }
.env-status { display: flex; align-items: center; gap: 0.5rem; }
.env-dot { width: 8px; height: 8px; border-radius: 50%; display: inline-block; }
.env-dot-set { background: #3fb950; }
.env-dot-unset { background: #8b949e; }
.error-msg { color: #f85149; font-size: 0.9rem; padding: 0.5rem 0; }
@media (max-width: 768px) { nav { width: 50px; } nav span { display: none; } .card-grid { grid-template-columns: 1fr; } }
</style>
</head>
<body>
<nav>
<h1>Gormes</h1>
<a onclick="showPage('sessions')" id="nav-sessions" class="active"><span>☰ Sessions</span></a>
<a onclick="showPage('config')" id="nav-config"><span>⚙ Config</span></a>
<a onclick="showPage('skills')" id="nav-skills"><span>📋 Skills</span></a>
<a onclick="showPage('logs')" id="nav-logs"><span>📊 Logs</span></a>
</nav>
<main id="main-content"><div class="header"><h2>Sessions</h2><span class="badge" id="session-count">0</span></div>
<table><thead><tr><th>Session ID</th><th>Title</th><th>Source</th><th>Created</th><th>Tokens</th></tr></thead><tbody id="sessions-tbody"></tbody></table></main>
<script>
var apiKey = '';
function showPage(page) {
  document.querySelectorAll('nav a').forEach(function(a) { a.classList.remove('active'); });
  var navEl = document.getElementById('nav-' + page);
  if (navEl) navEl.classList.add('active');
  if (page === 'sessions') loadSessions();
  else if (page === 'config') loadConfig();
  else if (page === 'skills') loadSkills();
  else if (page === 'logs') loadLogs();
}
// ---- SESSIONS ----
function loadSessions() {
  apiFetch('/api/sessions?limit=50', function(data) {
    var sessions = data.sessions || [];
    var main = document.getElementById('main-content');
    var cnt = document.getElementById('session-count');
    if (cnt) cnt.textContent = sessions.length;
    main.innerHTML = '<div class="header"><h2>Sessions</h2><span class="badge">' + sessions.length + '</span></div><table><thead><tr><th>Session ID</th><th>Title</th><th>Source</th><th>Created</th><th>Tokens</th></tr></thead><tbody>' +
      sessions.map(function(s) { return '<tr class="clickable" onclick="showSession(\'' + esc(s.id) + '\')"><td><code>' + esc(s.id) + '</code></td><td>' + esc(s.title || '(untitled)') + '</td><td>' + esc(s.source || '-') + '</td><td>' + (s.created ? fmtTime(s.created) : '-') + '</td><td>' + (s.tokens || 0) + '</td></tr>'; }).join('') +
      '</tbody></table>';
  });
}
function showSession(id) {
  var main = document.getElementById('main-content');
  main.innerHTML = '<div class="header"><h2>Session <code>' + esc(id) + '</code></h2><a class="back" onclick="loadSessions()">← Back</a></div><div id="session-detail">Loading...</div>';
  apiFetch('/api/sessions/' + encodeURIComponent(id), function(s) {
    document.getElementById('session-detail').innerHTML = '<table><tr><th>Field</th><th>Value</th></tr><tr><td>Title</td><td>' + esc(s.title || '(none)') + '</td></tr><tr><td>Source</td><td>' + esc(s.source || '-') + '</td></tr><tr><td>Created</td><td>' + (s.created ? fmtTime(s.created) : '-') + '</td></tr><tr><td>Updated</td><td>' + (s.updated ? fmtTime(s.updated) : '-') + '</td></tr><tr><td>Tokens In</td><td>' + (s.tokens_in || 0) + '</td></tr><tr><td>Tokens Out</td><td>' + (s.tokens_out || 0) + '</td></tr></table>';
  });
}
// ---- CONFIG ----
function loadConfig() {
  var main = document.getElementById('main-content');
  main.innerHTML = '<div class="header"><h2>Config</h2></div><div class="card-grid"><div class="card" id="cfg-model"><h3>Model</h3><p>Loading...</p></div><div class="card" id="cfg-providers"><h3>Providers</h3><p>Loading...</p></div></div>';
  apiFetch('/api/model/info', function(m) {
    document.getElementById('cfg-model').innerHTML = '<h3>Model</h3><div class="stat"><span class="stat-label">Model</span><span class="stat-value">' + esc(m.model) + '</span></div><div class="stat"><span class="stat-label">Provider</span><span class="stat-value">' + esc(m.provider) + '</span></div><div class="stat"><span class="stat-label">Tools</span><span class="stat-value">' + (m.capabilities && m.capabilities.supports_tools ? '✓' : '✗') + '</span></div><div class="stat"><span class="stat-label">Context Length</span><span class="stat-value">' + (m.effective_context_length || 'auto') + '</span></div>';
  });
  apiFetchAuth('/api/providers/oauth', function(data) {
    var provs = data.providers || [];
    var html = '<h3>OAuth Providers</h3>';
    provs.forEach(function(p) {
      var ok = p.status && p.status.logged_in;
      html += '<div class="env-card"><span class="env-key">' + esc(p.name) + '</span><span class="env-status">' +
        (ok ? '<span class="env-dot env-dot-set"></span> Connected <span class="badge">' + esc(p.status.token_preview || '') + '</span>' : '<span class="env-dot env-dot-unset"></span> Not connected') +
        '</span></div>';
    });
    document.getElementById('cfg-providers').innerHTML = html;
  });
}
// ---- SKILLS ----
function loadSkills() {
  var main = document.getElementById('main-content');
  main.innerHTML = '<div class="header"><h2>Skills &amp; Plugins</h2></div><div id="skills-content"><p>Loading...</p></div>';
  apiFetchAuth('/api/dashboard/plugins', function(data) {
    var plugins = data.plugins || [];
    var caps = data.capabilities || [];
    var html = '<p><span class="badge">' + plugins.length + ' plugins</span> <span class="badge">' + caps.length + ' capabilities</span></p>';
    plugins.forEach(function(p) {
      var stateTag = 'tag-' + (p.state === 'loaded' || p.state === 'enabled' ? 'loaded' : p.state === 'invalid' || p.state === 'malformed' ? 'invalid' : 'disabled');
      var srcTag = 'tag-' + (p.source === 'user' ? 'user' : 'bundled');
      html += '<div class="plugin-card"><div class="header-row"><span class="name">' + esc(p.name) + '</span><span><span class="tag ' + stateTag + '">' + esc(p.state) + '</span><span class="tag ' + srcTag + '">' + esc(p.source) + '</span><span class="version">v' + esc(p.version || '0') + '</span></span></div>' +
        (p.description ? '<div class="desc">' + esc(p.description) + '</div>' : '') +
        (p.manifest && p.manifest.requires_env && p.manifest.requires_env.length ? '<div class="meta">Requires: ' + p.manifest.requires_env.map(function(e) { return '<code>' + esc(e) + '</code>'; }).join(', ') + '</div>' : '') +
        '</div>';
    });
    if (!plugins.length) html += '<p class="error-msg">No plugins found. Plugin directory may be empty.</p>';
    document.getElementById('skills-content').innerHTML = html;
  });
}
// ---- LOGS ----
function loadLogs() {
  var main = document.getElementById('main-content');
  main.innerHTML = '<div class="header"><h2>Logs</h2><a class="back" onclick="loadLogs()">↻ Refresh</a></div><div id="logs-content"><p>Loading...</p></div>';
  apiFetch('/api/logs', function(data) {
    var entries = data.entries || [];
    if (!entries.length) { document.getElementById('logs-content').innerHTML = '<p class="error-msg">No log entries yet.</p>'; return; }
    var levelClass = { info: 'log-level-info', warn: 'log-level-warn', error: 'log-level-error' };
    document.getElementById('logs-content').innerHTML = entries.slice().reverse().map(function(e) {
      return '<div class="log-entry"><span class="log-time">' + esc(e.time) + '</span><span class="log-level ' + (levelClass[e.level] || '') + '">' + esc(e.level.toUpperCase()) + '</span><span class="log-msg">' + esc(e.message) + '</span></div>';
    }).join('');
    document.getElementById('logs-content').innerHTML += '<p style="color:#8b949e;font-size:0.85rem;margin-top:0.5rem">Showing last ' + entries.length + ' entries</p>';
  });
}
// ---- UTILITIES ----
function apiFetch(url, cb) {
  var xhr = new XMLHttpRequest();
  xhr.open('GET', url, true);
  xhr.onload = function() { try { cb(JSON.parse(xhr.responseText)); } catch(e) {} };
  xhr.onerror = function() { document.getElementById('main-content').innerHTML = '<p class="error-msg">Failed to load: ' + esc(url) + '</p>'; };
  xhr.send();
}
function apiFetchAuth(url, cb) {
  var xhr = new XMLHttpRequest();
  xhr.open('GET', url, true);
  if (apiKey) xhr.setRequestHeader('Authorization', 'Bearer ' + apiKey);
  xhr.onload = function() {
    if (xhr.status === 401 && !apiKey) {
      apiKey = prompt('Enter API key for ' + url + ':');
      if (apiKey) { apiFetchAuth(url, cb); return; }
    }
    try { cb(JSON.parse(xhr.responseText)); } catch(e) { document.getElementById('main-content').innerHTML = '<p class="error-msg">Auth required for this page. Set GORMES_DASHBOARD_API_KEY env var.</p>'; }
  };
  xhr.send();
}
function fmtTime(ts) { try { return new Date(ts * 1000).toLocaleString(); } catch(e) { return ts; } }
function esc(s) { var d = document.createElement('div'); d.textContent = s || ''; return d.innerHTML; }
loadSessions();
</script>
</body>
</html>`
