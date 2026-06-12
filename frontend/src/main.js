import './style.css';
import { t, ruleText, causeText, evidenceText, actionText } from './i18n';
import { GetStatus, RunScan, OpenLogFolder, RunAction, ListRollbackRecords, RollbackService } from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';

const LANG_KEY = 'diagnostic-studio-lang';

const state = {
  lang: localStorage.getItem(LANG_KEY) || 'zh',
  status: null,
  report: null,
  scanning: false,
  progress: { done: 0, total: 1 },
  error: '',
  tab: 'overview',
  rollbackRecords: [],
  actionResults: {}, // action key -> ActionResult
  actionBusy: '',    // action key currently executing
  confirm: null,     // {key, action} awaiting user confirmation
};

const TABS = ['overview', 'actions', 'processes', 'startup', 'services', 'software', 'events'];

const app = document.getElementById('app');

function esc(s) {
  return String(s ?? '').replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
}

async function init() {
  try {
    state.status = await GetStatus();
  } catch (e) {
    state.error = String(e);
  }
  try {
    state.rollbackRecords = (await ListRollbackRecords()) || [];
  } catch { /* records are optional context */ }
  EventsOn('scan:progress', p => {
    state.progress = p;
    const bar = document.querySelector('.progress-fill');
    const label = document.querySelector('.progress-label');
    if (bar) bar.style.width = `${(p.done / p.total) * 100}%`;
    if (label) label.textContent = `${p.done} / ${p.total}`;
  });
  render();
}

async function startScan(mode) {
  if (state.scanning) return;
  state.scanning = true;
  state.error = '';
  state.progress = { done: 0, total: mode === 'deep' ? 36 : 15 };
  render();
  try {
    state.report = await RunScan(mode);
    state.tab = 'overview';
  } catch (e) {
    state.error = String(e);
  }
  state.scanning = false;
  render();
}

function setLang(lang) {
  state.lang = lang;
  localStorage.setItem(LANG_KEY, lang);
  render();
}

function scoreClass(severity) {
  return { good: 'score-good', warning: 'score-warning', critical: 'score-critical' }[severity] || 'score-good';
}

// Generic polyline sparkline over sample values.
function sparkline(values, { max, lines = [] } = {}) {
  if (!values || values.length < 2) return '';
  const w = 560, h = 110, pad = 6;
  const top = (max ?? Math.max(...values)) * 1.05 || 1;
  const pts = values.map((v, i) => {
    const x = pad + (i / (values.length - 1)) * (w - pad * 2);
    const y = h - pad - (Math.max(0, v) / top) * (h - pad * 2);
    return `${x.toFixed(1)},${y.toFixed(1)}`;
  }).join(' ');
  const guides = lines.map(l => {
    const y = h - pad - (l.value / top) * (h - pad * 2);
    return `<line x1="${pad}" y1="${y}" x2="${w - pad}" y2="${y}" class="${l.cls}"/>`;
  }).join('');
  return `
    <svg viewBox="0 0 ${w} ${h}" class="spark" preserveAspectRatio="none">
      ${guides}
      <polyline points="${pts}" class="spark-line"/>
    </svg>`;
}

function table(headers, rows, emptyText) {
  if (!rows || !rows.length) return `<p class="hint">${esc(emptyText)}</p>`;
  return `
    <div class="table-wrap">
      <table class="data-table">
        <thead><tr>${headers.map(h => `<th>${esc(h)}</th>`).join('')}</tr></thead>
        <tbody>${rows.map(r => `<tr>${r.map(c => `<td>${c}</td>`).join('')}</tr>`).join('')}</tbody>
      </table>
    </div>`;
}

