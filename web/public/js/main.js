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
        setResults((prev) => [...prev, ...data]);
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

function ResultsBox({ results, isLoading }) {
  const handleDownload = (url, source) => {
    swal("Success!", "Subtitle will start downloading shortly.", "success");
    window.open(`/api/download?postUrl=${url}&source=${source}`, "_blank");
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
            <div
              class="s-result"
              onClick=${() => handleDownload(result.postUrl, result.source)}
            >
              <h3>${result.title}</h3>
              <h3 class="download-source">${sourceNames[result.source]}</h3>
              <img src="./img/down.svg" class="download-icon" />
            </div>
          `
        )}
      </div>
    </div>
  `;
}

render(html`<${App} />`, document.getElementById("app"));
