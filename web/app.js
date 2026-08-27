'use strict';

const POLL_INTERVAL = 5000;

const COLUMNS = [
  { status: 'pending',     label: 'Pending',     color: '#f59e0b' },
  { status: 'in_progress', label: 'In Progress',  color: '#818cf8' },
  { status: 'done',        label: 'Done',         color: '#34d399' },
  { status: 'failed',      label: 'Failed',       color: '#f87171' },
];

// Deterministic color per space name
const SPACE_PALETTE = ['#818cf8','#f472b6','#34d399','#fb923c','#a78bfa','#38bdf8','#4ade80','#facc15'];
function spaceColor(name) {
  let h = 0;
  for (let i = 0; i < name.length; i++) h = (h * 31 + name.charCodeAt(i)) & 0xffffffff;
  return SPACE_PALETTE[Math.abs(h) % SPACE_PALETTE.length];
}

const $id = id => document.getElementById(id);

// ── State ──────────────────────────────────────────────────────────────────
let tasks          = [];
let spaces         = [];        // projects from API
let activeSpace    = 'all';     // 'all' | '0' (unassigned) | numeric string
let activeLogES    = null;
let appConfig      = { auth_required: false, checkout_url: '' };
let authenticated  = false;

// ── API ────────────────────────────────────────────────────────────────────
async function api(method, path, body) {
  const opts = { method, headers: { 'Content-Type': 'application/json' } };
  if (body !== undefined) opts.body = JSON.stringify(body);
  const res = await fetch(path, opts);
  if (res.status === 204) return null;
  const data = await res.json();
  if (res.status === 401) {
    openAccessModal('That token was not accepted. Check it and try again.');
  }
  if (!res.ok) throw new Error(data.error || res.statusText);
  return data;
}

async function loadConfig() {
  const res = await fetch('/api/config');
  if (!res.ok) throw new Error('Unable to load workspace configuration');
  appConfig = await res.json();

  const upgrade = $id('upgrade-link');
  if (appConfig.checkout_url) {
    upgrade.href = appConfig.checkout_url;
    upgrade.style.display = '';
  }
  return true;
}

// ── Helpers ────────────────────────────────────────────────────────────────
function esc(s) {
  return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');
}

function badge(cls, text) {
  return `<span class="badge badge-${cls}">${esc(String(text))}</span>`;
}

// ── Log parsing & rendering helpers ────────────────────────────────────────
// Convert a raw text line (plain or JSON) into a simple event object that can
// be rendered in the timeline.  We treat a few special prefixes specially
// (dispatch/stderr) and fall back to a raw message when parsing fails.
function parseLogLine(line) {
  if (line.startsWith('[dispatch]')) {
    return {type: 'dispatch', text: line.slice('[dispatch]'.length).trim()};
  }
  if (line.startsWith('[stderr]')) {
    return {type: 'stderr', text: line.slice('[stderr]'.length).trim()};
  }
  try {
    const obj = JSON.parse(line);
    if (obj && obj.type) return {type: obj.type, data: obj};
  } catch (e) {
    // not JSON
  }
  return {type: 'message', text: line};
}

// Render a parsed event into a DOM element.  We add classes based on the type
// to allow CSS styling, and print out important fields for JSON-based events.
function renderLogEvent(ev) {
  const el = document.createElement('div');
  el.className = 'log-event log-' + ev.type.replace(/\W/g, '-');

  if (ev.type === 'dispatch' || ev.type === 'stderr' || ev.type === 'message') {
    el.textContent = ev.text;
  } else if (ev.data) {
    switch (ev.type) {
      case 'assistant':
        el.textContent = ev.data.content || '';
        break;
      case 'tool_use':
        el.innerHTML =
          `<span class="log-tool-name">${esc(ev.data.tool)}</span>` +
          ` <span class="log-tool-input">${esc(ev.data.input)}</span>`;
        break;
      case 'tool_result':
        el.innerHTML =
          `<span class="log-tool-name">${esc(ev.data.tool)}</span>` +
          ` <span class="log-tool-output"><pre>${esc(ev.data.output)}</pre></span>`;
        break;
      case 'result':
        el.textContent = ev.data.output || '';
        if (ev.data.is_error) el.classList.add('log-error');
        break;
      case 'system':
        if (ev.data.subtype === 'init') {
          const parts = [];
          if (ev.data.model) parts.push(`model=${ev.data.model}`);
          if (ev.data.starter) parts.push(`starter=${ev.data.starter}`);
          el.textContent = `[system init] ${parts.join(' ')}`;
        } else {
          el.innerHTML = `<pre>${esc(JSON.stringify(ev.data, null, 2))}</pre>`;
        }
        break;
      default:
        el.innerHTML = `<pre>${esc(JSON.stringify(ev.data, null, 2))}</pre>`;
    }
  } else {
    // fallback if we somehow received an event without data/text
    el.textContent = ev.text || '';
  }
  return el;
}

