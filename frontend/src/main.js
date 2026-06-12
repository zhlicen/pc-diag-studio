import './style.css';
import { t, ruleText, causeText, evidenceText, actionText } from './i18n';
import {
  GetStatus, RunScan, OpenLogFolder, RunAction,
  ListRollbackRecords, RollbackService, RollbackStartup, MarkLagNow,
  GetAIConfig, SaveAIConfig, GetAISendPreview, GenerateAIExplanation,
} from '../wailsjs/go/main/App';
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
  actionResults: {},
  actionBusy: '',
  confirm: null,        // {key, action} awaiting confirmation
  symptomPick: null,    // scan mode awaiting symptom selection
  markerCount: 0,
  lastMarkerAt: -1,
  highlight: null,      // {tab, ref} from evidence click-through
  aiConfig: null,
  aiDraft: null,
  aiSettingsOpen: false,
  aiBusy: '',
  aiResult: null,
  aiPreview: '',
};

const TABS = ['overview', 'actions', 'processes', 'startup', 'services', 'software', 'events', 'ai'];
const SYMPTOMS = ['symptom.boot-slow', 'symptom.always-slow', 'symptom.intermittent', 'symptom.fan-noise', 'symptom.battery-only', 'symptom.app-specific'];

const app = document.getElementById('app');

function esc(s) {
  return String(s ?? '').replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
}

function renderMarkdown(raw) {
  const lines = String(raw ?? '').replace(/\r\n/g, '\n').split('\n');
  let html = '';
  let paragraph = [];
  let list = '';
  let inCode = false;
  let codeLines = [];

  const renderInline = text => {
    const codes = [];
    let out = esc(text).replace(/`([^`]+)`/g, (_, code) => {
      const key = codes.length;
      codes.push(`<code>${code}</code>`);
      return `\u0000${key}\u0000`;
    });
    out = out
      .replace(/\[([^\]]+)\]\((https?:\/\/[^\s)]+)\)/g, '<a href="$2" target="_blank" rel="noreferrer">$1</a>')
      .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
      .replace(/__([^_]+)__/g, '<strong>$1</strong>')
      .replace(/\*([^*\n]+)\*/g, '<em>$1</em>')
      .replace(/_([^_\n]+)_/g, '<em>$1</em>');
    return out.replace(/\u0000(\d+)\u0000/g, (_, key) => codes[Number(key)] || '');
  };
  const flushParagraph = () => {
    if (!paragraph.length) return;
    html += `<p>${renderInline(paragraph.join(' '))}</p>`;
    paragraph = [];
  };
  const closeList = () => {
    if (!list) return;
    html += `</${list}>`;
    list = '';
  };
  const openList = tag => {
    flushParagraph();
    if (list === tag) return;
    closeList();
    list = tag;
    html += `<${tag}>`;
  };

  for (const line of lines) {
    if (/^\s*```/.test(line)) {
      if (inCode) {
        html += `<pre><code>${esc(codeLines.join('\n'))}</code></pre>`;
        codeLines = [];
        inCode = false;
      } else {
        flushParagraph();
        closeList();
        inCode = true;
      }
      continue;
    }

    if (inCode) {
      codeLines.push(line);
      continue;
    }

    if (!line.trim()) {
      flushParagraph();
      closeList();
      continue;
    }

    const heading = line.match(/^(#{1,4})\s+(.+)$/);
    if (heading) {
      flushParagraph();
      closeList();
      const level = Math.min(5, heading[1].length + 2);
      html += `<h${level}>${renderInline(heading[2].trim())}</h${level}>`;
      continue;
    }

    const bullet = line.match(/^\s*[-*+]\s+(.+)$/);
    if (bullet) {
      openList('ul');
      html += `<li>${renderInline(bullet[1].trim())}</li>`;
      continue;
    }

    const ordered = line.match(/^\s*\d+[.)]\s+(.+)$/);
    if (ordered) {
      openList('ol');
      html += `<li>${renderInline(ordered[1].trim())}</li>`;
      continue;
    }

    const quote = line.match(/^\s*>\s?(.+)$/);
    if (quote) {
      flushParagraph();
      closeList();
      html += `<blockquote>${renderInline(quote[1].trim())}</blockquote>`;
      continue;
    }

    closeList();
    paragraph.push(line.trim());
  }

  if (inCode) {
    html += `<pre><code>${esc(codeLines.join('\n'))}</code></pre>`;
  }
  flushParagraph();
  closeList();
  return html;
}