function renderOverview(ui, r, a) {
  const attribution = (a.attribution && a.attribution.length)
    ? a.attribution.map((c, i) => {
        const ct = causeText(state.lang, c.causeId);
        return `
          <div class="cause ${i === 0 ? 'cause-top' : ''}">
            <div class="cause-head">
              <span class="cause-rank">#${i + 1}</span>
              <span class="cause-title">${esc(ct.title)}</span>
              <span class="badge badge-${c.confidence}">${ui.confidence[c.confidence] || c.confidence}</span>
            </div>
            <p class="cause-desc">${esc(ct.desc)}</p>
            <ul class="evidence">
              ${c.evidence.map(ev => `<li>${esc(evidenceText(state.lang, ev))}</li>`).join('')}
            </ul>
          </div>`;
      }).join('')
    : (a.primaryRuleId === 'rule.cpu-freq-constrained' ? `<p class="hint">${ui.attributionEmpty}</p>` : '');

  const findings = (a.findings || []).map(f => {
    const ft = ruleText(state.lang, f.ruleId, f.params);
    return `
      <div class="finding">
        <div class="finding-head">
          <span class="sev sev-${f.severity}">${ui.findingSeverity[f.severity] || f.severity}</span>
          <span class="finding-title">${esc(ft.title)}</span>
        </div>
        <p class="finding-desc">${esc(ft.desc)}</p>
        <ul class="evidence">
          ${(f.evidence || []).map(ev => `<li>${esc(evidenceText(state.lang, ev))}</li>`).join('')}
        </ul>
      </div>`;
  }).join('');

  const samples = r.samples || [];
  const base = r.cpu.baseClockMHz;
  const freqChart = sparkline(samples.map(s => s.effectiveClockMHz), {
    max: Math.max(base, ...samples.map(s => s.effectiveClockMHz)),
    lines: [{ value: base, cls: 'spark-base' }, { value: base * 0.55, cls: 'spark-low' }],
  });
  const miniCharts = `
    <div class="mini-charts">
      <div><div class="mini-title">${ui.charts.load}</div>${sparkline(samples.map(s => s.cpuLoadPercent), { max: 100 })}</div>
      <div><div class="mini-title">${ui.charts.mem}</div>${sparkline(samples.map(s => s.memUsedPercent), { max: 100 })}</div>
      <div><div class="mini-title">${ui.charts.disk}</div>${sparkline(samples.map(s => s.diskActivePercent), { max: 100 })}</div>
    </div>`;

  return `
    ${attribution ? `<section class="panel"><div class="block-title">${ui.attribution}</div>${attribution}</section>` : ''}
    <section class="panel">
      <div class="block-title">${ui.samplingTitle}</div>
      ${freqChart}
      ${miniCharts}
    </section>
    <section class="panel">
      <div class="block-title">${ui.findings}</div>
      ${findings || `<p class="hint">${ruleText(state.lang, 'rule.no-major-issue', {}).desc}</p>`}
    </section>
    ${(r.collectorNotes && r.collectorNotes.length) ? `
    <section class="panel notes">
      <div class="block-title">${ui.collectorNotes}</div>
      <ul>${r.collectorNotes.map(n => `<li>${esc(n)}</li>`).join('')}</ul>
    </section>` : ''}`;
}

function actionKey(act, i) {
  return `${act.actionId}:${act.params?.serviceName || ''}:${i}`;
}

async function executeAction(key, act) {
  state.actionBusy = key;
  state.confirm = null;
  render();
  try {
    state.actionResults[key] = await RunAction(act.actionId, act.params || {});
  } catch (e) {
    state.actionResults[key] = { actionId: act.actionId, status: 'failed', detail: String(e) };
  }
  state.actionBusy = '';
  try {
    state.rollbackRecords = (await ListRollbackRecords()) || [];
  } catch { /* keep stale list */ }
  render();
}

async function executeRollback(rec) {
  state.actionBusy = `rollback:${rec.serviceName}:${rec.actionTime}`;
  render();
  try {
    const res = await RollbackService(rec.serviceName, rec.actionTime);
    state.actionResults[state.actionBusy] = res;
    state.rollbackRecords = (await ListRollbackRecords()) || [];
  } catch (e) {
    state.actionResults[state.actionBusy] = { status: 'failed', detail: String(e) };
  }
  state.actionBusy = '';
  render();
}

