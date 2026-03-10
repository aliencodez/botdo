'use strict';

const POLL_INTERVAL = 5000;

const $id = id => document.getElementById(id);

// ── State ──────────────────────────────────────────────────────────────────
let tasks = [];

// ── API helpers ────────────────────────────────────────────────────────────
async function api(method, path, body) {
  const opts = {
    method,
    headers: { 'Content-Type': 'application/json' },
  };
  if (body !== undefined) opts.body = JSON.stringify(body);
  const res = await fetch(path, opts);
  if (res.status === 204) return null;
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || res.statusText);
  return data;
}

// ── Render ─────────────────────────────────────────────────────────────────
function badge(cls, text) {
  return `<span class="badge badge-${cls}">${text}</span>`;
}

function renderTable() {
  const statusFilter = $id('filter-status').value;
  const agentFilter  = $id('filter-agent').value;

  let filtered = tasks;
  if (statusFilter) filtered = filtered.filter(t => t.status === statusFilter);
  if (agentFilter)  filtered = filtered.filter(t => t.agent  === agentFilter);

  const tbody = $id('task-tbody');
  if (filtered.length === 0) {
    tbody.innerHTML = `<tr><td colspan="5" class="empty">No tasks found.</td></tr>`;
    return;
  }

  tbody.innerHTML = filtered.map(t => `
    <tr>
      <td>${escHtml(t.title)}</td>
      <td>${t.description ? escHtml(t.description) : '<span style="color:#bbb">—</span>'}</td>
      <td>${badge(t.status, t.status.replace('_', ' '))}</td>
      <td>${t.agent ? badge('agent', t.agent) : '<span style="color:#bbb">—</span>'}</td>
      <td>
        <div class="actions">
          <button class="btn-danger" onclick="deleteTask(${t.id})">Delete</button>
        </div>
      </td>
    </tr>
  `).join('');
}

function escHtml(s) {
  return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');
}

// ── Data loading ───────────────────────────────────────────────────────────
async function loadTasks() {
  try {
    tasks = await api('GET', '/api/tasks');
    renderTable();
    $id('status-bar').textContent = `Last refreshed: ${new Date().toLocaleTimeString()}`;
  } catch (e) {
    $id('status-bar').textContent = `Error: ${e.message}`;
  }
}

// ── Actions ────────────────────────────────────────────────────────────────
async function createTask(e) {
  e.preventDefault();
  const title       = $id('new-title').value.trim();
  const description = $id('new-desc').value.trim();
  const agent       = $id('new-agent').value;
  if (!title) return;
  try {
    await api('POST', '/api/tasks', { title, description, agent: agent || '' });
    $id('new-title').value = '';
    $id('new-desc').value  = '';
    $id('new-agent').value = '';
    await loadTasks();
  } catch (e) {
    alert('Create failed: ' + e.message);
  }
}

async function deleteTask(id) {
  if (!confirm('Delete this task?')) return;
  try {
    await api('DELETE', `/api/tasks/${id}`);
    await loadTasks();
  } catch (e) {
    alert('Delete failed: ' + e.message);
  }
}

// ── Boot ───────────────────────────────────────────────────────────────────
document.addEventListener('DOMContentLoaded', () => {
  $id('task-form').addEventListener('submit', createTask);
  $id('filter-status').addEventListener('change', renderTable);
  $id('filter-agent').addEventListener('change', renderTable);

  loadTasks();
  setInterval(loadTasks, POLL_INTERVAL);
});