async function init() {
  try {
    state.status = await GetStatus();
  } catch (e) {
    state.error = String(e);
  }
  try {
    state.rollbackRecords = (await ListRollbackRecords()) || [];
  } catch { /* optional context */ }
  await loadAIConfig();
  EventsOn('scan:progress', p => {
    state.progress = p;
    const bar = document.querySelector('.progress-fill');
    const label = document.querySelector('.progress-label');
    if (bar) bar.style.width = `${(p.done / p.total) * 100}%`;
    if (label) label.textContent = `${p.done} / ${p.total}`;
  });
  render();
}

async function beginScan(mode, symptom) {
  if (state.scanning) return;
  state.scanning = true;
  state.symptomPick = null;
  state.error = '';
  state.markerCount = 0;
  state.lastMarkerAt = -1;
  state.progress = { done: 0, total: mode === 'deep' ? 36 : 15 };
  render();
  try {
    state.report = await RunScan(mode, symptom || '');
    state.tab = 'overview';
    state.highlight = null;
    state.aiResult = null;
    state.aiPreview = '';
  } catch (e) {
    state.error = String(e);
  }
  state.scanning = false;
  render();
}

async function markLag() {
  try {
    const offset = await MarkLagNow();
    if (offset >= 0) {
      state.markerCount += 1;
      state.lastMarkerAt = offset;
      const el = document.querySelector('.lag-feedback');
      if (el) el.textContent = t(state.lang).ui.lag.marked({ count: state.markerCount, offset });
    }
  } catch { /* scan may have just finished */ }
}

async function loadAIConfig() {
  try {
    state.aiConfig = await GetAIConfig();
    state.aiDraft = { ...state.aiConfig, apiKey: '', clearApiKey: false };
  } catch (e) {
    state.aiConfig = { enabled: false, hasApiKey: false, baseUrl: '', model: '' };
    state.aiDraft = { ...state.aiConfig, apiKey: '', clearApiKey: false };
    state.error = String(e);
  }
}

function openAISettings() {
  state.aiDraft = { ...(state.aiConfig || {}), apiKey: '', clearApiKey: false };
  state.aiSettingsOpen = true;
  render();
}

async function saveAISettings() {
  const form = document.getElementById('ai-settings-form');
  if (!form) return;
  const data = new FormData(form);
  const draft = {
    baseUrl: String(data.get('baseUrl') || '').trim(),
    model: String(data.get('model') || '').trim(),
    enabled: data.get('enabled') === 'on',
    apiKey: String(data.get('apiKey') || '').trim(),
    clearApiKey: data.get('clearApiKey') === 'on',
  };
  state.aiBusy = 'settings';
  render();
  try {
    await SaveAIConfig(draft);
    await loadAIConfig();
    state.aiSettingsOpen = false;
  } catch (e) {
    state.error = String(e);
  }
  state.aiBusy = '';
  render();
}

async function generateAI() {
  if (!state.report || state.aiBusy) return;
  state.aiBusy = 'generate';
  state.aiResult = null;
  render();
  try {
    state.aiResult = await GenerateAIExplanation(state.report, state.lang);
  } catch (e) {
    state.aiResult = { status: 'failed', detail: String(e), content: '' };
  }
  state.aiBusy = '';
  render();
}

