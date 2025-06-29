// DOM Elements
const elements = {
  searchInput: document.getElementById("searchInput"),
  searchButton: document.getElementById("searchButton"),
  dropdownButton: document.getElementById("dropdownButton"),
  dropdownContent: document.getElementById("dropdownContent"),
  loading: document.getElementById("loading"),
  resultsSection: document.getElementById("resultsSection"),
  resultsList: document.getElementById("resultsList"),
  resultsCount: document.getElementById("resultsCount"),
  noResults: document.getElementById("noResults"),
  consoleDiv: document.getElementById("console"),
  selectAllCheckbox: document.getElementById("source-all"),
  sourceStatus: document.getElementById("sourceStatus")
};

// API Endpoints
const API = {
  BASE_URL: "/api/v1",
  get SEARCH() { return `${this.BASE_URL}/search`; },
  get SEARCH_STREAM() { return `${this.BASE_URL}/search/stream`; },
  get DOWNLOAD() { return `${this.BASE_URL}/download`; },
  get SOURCES() { return `${this.BASE_URL}/sources`; }
};

// Application state
const state = {
  searchEventSource: null,
  activeSources: new Set()
};

/**
 * Log a message to the console panel
 * @param {string} message - Message to log
 * @param {boolean} isError - Whether the message is an error
 */
function logToConsole(message, isError = false) {
  elements.consoleDiv.classList.add("show");
  const line = document.createElement("div");
  line.classList.add("console-line");
  
  if (isError) {
    line.style.color = "#ff4444";
  }
  
  line.textContent = message;
  elements.consoleDiv.appendChild(line);
  elements.consoleDiv.scrollTop = elements.consoleDiv.scrollHeight;
}

/**
 * Load available subtitle sources from the server
 * @returns {Promise<void>}
 */
async function loadSources() {
  elements.searchInput.disabled = true;
  elements.searchButton.disabled = true;
  elements.dropdownButton.disabled = true;
  elements.searchInput.placeholder = "Loading subtitle sources...";

  try {
    logToConsole("Loading available subtitle sources...");
    const response = await fetch(API.SOURCES);

    if (!response.ok) {
      throw new Error(
        `Failed to load sources: ${response.status} ${response.statusText}`
      );
    }

    const data = await response.json();
    clearSourceItems();
    createSourceItems(data.sources);
    updateDropdownButton();

    logToConsole(`Loaded ${data.sources.length} subtitle sources`);
    
    elements.searchInput.disabled = false;
    elements.dropdownButton.disabled = false;
    elements.searchButton.disabled = false;
    elements.searchInput.placeholder = "Enter movie or TV show name...";
  } catch (error) {
    logToConsole(`Error: ${error.message}`, true);
    elements.searchInput.placeholder = "Error loading sources. Please refresh the page...";
  } finally {
    document.getElementById("sourcesLoading").classList.remove("show");
  }
}

/**
 * Clear existing source items from the dropdown
 */
function clearSourceItems() {
  const sourceItems = elements.dropdownContent.querySelectorAll(
    ".dropdown-item:not(:first-child)"
  );
  sourceItems.forEach((item) => item.remove());
}

/**
 * Create source items in the dropdown
 * @param {string[]} sources - List of source names
 */
function createSourceItems(sources) {
  sources.forEach((source) => {
    const item = document.createElement("div");
    item.className = "dropdown-item";

    const checkbox = document.createElement("input");
    checkbox.type = "checkbox";
    checkbox.id = `source-${source}`;
    checkbox.checked = true;
    checkbox.dataset.source = source;

    const label = document.createElement("label");
    label.htmlFor = `source-${source}`;
    label.textContent = source;

    item.appendChild(checkbox);
    item.appendChild(label);
    elements.dropdownContent.appendChild(item);

    checkbox.addEventListener("change", updateDropdownButton);
  });
}

/**
 * Set up event listeners for dropdown functionality
 */
function setupDropdownListeners() {
  elements.dropdownButton.addEventListener("click", function(e) {
    e.stopPropagation();
    elements.dropdownContent.classList.toggle("show");
  });

  document.addEventListener("click", function() {
    elements.dropdownContent.classList.remove("show");
  });

  elements.dropdownContent.addEventListener("click", function(e) {
    e.stopPropagation();
  });

  elements.selectAllCheckbox.addEventListener("change", handleSelectAllChange);
}

/**
 * Handle "Select All" checkbox change
 * @param {Event} event - Change event
 */
function handleSelectAllChange() {
  const isChecked = this.checked;
  const sourceCheckboxes = getSourceCheckboxes();

  sourceCheckboxes.forEach((checkbox) => {
    checkbox.checked = isChecked;
  });

  updateDropdownButton();
}