function renderActions(ui, r, a) {
  const acts = a.actions || [];
  const cards = acts.map((act, i) => {
    const key = actionKey(act, i);
    const at = actionText(state.lang, act.actionId, act.params);
    const res = state.actionResults[key];
    const busy = state.actionBusy === key;
    const irreversible = act.actionId === 'action.clean-temp';
    let status = '';
    if (busy) status = `<span class="badge badge-low">${ui.act.running}</span>`;
    else if (res) {
      status = res.status === 'success'
        ? `<span class="badge badge-ok">${ui.act.success}</span>${res.params?.freedMB !== undefined ? ` <span class="hint-inline">${esc(ui.act.freed(res.params))}</span>` : ''}`
        : `<span class="badge badge-err">${ui.act.failed}</span> <span class="hint-inline">${esc(res.detail || '')}</span>`;
    }
    const button = act.recommendOnly
      ? `<span class="badge badge-low">${ui.act.recommendOnly}</span>`
      : (res?.status === 'success' ? '' : `<button class="btn act-run" data-key="${esc(key)}" data-idx="${i}" ${busy || state.actionBusy ? 'disabled' : ''}>${ui.act.run}</button>`);
    return `
      <div class="action-card">
        <div class="action-head">
          <span class="badge risk-${act.risk}">${ui.act.risk[act.risk] || act.risk}</span>
          <span class="action-title">${esc(at.title)}</span>
          <span class="action-status">${status}</span>
          ${button}
        </div>
        <p class="action-desc">${esc(at.desc)}${irreversible ? ` <b>${ui.act.irreversible}</b>` : ''}</p>
      </div>`;
  }).join('');

  const records = (state.rollbackRecords || []).slice().reverse().map(rec => {
    const rbKey = `rollback:${rec.serviceName}:${rec.actionTime}`;
    const res = state.actionResults[rbKey];
    const busy = state.actionBusy === rbKey;
    const canRestore = !rec.rolledBack && rec.result === 'disabled';
    return `
      <div class="action-card">
        <div class="action-head">
          <span class="action-title">${esc(rec.displayName || rec.serviceName)}</span>
          ${rec.rolledBack
            ? `<span class="badge badge-ok">${ui.act.rolledBack} ${esc(rec.rollbackTime)}</span>`
            : (canRestore
              ? `<button class="btn act-rollback" data-svc="${esc(rec.serviceName)}" data-time="${esc(rec.actionTime)}" ${state.actionBusy ? 'disabled' : ''}>${busy ? ui.act.running : ui.act.rollback}</button>`
              : `<span class="badge badge-err">${esc(rec.result)}</span>`)}
        </div>
        <p class="action-desc">
          ${esc(rec.serviceName)} · ${ui.act.recPrev}: ${esc(rec.prevStartMode)}/${esc(rec.prevState)} · ${esc(rec.actionTime)}
          ${res && res.status === 'failed' ? `<br><span class="error">${esc(res.detail)}</span>` : ''}
        </p>
      </div>`;
  }).join('');

  const confirmModal = state.confirm ? (() => {
    const at = actionText(state.lang, state.confirm.action.actionId, state.confirm.action.params);
    const irreversible = state.confirm.action.actionId === 'action.clean-temp';
    return `
      <div class="modal-overlay">
        <div class="modal">
          <div class="modal-title">${ui.act.confirmTitle}</div>
          <div class="action-head" style="margin:8px 0 4px">
            <span class="badge risk-${state.confirm.action.risk}">${ui.act.risk[state.confirm.action.risk]}</span>
            <span class="action-title">${esc(at.title)}</span>
          </div>
          <p class="action-desc">${esc(at.desc)}${irreversible ? ` <b>${ui.act.irreversible}</b>` : ''}</p>
          <div class="modal-buttons">
            <button class="btn" id="confirm-cancel">${ui.act.cancel}</button>
            <button class="btn btn-primary" id="confirm-run">${ui.act.confirm}</button>
          </div>
        </div>
      </div>`;
  })() : '';

  return `
    <section class="panel">
      ${cards || `<p class="hint">${ui.act.noActions}</p>`}
    </section>
    <section class="panel">
      <div class="block-title">${ui.act.rollbackTitle}</div>
      <p class="hint" style="margin-bottom:8px">${ui.act.rollbackHint}</p>
      ${records || `<p class="hint">${ui.act.noRollback}</p>`}
    </section>
    ${confirmModal}`;
}

