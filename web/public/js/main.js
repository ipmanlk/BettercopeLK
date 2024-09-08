import { h, render } from "https://esm.sh/preact";
import { useState } from "https://esm.sh/preact/hooks";
import htm from "https://esm.sh/htm";

// Initialize htm with Preact
const html = htm.bind(h);

// Humanized source names
const sourceNames = {
  baiscopelk: "Baiscope.lk",
  cineru: "Cineru.lk",
  piratelk: "Piratelk.com",
  zoomlk: "Zoom.lk",
};

function App() {
  const [results, setResults] = useState([]);
  const [isLoading, setIsLoading] = useState(false);
  const [selectedResults, setSelectedResults] = useState([]);

  return html`
    <div class="logo">
      <h1>BettercopeLK</h1>
      <i>Better way to search & download Sinhala subtitles</i>
    </div>

    ${SearchBox({
      results,
      setResults,
      setIsLoading,
      isLoading,
    })}
    ${ResultsBox({
      results,
      isLoading,
      selectedResults,
      setSelectedResults,
    })}
  `;
}

function SearchBox({ results, setResults, setIsLoading, isLoading }) {
  const [query, setQuery] = useState("");

  const handleSubmit = (e) => {
    e.preventDefault();
    if (isLoading) return;

    setIsLoading(true);
    setResults([]);
    window.hasResults = false;

    const eventSource = new EventSource(
      encodeURI(`/api/search?query=${query}`)
    );
    eventSource.addEventListener("results", (event) => {
      console.debug("Results: ", event);
      try {
        const data = JSON.parse(event.data);
        setResults((prev) => {
          const newResults = [...prev, ...data];

          // Sort results by title based on search query
          // matches will be shown first
          newResults.sort((a, b) => {
            const aTitle = a.title.toLowerCase();
            const bTitle = b.title.toLowerCase();
            const queryLower = query.toLowerCase();

            if (aTitle.includes(queryLower) && !bTitle.includes(queryLower)) {
              return -1;
            }
            if (!aTitle.includes(queryLower) && bTitle.includes(queryLower)) {
              return 1;
            }
            return 0;
          });

          return newResults;
        });
        if (data.length > 0) {
          window.hasResults = true;
        }
      } catch (e) {
        console.error(e);
      }
    });
    eventSource.addEventListener("error", (event) => {
      console.debug("Error: ", event);
      if (event.data === "MISSING_QUERY") {
        swal("Error!", "Please enter a keyword to search.", "error");
      }
      if (eventSource) {
        eventSource.close();
      }
      setIsLoading(false);
    });
    eventSource.addEventListener("end", (event) => {
      console.debug("End: ", event);
      if (eventSource) {
        eventSource.close();
      }
      setIsLoading(false);

      if (!window.hasResults) {
        swal(
          "Oops!",
          "We couldn't find any subtitles for your search query.",
          "error"
        );
      }
    });
  };

  return html`
    <div class="controls-holder">
      <form class="controls" onSubmit=${handleSubmit}>
        <input
          type="text"
          placeholder="Type your keyword here..."
          class="search-bar"
          value=${query}
          onInput=${(e) => setQuery(e.target.value)}
        />
        <input
          type="submit"
          value="Search"
          class="search-btn"
          disabled=${isLoading}
        />
      </form>
    </div>
  `;
}

function ResultsBox({
  results,
  isLoading,
  selectedResults,
  setSelectedResults,
}) {
  const handleDownload = (url, source) => {
    // Single subtitle download
    if (selectedResults.length === 0) {
      swal("Success!", "Subtitle will start downloading shortly.", "success");
      window.open(`/api/download?postUrl=${url}&source=${source}`, "_blank");
      return;
    }

    // Bulk download
    // send selected results array as json to /api/bulk-download
    // it will return application/zip with Content-Type and Content-Disposition headers
    // so browser should save that file automatically
    const data = JSON.stringify({ data: selectedResults });
    const xhr = new XMLHttpRequest();
    xhr.open("POST", "/api/bulk-download", true);
    xhr.setRequestHeader("Content-Type", "application/json");
    xhr.responseType = "blob";
    xhr.onload = function () {
      if (this.status === 200) {
        const filename = "subtitles.zip";
        const blob = new Blob([this.response], { type: "application/zip" });
        const link = document.createElement("a");
        link.href = window.URL.createObjectURL(blob);
        link.download = filename;
        link.click();
      }
    };
    xhr.onerror = function () {
      swal("Error!", "Something went wrong. Please try again.", "error");
    };
    xhr.send(data);

    // Reset selected results
    setSelectedResults([]);
  };

  const handleSelect = (e, result) => {
    if (e.target.checked) {
      if (selectedResults.length > 20) {
        e.target.checked = false;
        swal(
          "Oops!",
          "You can't bulk download more than 20 subtitles at once.",
          "error"
        );
        return;
      }

      setSelectedResults((prev) => {
        if (prev.includes(result)) {
          return prev;
        }
        return [...prev, result];
      });
      return;
    }

    setSelectedResults((prev) => {
      return prev.filter((item) => item !== result);
    });
  };

  return html`
    <div class="s-results-holder">
      <div class="s-results ${!isLoading && !results.length && "hidden"}">
        ${isLoading && !results.length
          ? html`<center>
              <img class="spinner-img" src="./img/loading.gif" width="100px" />
            </center>`
          : ""}
        ${results.map(
          (result) => html`
            <div class="s-result">
              <input
                type="checkbox"
                onChange=${(e) => handleSelect(e, result)}
                checked=${selectedResults.includes(result)}
              />
              <div
                class="s-result-content"
                onClick=${() => handleDownload(result.postUrl, result.source)}
              >
                <h3>${result.title}</h3>
                <h3 class="download-source">${sourceNames[result.source]}</h3>
                <img src="./img/down.svg" class="download-icon" />
              </div>
            </div>
          `
        )}
      </div>
    </div>
  `;
}

render(html`<${App} />`, document.getElementById("app"));
