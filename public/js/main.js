const sourceNames = {
  baiscopelk: "Baiscope.lk",
  cineru: "Cineru.lk",
  piratelk: "Piratelk.com",
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

  try {
    eventSource = new EventSource(encodeURI(`/search?keyword=${keyword}`));
    eventSource.addEventListener("result", handleResult);
    eventSource.addEventListener("error", handleError);
    eventSource.addEventListener("end", handleEnd);
  } catch (error) {
    console.error("Error:", error);
    handleError();
  }
});

const handleResult = (event) => {
  console.debug("Result: ", event);
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
      <div class="s-result" onClick="downloadSub('${result.postUrl}', '${result.source}')">
        <h3>${result.title}</h3>
        <h3 class="download-source">${sourceNames[result.source]}</h3>
        <img src="./img/down.svg" class="download-icon" />
      </div>`;
  });

  searchResults.innerHTML += listItems.join("");
};

const handleError = () => {
  console.error("Error: ");
  if (eventSource) {
    eventSource.close();
  }
  searchStatus = false;
};

const handleEnd = () => {
  console.debug("End: ");
  if (eventSource) {
    eventSource.close();
  }
  searchStatus = false;
};

const downloadSub = (postUrl, source) => {
  swal("Success!", "Subtitle will start downloading shortly.", "success");
  document.getElementById("txtPostUrl").value = postUrl;
  document.getElementById("txtSource").value = source;
  setTimeout(() => {
    document.getElementById("btnDownload").click();
  }, 1000);
};

const resetState = () => {
  searchResults.classList.add("hidden");
  searchResults.innerHTML = "";
  searchBar.value = "";
};