// Append a raw line to the given container, parsing + rendering in one step.
function appendLogLine(line, container) {
  const ev = parseLogLine(line);
  const node = renderLogEvent(ev);
  container.appendChild(node);
}

function spaceName(id) {
  const s = spaces.find(s => s.id === id);
  return s ? s.name : String(id);
}

function visibleTasks() {
  const agentFilter = $id('filter-agent').value;
  let list = tasks;
  if (activeSpace === '0') {
    list = list.filter(t => !t.project_id);
  } else if (activeSpace !== 'all') {
    const pid = parseInt(activeSpace, 10);
    list = list.filter(t => t.project_id === pid);
  }
  if (agentFilter) list = list.filter(t => t.agent === agentFilter);
  return list;
}

// ── Render: space nav ──────────────────────────────────────────────────────
function renderSpaceNav() {
  // Static item counts
  $id('count-all').textContent = tasks.length;
  $id('count-0').textContent   = tasks.filter(t => !t.project_id).length;

  // Remove old dynamic items
  $id('space-nav').querySelectorAll('[data-dynamic]').forEach(el => el.remove());

  // Append one li per space
  spaces.forEach(s => {
    const count  = tasks.filter(t => t.project_id === s.id).length;
    const color  = spaceColor(s.name);
    const active = activeSpace === String(s.id);
    const li     = document.createElement('li');
    li.className          = 'space-item' + (active ? ' active' : '');
    li.dataset.id         = String(s.id);
    li.dataset.dynamic    = '1';
    li.onclick            = () => selectSpace(String(s.id));
    li.innerHTML = `
      <span class="space-dot" style="background:${color}"></span>
      <span class="space-name">${esc(s.name)}</span>
      <span class="space-count">${count}</span>`;
    $id('space-nav').appendChild(li);
  });

  // Sync active state on static items
  $id('space-nav').querySelectorAll('.space-item:not([data-dynamic])').forEach(li => {
    li.classList.toggle('active', li.dataset.id === activeSpace);
  });

  // Populate task modal space select
  const sel = $id('new-space-select');
  if (sel) {
    const cur = sel.value;
    sel.innerHTML = '<option value="">Unassigned</option>' +
      spaces.map(s => `<option value="${s.id}">${esc(s.name)}</option>`).join('');
    if (cur) sel.value = cur;
  }
}

// ── Render: main header ────────────────────────────────────────────────────
function renderHeader() {
  let title = 'All Tasks', workdir = '', showDelete = false;

  if (activeSpace === '0') {
    title = 'Unassigned';
  } else if (activeSpace !== 'all') {
    const s = spaces.find(s => String(s.id) === activeSpace);
    if (s) { title = s.name; workdir = s.work_dir || ''; showDelete = true; }
  }

  $id('space-title').textContent   = title;
  $id('space-workdir').textContent = workdir;
  $id('btn-delete-space').style.display = showDelete ? '' : 'none';
}

// ── Render: kanban ─────────────────────────────────────────────────────────
function renderKanban() {
  const list = visibleTasks();
  $id('kanban-board').innerHTML = COLUMNS.map(col => {
    const colTasks = list.filter(t => t.status === col.status);
    const cards = colTasks.length === 0
      ? `<div class="kanban-empty">Empty</div>`
      : colTasks.map(renderCard).join('');
    return `
      <div class="kanban-col" style="--col-color:${col.color}">
        <div class="kanban-col-header">
          <span class="kanban-col-title">${col.label}</span>
          <span class="kanban-col-count">${colTasks.length}</span>
        </div>
        <div class="kanban-cards">${cards}</div>
      </div>`;
  }).join('');
}