function renderTab(ui, r, a) {
  switch (state.tab) {
    case 'overview':
      return renderOverview(ui, r, a);
    case 'actions':
      return renderActions(ui, r, a);
    case 'processes':
      return `<section class="panel">${table(
        [ui.th.procName, ui.th.pid, ui.th.cpu, ui.th.mem],
        (r.processes || []).map(p => [esc(p.name), p.pid, `${p.cpuPercent.toFixed(1)}%`, `${p.workingSetMB.toFixed(0)} MB`]),
        ui.empty)}</section>`;
    case 'startup': {
      const items = table(
        [ui.th.name, ui.th.command, ui.th.location, ui.th.reviewWorthy],
        (r.startupItems || []).map(s => [
          esc(s.name), `<span class="mono">${esc(s.command)}</span>`, esc(s.location),
          s.reviewWorthy ? `<span class="badge badge-warn">${ui.yes}</span>` : '',
        ]),
        ui.empty);
      const tasks = table(
        [ui.th.name, ui.th.taskPath, ui.th.state],
        (r.scheduledTasks || []).map(tk => [esc(tk.name), `<span class="mono">${esc(tk.path)}</span>`, esc(tk.state)]),
        ui.empty);
      return `<section class="panel">${items}</section>
              <section class="panel"><div class="block-title">${ui.startupTasksTitle}</div>${tasks}</section>`;
    }
    case 'services':
      return `<section class="panel">${table(
        [ui.th.name, ui.th.displayName, ui.th.state, ui.th.startMode, ui.th.hint],
        (r.vendorServices || []).map(s => [
          esc(s.name), esc(s.displayName),
          `<span class="badge ${s.state === 'Running' ? 'badge-ok' : 'badge-low'}">${esc(s.state)}</span>`,
          esc(s.startMode), `<span class="mono">${esc(s.vendorHint)}</span>`,
        ]),
        ui.empty)}</section>`;
    case 'software':
      return `<section class="panel">${table(
        [ui.th.name, ui.th.version, ui.th.publisher, ui.th.category],
        (r.installedApps || []).map(x => [
          esc(x.name), esc(x.version), esc(x.publisher),
          x.category ? `<span class="badge badge-warn">${ui.utilityCategories[x.category] || esc(x.category)}</span>` : '',
        ]),
        ui.empty)}</section>`;
    case 'events':
      return `<section class="panel">${table(
        [ui.th.time, ui.th.level, ui.th.provider, ui.th.eventId, ui.th.message],
        (r.systemEvents || []).map(e => [
          esc(e.timeCreated),
          `<span class="sev ${e.level === 'Warning' ? 'sev-warning' : 'sev-critical'}">${esc(e.level)}</span>`,
          esc(e.provider), e.eventId, esc(e.message),
        ]),
        ui.empty)}</section>`;
    default:
      return '';
  }
}