async function loadAIPreview() {
  if (!state.report || state.aiBusy) return;
  state.aiBusy = 'preview';
  render();
  try {
    state.aiPreview = await GetAISendPreview(state.report);
  } catch (e) {
    state.aiPreview = String(e);
  }
  state.aiBusy = '';
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

function kbText(kbId) {
  return t(state.lang).ui.kb[kbId] || null;
}

// Evidence list with click-through links (params._tab/_ref set by analyzer).
function evidenceList(evs) {
  if (!evs || !evs.length) return '';
  return `<ul class="evidence">${evs.map(ev => {
    const text = esc(evidenceText(state.lang, ev));
    const tab = ev.params?._tab;
    const ref = ev.params?._ref;
    return tab
      ? `<li class="ev-link" data-tab="${esc(tab)}" data-ref="${esc(ref || '')}">${text}</li>`
      : `<li>${text}</li>`;
  }).join('')}</ul>`;
}

function sparkline(values, { max, lines = [], markers = [], lastOffset = 0 } = {}) {
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
  const marks = (lastOffset > 0 ? markers : []).map(m => {
    const x = pad + (Math.min(m, lastOffset) / lastOffset) * (w - pad * 2);
    return `<line x1="${x.toFixed(1)}" y1="${pad}" x2="${x.toFixed(1)}" y2="${h - pad}" class="spark-marker"/>`;
  }).join('');
  return `
    <svg viewBox="0 0 ${w} ${h}" class="spark" preserveAspectRatio="none">
      ${guides}${marks}
      <polyline points="${pts}" class="spark-line"/>
    </svg>`;
}

// rows: array of cell arrays; refs: optional per-row ref id for highlight.
function table(headers, rows, emptyText, refs) {
  if (!rows || !rows.length) return `<p class="hint">${esc(emptyText)}</p>`;
  const hl = state.highlight;
  const colClasses = headers.map(h => {
    if (h === '状态' || h === 'State') return 'col-state';
    return '';
  });
  return `
    <div class="table-wrap">
      <table class="data-table">
        <thead><tr>${headers.map((h, i) => `<th${colClasses[i] ? ` class="${colClasses[i]}"` : ''}>${esc(h)}</th>`).join('')}</tr></thead>
        <tbody>${rows.map((r, i) => {
          const ref = refs ? refs[i] : '';
          const cls = hl && ref && hl.ref === ref ? ' class="row-flag"' : '';
          return `<tr${cls}>${r.map((c, j) => `<td${colClasses[j] ? ` class="${colClasses[j]}"` : ''}>${c}</td>`).join('')}</tr>`;
        }).join('')}</tbody>
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
            ${evidenceList(c.evidence)}
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
        ${evidenceList(f.evidence)}
      </div>`;
  }).join('');

  const samples = r.samples || [];
  const base = r.cpu.baseClockMHz;
  const lastOffset = samples.length ? samples[samples.length - 1].offsetSec : 0;
  const freqChart = sparkline(samples.map(s => s.effectiveClockMHz), {
    max: Math.max(base, ...samples.map(s => s.effectiveClockMHz)),
    lines: [{ value: base, cls: 'spark-base' }, { value: base * 0.55, cls: 'spark-low' }],
    markers: r.lagMarkers || [],
    lastOffset,
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
  return `${act.actionId}:${act.params?.serviceName || act.params?.name || ''}:${i}`;
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
  const rbKey = `rollback:${rec.serviceName}:${rec.actionTime}`;
  state.actionBusy = rbKey;
  render();
  try {
    const res = rec.kind === 'startup'
      ? await RollbackStartup(rec.serviceName, rec.actionTime)
      : await RollbackService(rec.serviceName, rec.actionTime);
    state.actionResults[rbKey] = res;
    state.rollbackRecords = (await ListRollbackRecords()) || [];
  } catch (e) {
    state.actionResults[rbKey] = { status: 'failed', detail: String(e) };
  }
  state.actionBusy = '';
  render();
}

function kbBlock(kbId) {
  const kb = kbText(kbId);
  if (!kb) return '';
  return `
    <div class="kb-block">
      <p>${esc(kb.what)}</p>
      <p>${esc(kb.ifDisabled)}</p>
      ${kb.caution ? `<p class="kb-caution">${esc(kb.caution)}</p>` : ''}
    </div>`;
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
        ${act.params?.kbId ? kbBlock(act.params.kbId) : ''}
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
          ${rec.kind === 'startup' ? `<span class="badge badge-low">${ui.tabs.startup}</span>` : ''}
          ${rec.rolledBack
            ? `<span class="badge badge-ok">${ui.act.rolledBack} ${esc(rec.rollbackTime)}</span>`
            : (canRestore
              ? `<button class="btn act-rollback" data-svc="${esc(rec.serviceName)}" data-time="${esc(rec.actionTime)}" ${state.actionBusy ? 'disabled' : ''}>${busy ? ui.act.running : ui.act.rollback}</button>`
              : `<span class="badge badge-err">${esc(rec.result)}</span>`)}
        </div>
        <p class="action-desc">
          ${esc(rec.serviceName)}${rec.kind !== 'startup' ? ` · ${ui.act.recPrev}: ${esc(rec.prevStartMode)}/${esc(rec.prevState)}` : ''} · ${esc(rec.actionTime)}
          ${res && res.status === 'failed' ? `<br><span class="error">${esc(res.detail)}</span>` : ''}
        </p>
      </div>`;
  }).join('');

  return `
    <section class="panel">
      ${cards || `<p class="hint">${ui.act.noActions}</p>`}
    </section>
    <section class="panel">
      <div class="block-title">${ui.act.rollbackTitle}</div>
      <p class="hint" style="margin-bottom:8px">${ui.act.rollbackHint}</p>
      ${records || `<p class="hint">${ui.act.noRollback}</p>`}
    </section>`;
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
        ui.empty,
        (r.processes || []).map(p => p.name))}</section>`;
    case 'startup': {
      const items = r.startupItems || [];
      const itemsTable = table(
        [ui.th.name, ui.th.command, ui.th.state, ''],
        items.map(s => {
          const stateBadge = s.disabled
            ? `<span class="badge badge-low">${ui.startupState.disabled}</span>`
            : `<span class="badge badge-ok">${ui.startupState.enabled}</span>`;
          const btn = (!s.disabled && s.canToggle)
            ? `<button class="btn btn-sm startup-disable" data-name="${esc(s.name)}" data-location="${esc(s.location)}" data-command="${esc(s.command)}" ${state.actionBusy ? 'disabled' : ''}>${ui.disableBtn}</button>`
            : '';
          return [esc(s.name), `<span class="mono">${esc(s.command)}</span>`, stateBadge, btn];
        }),
        ui.empty,
        items.map(s => s.name));
      const tasks = table(
        [ui.th.name, ui.th.taskPath, ui.th.state],
        (r.scheduledTasks || []).map(tk => [esc(tk.name), `<span class="mono">${esc(tk.path)}</span>`, esc(tk.state)]),
        ui.empty);
      return `<section class="panel">${itemsTable}</section>
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
        ui.empty,
        (r.vendorServices || []).map(s => s.name))}</section>`;
    case 'software':
      return `<section class="panel">${table(
        [ui.th.name, ui.th.version, ui.th.publisher, ui.th.category],
        (r.installedApps || []).map(x => [
          esc(x.name), esc(x.version), esc(x.publisher),
          x.category ? `<span class="badge badge-warn">${ui.utilityCategories[x.category] || esc(x.category)}</span>` : '',
        ]),
        ui.empty,
        (r.installedApps || []).map(x => x.name))}</section>`;
    case 'events':
      return `<section class="panel">${table(
        [ui.th.time, ui.th.level, ui.th.provider, ui.th.eventId, ui.th.message],
        (r.systemEvents || []).map(e => [
          esc(e.timeCreated),
          `<span class="sev ${e.level === 'Warning' ? 'sev-warning' : 'sev-critical'}">${esc(e.level)}</span>`,
          esc(e.provider), e.eventId, esc(e.message),
        ]),
        ui.empty)}</section>`;
    case 'ai':
      return renderAI(ui, r);
    default:
      return '';
  }
}

function renderAI(ui, r) {
  const cfg = state.aiConfig || {};
  const ai = ui.ai;
  const status = cfg.enabled
    ? (cfg.hasApiKey ? ai.statusReady : ai.statusMissingKey)
    : ai.statusDisabled;
  const result = state.aiResult;
  return `
    <section class="panel">
      <div class="ai-head">
        <div>
          <div class="block-title">${ai.title}</div>
          <p class="hint">${ai.subtitle}</p>
        </div>
        <button class="btn" id="ai-settings-inline">${ai.settings}</button>
      </div>
      <div class="ai-config-line">
        <span class="badge ${cfg.enabled && cfg.hasApiKey ? 'badge-ok' : 'badge-low'}">${esc(status)}</span>
        <span class="mono">${esc(cfg.baseUrl || '')}</span>
        <span class="mono">${esc(cfg.model || '')}</span>
      </div>
      <div class="ai-actions">
        <button class="btn btn-primary" id="ai-generate" ${state.aiBusy || !r ? 'disabled' : ''}>${state.aiBusy === 'generate' ? ai.generating : ai.generate}</button>
        <button class="btn" id="ai-preview" ${state.aiBusy || !r ? 'disabled' : ''}>${state.aiBusy === 'preview' ? ai.loadingPreview : ai.preview}</button>
      </div>
      ${result ? renderAIResult(ai, result) : `<p class="hint">${ai.empty}</p>`}
    </section>
    ${state.aiPreview ? `<section class="panel"><div class="block-title">${ai.previewTitle}</div><pre class="ai-preview">${esc(state.aiPreview)}</pre></section>` : ''}`;
}

function renderAIResult(ai, result) {
  if (result.status === 'success') {
    return `
      <div class="ai-result">
        <div class="ai-result-meta">${esc(result.model || '')}${result.sentAt ? ` · ${esc(result.sentAt)}` : ''}</div>
        <div class="ai-content">${renderMarkdown(result.content)}</div>
      </div>`;
  }
  const message = ai.statusText[result.status] || result.detail || result.status;
  return `<div class="ai-result ai-result-muted"><b>${esc(message)}</b>${result.detail ? `<p>${esc(result.detail)}</p>` : ''}</div>`;
}

function renderModals(ui) {
  if (state.aiSettingsOpen) {
    const cfg = state.aiDraft || {};
    return `
      <div class="modal-overlay">
        <div class="modal">
          <div class="modal-title">${ui.ai.settings}</div>
          <form id="ai-settings-form" class="form-grid">
            <label class="check-row"><input type="checkbox" name="enabled" ${cfg.enabled ? 'checked' : ''}> ${ui.ai.enabled}</label>
            <label>${ui.ai.baseUrl}<input name="baseUrl" value="${esc(cfg.baseUrl || '')}" placeholder="https://api.openai.com/v1"></label>
            <label>${ui.ai.model}<input name="model" value="${esc(cfg.model || '')}" placeholder="gpt-5.5"></label>
            <label>${ui.ai.apiKey}<input type="password" name="apiKey" placeholder="${cfg.hasApiKey ? ui.ai.keepKey : ui.ai.enterKey}"></label>
            <label class="check-row"><input type="checkbox" name="clearApiKey"> ${ui.ai.clearKey}</label>
          </form>
          <p class="hint">${ui.ai.safety}</p>
          <div class="modal-buttons">
            <button class="btn" id="ai-settings-cancel">${ui.act.cancel}</button>
            <button class="btn btn-primary" id="ai-settings-save" ${state.aiBusy === 'settings' ? 'disabled' : ''}>${state.aiBusy === 'settings' ? ui.ai.saving : ui.ai.save}</button>
          </div>
        </div>
      </div>`;
  }
  if (state.symptomPick) {
    return `
      <div class="modal-overlay">
        <div class="modal">
          <div class="modal-title">${ui.symptom.title}</div>
          <p class="hint" style="margin:6px 0 10px">${ui.symptom.hint}</p>
          <div class="symptom-grid">
            ${SYMPTOMS.map(s => `<button class="btn symptom-btn" data-symptom="${s}">${ui.symptom[s]}</button>`).join('')}
          </div>
          <div class="modal-buttons">
            <button class="btn" id="symptom-cancel">${ui.act.cancel}</button>
            <button class="btn btn-primary" id="symptom-skip">${ui.symptom.skip}</button>
          </div>
        </div>
      </div>`;
  }
  if (state.confirm) {
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
          ${state.confirm.action.params?.kbId ? kbBlock(state.confirm.action.params.kbId) : ''}
          <div class="modal-buttons">
            <button class="btn" id="confirm-cancel">${ui.act.cancel}</button>
            <button class="btn btn-primary" id="confirm-run">${ui.act.confirm}</button>
          </div>
        </div>
      </div>`;
  }
  return '';
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
        <button id="ai-settings" class="btn">${ui.ai.settings}</button>
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
        <button id="lag-button" class="btn lag-button">${ui.lag.button}</button>
        <div class="lag-feedback hint">${state.markerCount > 0 ? ui.lag.marked({ count: state.markerCount, offset: state.lastMarkerAt }) : ''}</div>
        <p class="hint">${esc(ui.lag.hint)}</p>
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
    const symptomChip = r.symptom
      ? `<div class="symptom-chip">${ui.symptom[r.symptom] || r.symptom}</div>`
      : '';

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
            ${symptomChip}
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

  app.innerHTML = header + `<div class="content">${body}</div>` + renderModals(ui);
  bindEvents();
}

function bindEvents() {
  document.getElementById('lang-select')?.addEventListener('change', e => setLang(e.target.value));
  document.getElementById('ai-settings')?.addEventListener('click', openAISettings);
  document.getElementById('ai-settings-inline')?.addEventListener('click', openAISettings);
  document.getElementById('ai-settings-cancel')?.addEventListener('click', () => { state.aiSettingsOpen = false; render(); });
  document.getElementById('ai-settings-save')?.addEventListener('click', saveAISettings);
  document.getElementById('ai-generate')?.addEventListener('click', generateAI);
  document.getElementById('ai-preview')?.addEventListener('click', loadAIPreview);
  document.getElementById('open-log')?.addEventListener('click', () => OpenLogFolder());
  document.getElementById('scan-quick')?.addEventListener('click', () => { state.symptomPick = 'quick'; render(); });
  document.getElementById('scan-deep')?.addEventListener('click', () => { state.symptomPick = 'deep'; render(); });
  document.getElementById('lag-button')?.addEventListener('click', markLag);

  document.querySelectorAll('.symptom-btn').forEach(el => el.addEventListener('click', () => {
    beginScan(state.symptomPick, el.dataset.symptom);
  }));
  document.getElementById('symptom-skip')?.addEventListener('click', () => beginScan(state.symptomPick, ''));
  document.getElementById('symptom-cancel')?.addEventListener('click', () => { state.symptomPick = null; render(); });

  document.querySelectorAll('.tab').forEach(el => el.addEventListener('click', () => {
    state.tab = el.dataset.tab;
    state.highlight = null;
    render();
  }));
  document.querySelectorAll('.ev-link').forEach(el => el.addEventListener('click', () => {
    state.tab = el.dataset.tab;
    state.highlight = { tab: el.dataset.tab, ref: el.dataset.ref };
    render();
    document.querySelector('.row-flag')?.scrollIntoView({ block: 'center' });
  }));

  document.querySelectorAll('.act-run').forEach(el => el.addEventListener('click', () => {
    const acts = state.report?.analysis?.actions || [];
    const act = acts[Number(el.dataset.idx)];
    if (act) {
      state.confirm = { key: el.dataset.key, action: act };
      render();
    }
  }));
  document.querySelectorAll('.startup-disable').forEach(el => el.addEventListener('click', () => {
    const act = {
      actionId: 'action.disable-startup',
      risk: 'review',
      params: { name: el.dataset.name, location: el.dataset.location, command: el.dataset.command },
    };
    state.confirm = { key: `startup:${el.dataset.name}`, action: act };
    render();
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
