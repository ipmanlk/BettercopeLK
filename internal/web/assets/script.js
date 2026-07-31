// BettercopeLK frontend — vanilla JS, no build step.

(function () {
  'use strict';

  /* ---- DOM refs ---- */
  const $ = (sel) => document.querySelector(sel);
  const $$ = (sel) => document.querySelectorAll(sel);

  const searchInput = $('#searchInput');
  const searchBtn = $('#searchButton');
  const dropdownBtn = $('#dropdownButton');
  const dropdownContent = $('#dropdownContent');
  const sourceList = $('#sourceList');
  const sourceAll = $('#source-all');
  const sourceStatus = $('#sourceStatus');
  const loading = $('#loading');
  const resultsSection = $('#resultsSection');
  const resultsList = $('#resultsList');
  const resultsCount = $('#resultsCount');
  const noResults = $('#noResults');
  const selectAll = $('#selectAllCheckbox');
  const bulkBtn = $('#bulkDownloadBtn');
  const consoleEl = $('#console');
  const consoleContent = $('#consoleContent');
  const clearConsole = $('#clearConsole');

  /* ---- state ---- */
  let sources = [];
  let activeSearch = null;       // EventSource
  let results = [];              // { title, url, source, selected }
  let initialLog = null;         // source-state message restored on new search

  /* ---- helpers ---- */
  function log(msg, isError) {
    consoleEl.classList.add('show');
    const line = document.createElement('div');
    line.classList.add('console-line');
    if (isError) line.style.color = '#ff4444';
    line.textContent = msg;
    consoleContent.appendChild(line);
    consoleEl.scrollTop = consoleEl.scrollHeight;
  }

  function clearLog() {
    consoleContent.innerHTML = '';
    consoleEl.classList.remove('show');
  }

  function resetLog() {
    clearLog();
    if (initialLog) log(initialLog);
  }

  function disableSearch(placeholder) {
    searchInput.disabled = true;
    searchBtn.disabled = true;
    dropdownBtn.disabled = true;
    if (placeholder) searchInput.placeholder = placeholder;
  }

  function enableSearch() {
    searchInput.disabled = false;
    searchBtn.disabled = false;
    dropdownBtn.disabled = false;
    searchInput.placeholder = 'Search subtitles...';
  }

  function showLoading() { loading.hidden = false; }

  function hideLoading() { loading.hidden = true; }

  function getSelectedSources() {
    const checked = sourceList.querySelectorAll('input[type="checkbox"]:checked');
    return Array.from(checked).map((cb) => cb.value).filter(Boolean);
  }

  function updateDropdownLabel() {
    const label = dropdownBtn.querySelector('.dropdown-label');
    if (!label) return;
    const selected = getSelectedSources();
    if (selected.length === 0) {
      label.textContent = 'No sources';
    } else if (selected.length === sources.length) {
      label.textContent = 'All sources';
    } else {
      label.textContent = selected.length + ' source(s)';
    }
  }

  function updateBulkButton() {
    const anySelected = results.some((r) => r.selected);
    bulkBtn.disabled = !anySelected;
  }

  function updateSelectAll() {
    if (results.length === 0) return;
    const allChecked = results.every((r) => r.selected);
    selectAll.checked = allChecked;
  }

  /* ---- source loading ---- */
  async function loadSources() {
    disableSearch('Loading subtitle sources...');
    log('Loading available subtitle sources...');
    try {
      const resp = await fetch('/api/v1/sources');
      if (!resp.ok) throw new Error('HTTP ' + resp.status);
      const data = await resp.json();
      sources = data.sources || [];

      // Clear existing source items
      sourceList.innerHTML = '';
      sources.forEach((src) => {
        const label = document.createElement('label');
        label.className = 'dropdown-item';
        const cb = document.createElement('input');
        cb.type = 'checkbox';
        cb.value = src.id;
        cb.checked = true;
        cb.addEventListener('change', () => {
          updateDropdownLabel();
          sourceAll.checked = getSelectedSources().length === sources.length;
        });
        const text = document.createElement('span');
        text.className = 'source-name';
        text.textContent = src.name;
        label.appendChild(cb);
        label.appendChild(text);
        sourceList.appendChild(label);
      });

      updateDropdownLabel();
      initialLog = 'Loaded ' + sources.length + ' subtitle source(s)';
      log(initialLog);
      if (sources.length === 0) {
        sourceStatus.textContent = 'No sources available';
      }
    } catch (err) {
      initialLog = 'Failed to load sources: ' + err.message;
      log(initialLog, true);
      sourceStatus.textContent = 'Failed to load sources';
    } finally {
      enableSearch();
    }
  }

  /* ---- search ---- */
  function doSearch() {
    const query = searchInput.value.trim();
    if (!query) return;

    // Abort previous search.
    if (activeSearch) {
      activeSearch.close();
      activeSearch = null;
    }

    // Reset.
    resultsList.innerHTML = '';
    noResults.hidden = true;
    resultsSection.hidden = true;
    results = [];
    updateBulkButton();
    selectAll.checked = false;
    resetLog();
    showLoading();
    log('Searching for "' + query + '"...');

    const selectedSources = getSelectedSources();
    let url = '/api/v1/search/stream?query=' + encodeURIComponent(query);
    if (selectedSources.length > 0 && selectedSources.length < sources.length) {
      url += '&sources=' + selectedSources.join(',');
    }

    const es = new EventSource(url);
    activeSearch = es;

    es.addEventListener('result', (e) => {
      try {
        const data = JSON.parse(e.data);
        results.push({ title: data.title, url: data.url, source: data.source, selected: false });
        resultsSection.hidden = false;
        noResults.hidden = true;
        appendResultRow(data.title, data.url, data.source, results.length - 1);
      } catch (err) {
        log('Parse error: ' + err.message, true);
      }
    });

    es.addEventListener('source-complete', (e) => {
      try {
        const data = JSON.parse(e.data);
        log('Source "' + (data.source || '?') + '" complete (' + data.count + ' results)');
      } catch (err) { /* ignore */ }
    });

    es.addEventListener('error', (e) => {
      // Ignore events from a search that was already replaced or completed.
      if (activeSearch !== es) return;
      hideLoading();
      activeSearch = null;
      es.close();
      log('Search failed: the server is not reachable or returned an error.', true);
    });

    es.addEventListener('done', () => {
      if (activeSearch !== es) return;
      es.close();
      activeSearch = null;
      hideLoading();
      const count = results.length;
      resultsCount.textContent = count + ' result(s)';
      if (count === 0) {
        noResults.hidden = false;
      }
      resultsSection.hidden = false;
      log('Search complete — ' + count + ' result(s)');
    });
  }

  function appendResultRow(title, url, source, index) {
    const row = document.createElement('div');
    row.className = 'result-row';
    row.dataset.index = index;

    const cb = document.createElement('input');
    cb.type = 'checkbox';
    cb.className = 'result-checkbox';
    cb.addEventListener('change', () => {
      results[index].selected = cb.checked;
      updateBulkButton();
      updateSelectAll();
    });

    const info = document.createElement('div');
    info.className = 'result-info';

    const titleEl = document.createElement('span');
    titleEl.className = 'result-title';
    titleEl.textContent = title;

    const sourceEl = document.createElement('span');
    sourceEl.className = 'result-source';
    sourceEl.textContent = source;

    info.appendChild(titleEl);
    info.appendChild(sourceEl);

    const actions = document.createElement('div');
    actions.className = 'result-actions';

    const dlLink = document.createElement('a');
    dlLink.className = 'btn btn-small';
    dlLink.href = '/api/v1/download?url=' + encodeURIComponent(url) + '&source=' + encodeURIComponent(source);
    dlLink.textContent = 'Download';
    dlLink.target = '_blank';
    dlLink.rel = 'noopener';

    actions.appendChild(dlLink);

    row.appendChild(cb);
    row.appendChild(info);
    row.appendChild(actions);
    resultsList.appendChild(row);
  }

  /* ---- bulk download ---- */
  async function doBulkDownload() {
    const selected = results.filter((r) => r.selected);
    if (selected.length === 0) return;

    log('Downloading ' + selected.length + ' subtitle(s)...');

    const items = selected.map((r) => ({ url: r.url, source: r.source }));

    try {
      const resp = await fetch('/api/v1/download/bulk', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ items }),
      });

      if (!resp.ok) {
        let msg = 'HTTP ' + resp.status;
        try {
          const errData = await resp.json();
          msg += ': ' + (errData.error || '');
        } catch (_) { /* ignore */ }
        throw new Error(msg);
      }

      // Trigger download.
      const blob = await resp.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'subtitles.zip';
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);

      log('Downloaded ' + selected.length + ' subtitle(s) as subtitles.zip');
    } catch (err) {
      log('Bulk download failed: ' + err.message, true);
    }
  }

  /* ---- event wiring ---- */
  searchBtn.addEventListener('click', doSearch);
  searchInput.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') doSearch();
  });

  dropdownBtn.addEventListener('click', () => {
    dropdownContent.classList.toggle('show');
  });
  document.addEventListener('click', (e) => {
    if (!dropdownBtn.contains(e.target) && !dropdownContent.contains(e.target)) {
      dropdownContent.classList.remove('show');
    }
  });

  sourceAll.addEventListener('change', () => {
    const checked = sourceAll.checked;
    sourceList.querySelectorAll('input[type="checkbox"]').forEach((cb) => {
      cb.checked = checked;
    });
    updateDropdownLabel();
  });

  selectAll.addEventListener('change', () => {
    const checked = selectAll.checked;
    results.forEach((r, i) => { r.selected = checked; });
    resultsList.querySelectorAll('.result-checkbox').forEach((cb) => {
      cb.checked = checked;
    });
    updateBulkButton();
  });

  bulkBtn.addEventListener('click', doBulkDownload);

  clearConsole.addEventListener('click', clearLog);

  /* ---- init ---- */
  loadSources();
})();
