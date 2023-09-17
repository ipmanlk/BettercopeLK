let searchStatus = false;

const sourceNames = {
  baiscopelk: "Baiscope.lk",
  cineru: "Cineru.lk",
  piratelk: "Piratelk.com",
};

const loadingHTML = `<center><img class="spinner-img" src="./img/loading.gif" width="100px" /></center>`;

document.querySelector("form").addEventListener("submit", (e) => {
  e.preventDefault();
  if (searchStatus) return;
  searchSubtitles();
});

const searchSubtitles = async () => {
  const keyword = document.querySelector(".search-bar").value;
  const searchResults = document.querySelector(".s-results");

  searchResults.innerHTML = loadingHTML;
  searchResults.classList.remove("hidden");

  searchStatus = true;

  const eventSource = new EventSource(encodeURI(`/search?keyword=${keyword}`));

  eventSource.addEventListener("result", (event) => {
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

    const listItems = [];

    data.forEach((result) => {
      listItems.push(`
			<div class="s-result" onClick="downloadSub('${result.postUrl}', '${
        result.source
      }')">
				<h3>${result.title}</h3>
				<h3 class="download-source">${sourceNames[result.source]}</h3>
				<img src="./img/down.svg" class="download-icon" />
			</div>`);
    });

    searchResults.innerHTML += listItems.join("");
  });

  eventSource.addEventListener("error", (event) => {
    console.error("Error: ", event);
    eventSource.close();
    searchStatus = false;
    resetState();
  });

  eventSource.addEventListener("end", (event) => {
    console.debug("End: ", event);
    eventSource.close();
    searchStatus = false;
  });
};

const downloadSub = (postUrl, source) => {
  swal("Success!", "Subtitle will start download shortly.", "success");
  document.getElementById("txtPostUrl").value = postUrl;
  document.getElementById("txtSource").value = source;
  setTimeout(() => {
    document.getElementById("btnDownload").click();
  }, 1000);
};

const resetState = () => {
  document.querySelector(".s-results").classList.add("hidden");
  document.querySelector(".s-results").innerHTML = "";
  document.querySelector(".search-bar").value = "";
};