/**
 * Get all source checkboxes
 * @returns {NodeListOf<HTMLInputElement>}
 */
function getSourceCheckboxes() {
  return elements.dropdownContent.querySelectorAll(
    'input[type="checkbox"]:not(#source-all)'
  );
}

/**
 * Update dropdown button label based on selected sources
 */
function updateDropdownButton() {
  const dropdownLabel = document.getElementById("dropdownLabel");
  const sourceCheckboxes = getSourceCheckboxes();
  const checkedSources = Array.from(sourceCheckboxes).filter(
    (cb) => cb.checked
  );

  if (checkedSources.length === 0) {
    dropdownLabel.textContent = "Sources: None Selected";
    elements.selectAllCheckbox.checked = false;
  } else if (checkedSources.length === sourceCheckboxes.length) {
    dropdownLabel.textContent = "Sources: All Selected";
    elements.selectAllCheckbox.checked = true;
  } else {
    dropdownLabel.textContent = `Sources: ${checkedSources.length} Selected`;
    elements.selectAllCheckbox.checked = false;
  }
}

/**
 * Set up search-related event listeners
 */
function setupSearchListeners() {
  elements.searchButton.addEventListener("click", performSearch);
  elements.searchInput.addEventListener("keypress", function(e) {
    if (e.key === "Enter") {
      performSearch();
    }
  });
}

/**
 * Perform subtitle search
 */
function performSearch() {
  const query = elements.searchInput.value.trim();
  if (!validateSearch(query)) {
    return;
  }

  const selectedSources = getSelectedSources();
  if (selectedSources.length === 0) {
    alert("Please select at least one source");
    return;
  }

  updateUIForSearchStart();
  
  elements.resultsList.innerHTML = "";
  let resultsReceived = 0;
  
  state.activeSources = new Set(selectedSources);
  
  createSourceBadges(selectedSources);
  
  const searchUrl = buildSearchUrl(query, selectedSources);

  logToConsole(
    `Searching for: "${query}" on ${selectedSources.length} sources (streaming results)`
  );

  if (state.searchEventSource) {
    state.searchEventSource.close();
  }

  const eventSource = new EventSource(searchUrl);
  state.searchEventSource = eventSource;
  
  updateUIForResults();
  
  setupSearchResultListeners(eventSource, resultsReceived);
}

/**
 * Validate search query
 * @param {string} query - User search query
 * @returns {boolean} - Whether the search is valid
 */
function validateSearch(query) {
  if (!query) {
    alert("Please enter a search term");
    return false;
  }
  return true;
}

/**
 * Get selected subtitle sources
 * @returns {string[]} - Array of selected source names
 */
function getSelectedSources() {
  const selectedSourceCheckboxes = elements.dropdownContent.querySelectorAll(
    'input[type="checkbox"]:not(#source-all):checked'
  );
  return Array.from(selectedSourceCheckboxes).map(
    (cb) => cb.dataset.source
  );
}

/**
 * Update UI elements when search starts
 */
function updateUIForSearchStart() {
  elements.loading.classList.add("show");
  elements.resultsSection.classList.remove("show");
  elements.searchButton.disabled = true;
}

/**
 * Create source status badges
 * @param {string[]} sources - Array of source names
 */
function createSourceBadges(sources) {
  elements.sourceStatus.innerHTML = "";
  elements.sourceStatus.style.display = "flex";
  
  sources.forEach(source => {
    const badge = document.createElement("div");
    badge.className = "source-badge";
    badge.id = `source-badge-${source}`;
    badge.innerHTML = `
      <span class="source-spinner"></span>
      <span>${source}</span>
    `;
    elements.sourceStatus.appendChild(badge);
  });
}

/**
 * Build search URL with query parameters
 * @param {string} query - Search query
 * @param {string[]} sources - Selected sources
 * @returns {string} - Complete search URL
 */
function buildSearchUrl(query, sources) {
  let url = `${API.SEARCH_STREAM}?query=${encodeURIComponent(query)}`;
  if (sources.length > 0) {
    url += `&sources=${sources.join(",")}`;
  }
  return url;
}

/**
 * Update UI for displaying results
 */
function updateUIForResults() {
  elements.resultsSection.classList.add("show");
  elements.resultsList.style.display = "block";
  elements.noResults.style.display = "none";
  elements.resultsCount.textContent = "Searching...";
}

/**
 * Set up event listeners for search results
 * @param {EventSource} eventSource - The SSE connection
 * @param {number} resultsReceived - Counter for received results
 */