function renderCard(t) {
  const descHtml = t.description
    ? `<p class="card-desc">${esc(t.description)}</p>` : '';

  const tags = [];
  if (t.agent)      tags.push(badge('agent', t.agent));
  // Show space badge only in "All Tasks" view
  if (t.project_id && activeSpace === 'all') tags.push(badge('space', spaceName(t.project_id)));
  const p = t.permissions || {};
  if (p.allow_write) tags.push(badge('perm', 'write'));
  if (p.allow_edit)  tags.push(badge('perm', 'edit'));
  const tagsHtml = tags.length ? `<div class="card-tags">${tags.join('')}</div>` : '';

  const retryBtn = t.status === 'failed'
    ? `<button class="btn-card btn-retry-card" onclick="event.stopPropagation();retryTask(${t.id})">Retry</button>` : '';

  return `
    <div class="task-card" onclick="viewTask(${t.id})">
      <p class="card-title">${esc(t.title)}</p>
      ${descHtml}${tagsHtml}
      <div class="card-footer">
        <div class="card-actions">
          ${retryBtn}
          <button class="btn-card btn-delete-card" onclick="event.stopPropagation();deleteTask(${t.id})">Delete</button>
        </div>
      </div>
    </div>`;
}

// ── Space selection ────────────────────────────────────────────────────────
function selectSpace(id) {
  activeSpace = id;
  renderSpaceNav();
  renderHeader();
  renderKanban();
}

// ── Data loading ───────────────────────────────────────────────────────────
async function loadSpaces() {
  spaces = await api('GET', '/api/projects');
  renderSpaceNav();
  renderHeader();
}

async function loadTasks() {
  try {
    tasks = await api('GET', '/api/tasks');
    authenticated = true;
    renderSpaceNav();
    renderKanban();
    setSyncStatus(false);
    return true;
  } catch {
    authenticated = false;
    setSyncStatus(true);
    return false;
  }
}

async function loadAll() {
  try { await loadSpaces(); } catch { /* non-fatal */ }
  await loadTasks();
}

function setSyncStatus(isError) {
  const el = $id('sync-status');
  if (!el) return;
  if (isError) {
    el.textContent = 'sync error';
    el.className   = 'sync-status sync-error';
  } else {
    el.textContent = `synced ${new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`;
    el.className   = 'sync-status';
  }
}

// ── Modals ─────────────────────────────────────────────────────────────────
function openModal(id) {
  $id(id).style.display  = 'flex';
  document.body.style.overflow = 'hidden';
}

function closeModal(id) {
  $id(id).style.display  = 'none';
  document.body.style.overflow = '';
}

function closeModalOnOverlay(e, id) {
  if (e.target === $id(id)) closeModal(id);
}

function openAccessModal(message) {
  $id('access-error').textContent = message || '';
  $id('access-modal').style.display = 'flex';
  document.body.style.overflow = 'hidden';
  setTimeout(() => $id('access-token')?.focus(), 50);
}

async function saveAccessToken(e) {
  e.preventDefault();
  const token = $id('access-token').value.trim();
  if (!token) return;
  try {
    const res = await fetch('/api/session', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token }),
    });
    if (!res.ok) {
      const data = await res.json();
      throw new Error(data.error || 'That token was not accepted');
    }
    authenticated = true;
    $id('access-token').value = '';
    closeModal('access-modal');
    await loadAll();
  } catch (err) {
    $id('access-error').textContent = err.message;
  }
}

function openNewTaskModal() {
  const sel = $id('new-space-select');
  if (sel) sel.value = (activeSpace !== 'all' && activeSpace !== '0') ? activeSpace : '';
  openModal('task-modal');
  setTimeout(() => $id('new-title')?.focus(), 50);
}

// ── Space CRUD ─────────────────────────────────────────────────────────────
async function createSpace(e) {
  e.preventDefault();
  const name    = $id('new-space-name').value.trim();
  const workDir = $id('new-space-workdir').value.trim();
  if (!name) return;
  try {
    const s = await api('POST', '/api/projects', { name, work_dir: workDir });
    $id('new-space-name').value    = '';
    $id('new-space-workdir').value = '';
    closeModal('space-modal');
    await loadSpaces();
    selectSpace(String(s.id));
  } catch (err) {
    alert('Create space failed: ' + err.message);
  }
}

async function deleteCurrentSpace() {
  if (activeSpace === 'all' || activeSpace === '0') return;
  const pid = parseInt(activeSpace, 10);
  const s   = spaces.find(s => s.id === pid);
  if (!confirm(`Delete space "${s?.name}"? Tasks will become unassigned.`)) return;
  try {
    await api('DELETE', `/api/projects/${pid}`);
    selectSpace('all');
    await loadAll();
  } catch (err) {
    alert('Delete space failed: ' + err.message);
  }
}

