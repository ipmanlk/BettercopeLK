const sourceNames = {
  baiscopelk: "Baiscope.lk",
  cineru: "Cineru.lk",
  piratelk: "Piratelk.com",
  zoomlk: "Zoom.lk",
};

const loadingHTML = `<center><img class="spinner-img" src="./img/loading.gif" width="100px" /></center>`;
const searchForm = document.querySelector("form");
const searchResults = document.querySelector(".s-results");
const searchBar = document.querySelector(".search-bar");

let searchStatus = false;
let eventSource = null;

searchForm.addEventListener("submit", async (e) => {
  e.preventDefault();
  if (searchStatus) return;

  const keyword = searchBar.value;
  searchResults.innerHTML = loadingHTML;
  searchResults.classList.remove("hidden");
  searchStatus = true;

  eventSource = new EventSource(encodeURI(`/api/search?query=${keyword}`));
  eventSource.addEventListener("results", handleResults);
  eventSource.addEventListener("error", handleError);
  eventSource.addEventListener("end", handleEnd);
});

const handleResults = (event) => {
  console.debug("Results: ", event);
  const data = JSON.parse(event.data);

  if (data.error) {
    swal("Error!", data.error, "error");
    resetState();
    return;
  }

  if (!data?.length) return;
  if (searchResults.innerHTML.includes("loading.gif")) {
    searchResults.innerHTML = "";
  }

  const listItems = data.map((result) => {
    return `
      <div class="s-result" onClick="downloadSub('${result.postUrl}', '${
      result.source
    }')">
        <h3>${result.title}</h3>
        <h3 class="download-source">${sourceNames[result.source]}</h3>
        <img src="./img/down.svg" class="download-icon" />
      </div>`;
  });

  searchResults.innerHTML += listItems.join("");
};

const handleError = (event) => {
  console.debug("Error: ", event);

  if (event.data === "MISSING_QUERY") {
    swal("Error!", "Please enter a keyword to search.", "error");
    resetState();
  }

  if (eventSource) {
    eventSource.close();
  }
  searchStatus = false;
};

const handleEnd = (event) => {
  console.debug("End: ", event);
  if (eventSource) {
    eventSource.close();
  }
  searchStatus = false;

  if (searchResults.innerHTML.includes("loading.gif")) {
    resetState();
    swal("Sorry!", "No results found.", "error");
  }
};

const downloadSub = (postUrl, source) => {
  swal("Success!", "Subtitle will start downloading shortly.", "success");
  window.open(`/api/download?postUrl=${postUrl}&source=${source}`, "_blank");
};

const resetState = () => {
  searchResults.classList.add("hidden");
  searchResults.innerHTML = "";
  searchBar.value = "";
};