function setupSearchResultListeners(eventSource, resultsReceived) {
  eventSource.addEventListener('result', function(event) {
    const result = JSON.parse(event.data);
    addResultItem(result);
    resultsReceived++;
    updateResultsCount(resultsReceived);
    checkScrollFade();
    logToConsole(`Received result: ${result.title} from ${result.source}`);
  });
  
  eventSource.addEventListener('source-complete', function(event) {
    const sourceData = JSON.parse(event.data);
    handleSourceComplete(sourceData);
  });
  
  eventSource.addEventListener('error', function(event) {
    logToConsole(`Error in stream: ${event.data || 'Connection error'}`, true);
  });
  
  eventSource.addEventListener('end', function() {
    handleSearchEnd(resultsReceived);
  });
}

/**
 * Add a result item to the results list
 * @param {Object} result - Result data
 */
function addResultItem(result) {
  const resultItem = document.createElement('li');
  resultItem.className = 'result-item';
  resultItem.innerHTML = `
    <div class="result-title" data-url="${result.url}" data-source="${result.source}" onclick="downloadSubtitle(this)">${result.title}</div>
    <div class="result-source">Source: ${result.source}</div>
  `;
  
  elements.resultsList.appendChild(resultItem);
}

/**
 * Update the results count display
 * @param {number} count - Number of results
 */
function updateResultsCount(count) {
  elements.resultsCount.textContent = `${count} results found`;
}

/**
 * Check if scroll fade should be displayed
 */
function checkScrollFade() {
  if (elements.resultsList.scrollHeight > elements.resultsList.clientHeight) {
    const scrollFade = document.querySelector('.scroll-fade');
    scrollFade.style.display = "block";
    
    elements.resultsList.addEventListener('scroll', function() {
      if (this.scrollHeight - this.scrollTop - this.clientHeight < 10) {
        scrollFade.style.opacity = "0";
      } else {
        scrollFade.style.opacity = "0.8";
      }
    });
  }
}

/**
 * Handle completion of a source search
 * @param {Object} sourceData - Data about completed source
 */
function handleSourceComplete(sourceData) {
  const source = sourceData.source;
  const count = sourceData.count || 0;
  
  if (state.activeSources.has(source)) {
    state.activeSources.delete(source);
    logToConsole(`${source} search complete: found ${count} results`);
    
    const badge = document.getElementById(`source-badge-${source}`);
    if (badge) {
      badge.classList.add("completed");
      badge.innerHTML = `<span>${source} (${count})</span>`;
    }
  }
}

/**
 * Handle search completion
 * @param {number} resultsCount - Total results found
 */
function handleSearchEnd(resultsCount) {
  logToConsole(`Search complete: found ${resultsCount} results from all sources`);
  elements.loading.classList.remove("show");
  elements.searchButton.disabled = false;
  state.searchEventSource.close();
  
  if (resultsCount === 0) {
    elements.resultsList.style.display = "none";
    elements.noResults.style.display = "block";
    document.querySelector('.scroll-fade').style.display = "none";
  }
  
  const pendingBadges = document.querySelectorAll('.source-badge:not(.completed)');
  pendingBadges.forEach(badge => {
    badge.classList.add("completed");
    const sourceName = badge.textContent.trim();
    badge.innerHTML = `<span>${sourceName} (0)</span>`;
    logToConsole(`${sourceName} search complete: found 0 results`);
  });
}

/**
 * Download subtitle file
 * @param {HTMLElement} element - Element that triggered the download
 */
async function downloadSubtitle(element) {
  const url = element.dataset.url;
  const source = element.dataset.source;
  const title = element.textContent;

  if (!url || !source) {
    logToConsole("Error: Missing URL or source for download", true);
    return;
  }

  logToConsole(`Downloading: ${title} from ${source}...`);

  try {
    const downloadUrl = `${API.DOWNLOAD}?url=${encodeURIComponent(
      url
    )}&source=${encodeURIComponent(source)}`;

    triggerDownload(downloadUrl);
    logToConsole(`Download initiated successfully`);
  } catch (error) {
    logToConsole(`Download failed: ${error.message}`, true);
  }
}

/**
 * Trigger browser download using a hidden link
 * @param {string} url - URL to download
 */
function triggerDownload(url) {
  const link = document.createElement("a");
  link.href = url;
  link.style.display = "none";
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
}

/**
 * Initialize the application
 */
function initApp() {
  loadSources();
  setupDropdownListeners();
  setupSearchListeners();
}

/**
 * Clean up resources before page unload
 */
function cleanupResources() {
  if (state.searchEventSource) {
    state.searchEventSource.close();
  }
}

// Initialize the application when DOM is ready
document.addEventListener("DOMContentLoaded", initApp);

// Clean up when page is unloaded
window.addEventListener("beforeunload", cleanupResources);

// Expose functions to global scope that need to be accessed from HTML
window.downloadSubtitle = downloadSubtitle;