// ── Task CRUD ──────────────────────────────────────────────────────────────
async function createTask(e) {
  e.preventDefault();
  const title       = $id('new-title').value.trim();
  const description = $id('new-desc').value.trim();
  const agent       = $id('new-agent').value;
  const spaceVal    = $id('new-space-select').value;
  const project_id  = spaceVal ? parseInt(spaceVal, 10) : null;
  const permissions = {
    allow_write: $id('perm-write').checked,
    allow_edit:  $id('perm-edit').checked,
  };
  if (!title) return;
  try {
    await api('POST', '/api/tasks', { title, description, agent: agent || '', project_id, permissions });
    $id('new-title').value    = '';
    $id('new-desc').value     = '';
    $id('new-agent').value    = '';
    $id('perm-write').checked = false;
    $id('perm-edit').checked  = false;
    togglePermissions();
    closeModal('task-modal');
    await loadTasks();
  } catch (err) {
    alert('Create task failed: ' + err.message);
  }
}

function togglePermissions() {
  $id('agent-permissions').style.display = $id('new-agent').value ? '' : 'none';
}

async function retryTask(id) {
  try {
    await api('POST', `/api/tasks/${id}/retry`);
    await loadTasks();
  } catch (err) {
    alert('Retry failed: ' + err.message);
  }
}

async function deleteTask(id) {
  if (!confirm('Delete this task?')) return;
  try {
    await api('DELETE', `/api/tasks/${id}`);
    await loadTasks();
  } catch (err) {
    alert('Delete failed: ' + err.message);
  }
}

// ── Task detail drawer ─────────────────────────────────────────────────────
async function viewTask(id) {
  const t = await api('GET', `/api/tasks/${id}`);
  openDetail(t);
}

function openDetail(t) {
  $id('detail-title').textContent = t.title;
  $id('detail-desc').textContent  = t.description || '';

  const meta = [badge(t.status, t.status.replace('_', ' '))];
  if (t.agent)      meta.push(badge('agent', t.agent));
  if (t.project_id) meta.push(badge('space', spaceName(t.project_id)));
  const p = t.permissions || {};
  if (p.allow_write) meta.push(badge('perm', 'write'));
  if (p.allow_edit)  meta.push(badge('perm', 'edit'));
  $id('detail-meta').innerHTML = meta.join('');

  openModal('detail-overlay');
  loadDetailLogs(t);
}

function loadDetailLogs(t) {
  if (activeLogES) { activeLogES.close(); activeLogES = null; }

  const logEl  = $id('detail-logs');
  const statEl = $id('log-status');
  logEl.innerHTML    = '<span class="log-empty">Loading…</span>';
  statEl.textContent = '';

  // helper to scroll after adding
  function scrollBottom() {
    logEl.scrollTop = logEl.scrollHeight;
  }

  if (t.status === 'in_progress') {
    statEl.textContent = '● live';
    const es = new EventSource(`/api/tasks/${t.id}/logs?stream=1`);
    activeLogES = es;
    let hasLines = false;
    es.onmessage = ev => {
      if (!hasLines) { logEl.innerHTML = ''; hasLines = true; }
      appendLogLine(ev.data, logEl);
      scrollBottom();
    };
    es.onerror = () => { es.close(); activeLogES = null; statEl.textContent = ''; };
  } else {
    api('GET', `/api/tasks/${t.id}/logs`).then(data => {
      const lines = data.lines || [];
      if (lines.length) {
        logEl.innerHTML = '';
        lines.forEach(l => appendLogLine(l, logEl));
        scrollBottom();
      } else {
        logEl.innerHTML = '<span class="log-empty">No logs.</span>';
      }
    }).catch(() => {
      logEl.innerHTML = '<span class="log-empty">Failed to load logs.</span>';
    });
  }
}

function closeDetail() {
  closeModal('detail-overlay');
  if (activeLogES) { activeLogES.close(); activeLogES = null; }
  $id('log-status').textContent = '';
}

function closeDetailOnOverlay(e) {
  if (e.target === $id('detail-overlay')) closeDetail();
}

// ── Boot ───────────────────────────────────────────────────────────────────
document.addEventListener('DOMContentLoaded', () => {
  $id('filter-agent').addEventListener('change', renderKanban);
  document.addEventListener('keydown', e => {
    if (e.key !== 'Escape') return;
    if ($id('detail-overlay').style.display !== 'none') closeDetail();
    else if ($id('task-modal').style.display !== 'none')  closeModal('task-modal');
    else if ($id('space-modal').style.display !== 'none') closeModal('space-modal');
  });

  loadConfig()
    .then(ready => { if (ready) return loadAll(); })
    .catch(() => setSyncStatus(true));
  setInterval(() => {
    if (!appConfig.auth_required || authenticated) loadAll();
  }, POLL_INTERVAL);
});
