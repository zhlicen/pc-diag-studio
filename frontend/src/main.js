import './style.css';
import { t, ruleText, causeText, evidenceText } from './i18n';
import { GetStatus, RunScan, OpenLogFolder } from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';

const LANG_KEY = 'diagnostic-studio-lang';

const state = {
  lang: localStorage.getItem(LANG_KEY) || 'zh',
  status: null,
  report: null,
  scanning: false,
  progress: { done: 0, total: 1 },
  error: '',
};

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

function sparkline(samples, baseMHz) {
  if (!samples || samples.length < 2) return '';
  const w = 560, h = 120, pad = 6;
  const max = Math.max(baseMHz, ...samples.map(s => s.effectiveClockMHz)) * 1.05;
  const pts = samples.map((s, i) => {
    const x = pad + (i / (samples.length - 1)) * (w - pad * 2);
    const y = h - pad - (s.effectiveClockMHz / max) * (h - pad * 2);
    return `${x.toFixed(1)},${y.toFixed(1)}`;
  }).join(' ');
  const baseY = h - pad - (baseMHz / max) * (h - pad * 2);
  const lowY = h - pad - (baseMHz * 0.55 / max) * (h - pad * 2);
  return `
    <svg viewBox="0 0 ${w} ${h}" class="spark" preserveAspectRatio="none">
      <line x1="${pad}" y1="${baseY}" x2="${w - pad}" y2="${baseY}" class="spark-base"/>
      <line x1="${pad}" y1="${lowY}" x2="${w - pad}" y2="${lowY}" class="spark-low"/>
      <polyline points="${pts}" class="spark-line"/>
    </svg>`;
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

    const s = r.sampling || {};
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
          ${attribution ? `<section class="panel"><div class="block-title">${ui.attribution}</div>${attribution}</section>` : ''}
          <section class="panel">
            <div class="block-title">${ui.samplingTitle}</div>
            ${sparkline(r.samples, r.cpu.baseClockMHz)}
          </section>
          <section class="panel">
            <div class="block-title">${ui.findings}</div>
            ${findings || `<p class="hint">${ruleText(state.lang, 'rule.no-major-issue', {}).desc}</p>`}
          </section>
          ${(r.collectorNotes && r.collectorNotes.length) ? `
          <section class="panel notes">
            <div class="block-title">${ui.collectorNotes}</div>
            <ul>${r.collectorNotes.map(n => `<li>${esc(n)}</li>`).join('')}</ul>
          </section>` : ''}
        </main>
      </div>`;
  }

  app.innerHTML = header + `<div class="content">${body}</div>`;

  document.getElementById('lang-select')?.addEventListener('change', e => setLang(e.target.value));
  document.getElementById('open-log')?.addEventListener('click', () => OpenLogFolder());
  document.getElementById('scan-quick')?.addEventListener('click', () => startScan('quick'));
  document.getElementById('scan-deep')?.addEventListener('click', () => startScan('deep'));
}

init();