function render() {
  const ui = t(state.lang).ui;
  const r = state.report;
  const a = r?.analysis;

  const header = `
    <header class="cmdbar">
      <div class="brand">
        <span class="brand-dot"></span>
        <span class="brand-name">${ui.appName}</span>
        <span class="brand-sub">${ui.subtitle}</span>
      </div>
      <div class="cmdbar-actions">
        <select id="lang-select" class="lang-select">
          <option value="zh" ${state.lang === 'zh' ? 'selected' : ''}>中文</option>
          <option value="en" ${state.lang === 'en' ? 'selected' : ''}>English</option>
        </select>
        <button id="open-log" class="btn">${ui.openLog}</button>
        <button id="scan-quick" class="btn" ${state.scanning ? 'disabled' : ''}>${ui.quickScan}</button>
        <button id="scan-deep" class="btn btn-primary" ${state.scanning ? 'disabled' : ''}>${ui.deepScan}</button>
      </div>
    </header>`;

  let body = '';
  if (state.scanning) {
    body = `
      <div class="panel center-panel">
        <div class="scan-title">${ui.scanning}…</div>
        <div class="progress"><div class="progress-fill" style="width:${(state.progress.done / state.progress.total) * 100}%"></div></div>
        <div class="progress-label">${state.progress.done} / ${state.progress.total}</div>
        <p class="hint">${esc(ui.scanHint)}</p>
      </div>`;
  } else if (!r) {
    body = `
      <div class="panel center-panel">
        <p class="hint">${ui.noReport}</p>
        ${state.error ? `<p class="error">${esc(state.error)}</p>` : ''}
      </div>`;
  } else {
    const primary = ruleText(state.lang, a.primaryRuleId, a.primaryParams);
    const adminBadge = r.isAdmin
      ? `<span class="badge badge-ok">${ui.admin}</span>`
      : `<span class="badge badge-warn">${ui.notAdmin}</span>`;
    const s = r.sampling || {};

    const tabBar = `
      <nav class="tabbar">
        ${TABS.map(tb => `<button class="tab ${state.tab === tb ? 'tab-active' : ''}" data-tab="${tb}">${ui.tabs[tb]}</button>`).join('')}
      </nav>`;

    body = `
      <div class="layout">
        <aside class="panel side">
          <div class="score-block ${scoreClass(a.severity)}">
            <div class="score-num">${a.score}</div>
            <div class="score-label">${ui.score} · ${ui.severity[a.severity] || a.severity}</div>
          </div>
          <div class="primary-block">
            <div class="block-title">${ui.primary}</div>
            <div class="primary-title">${esc(primary.title)}</div>
            <p class="primary-desc">${esc(primary.desc)}</p>
          </div>
          <div class="meta-block">
            ${adminBadge}
            <div class="meta-line">${ui.machine}: ${esc(r.computer.manufacturer)} ${esc(r.computer.model)}</div>
            <div class="meta-line">${esc(r.computer.osName)} ${esc(r.computer.osBuild)}</div>
            <div class="meta-line">${esc(r.cpu.name)}</div>
          </div>
          <div class="stats">
            <div class="stat"><span>${ui.avgClock}</span><b>${Math.round(s.avgEffectiveClockMHz || 0)} MHz</b></div>
            <div class="stat"><span>${ui.minClock}</span><b>${Math.round(s.minEffectiveClockMHz || 0)} MHz</b></div>
            <div class="stat"><span>${ui.avgLoad}</span><b>${(s.avgCpuLoadPercent || 0).toFixed(0)}%</b></div>
            <div class="stat"><span>${ui.memUsed}</span><b>${(s.avgMemUsedPercent || 0).toFixed(0)}%</b></div>
            <div class="stat"><span>${ui.diskActive}</span><b>${(s.avgDiskActivePercent || 0).toFixed(0)}%</b></div>
          </div>
        </aside>
        <main class="workspace">
          ${tabBar}
          ${renderTab(ui, r, a)}
        </main>
      </div>`;
  }

  app.innerHTML = header + `<div class="content">${body}</div>`;

  document.getElementById('lang-select')?.addEventListener('change', e => setLang(e.target.value));
  document.getElementById('open-log')?.addEventListener('click', () => OpenLogFolder());
  document.getElementById('scan-quick')?.addEventListener('click', () => startScan('quick'));
  document.getElementById('scan-deep')?.addEventListener('click', () => startScan('deep'));
  document.querySelectorAll('.tab').forEach(el => el.addEventListener('click', () => {
    state.tab = el.dataset.tab;
    render();
  }));
  document.querySelectorAll('.act-run').forEach(el => el.addEventListener('click', () => {
    const acts = state.report?.analysis?.actions || [];
    const act = acts[Number(el.dataset.idx)];
    if (act) {
      state.confirm = { key: el.dataset.key, action: act };
      render();
    }
  }));
  document.querySelectorAll('.act-rollback').forEach(el => el.addEventListener('click', () => {
    const rec = (state.rollbackRecords || []).find(x => x.serviceName === el.dataset.svc && x.actionTime === el.dataset.time);
    if (rec) executeRollback(rec);
  }));
  document.getElementById('confirm-cancel')?.addEventListener('click', () => {
    state.confirm = null;
    render();
  });
  document.getElementById('confirm-run')?.addEventListener('click', () => {
    if (state.confirm) executeAction(state.confirm.key, state.confirm.action);
  });
}

init();
