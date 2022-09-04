let searchStatus = false;

document.querySelector("form").addEventListener("submit", (e) => {
	e.preventDefault();
	if (searchStatus) return;

	const keyword = document.querySelector(".search-bar").value;
	const searchResults = document.querySelector(".s-results");
	const loadingHTML = `<center><img class="spinner-img" src="./img/loading.gif" width="100px" /></center>`;
	searchResults.innerHTML = loadingHTML;
	searchResults.classList.remove("hidden");

	searchStatus = true;

	fetch(`./search/${keyword}`)
		.then(async (res) => {
			const response = await res.json();

			if (res.error) {
				window.alert(response.msg);
				return;
			}

			searchResults.innerHTML = "";
			let html = "";
			response.data.forEach((result) => {
				html += `
				<div class="s-result" onClick="downloadSub('${result.postUrl}', '${
					result.source
				}')">
					<h3>${result.title}</h3>
					<h3 class="download-source">${getReadableSource(result.source)}</h3>
					<img src="./img/down.svg" class="download-icon" />
				</div>`;
			});
			searchResults.innerHTML = html;
			searchStatus = false;
		})
		.catch((e) => {
			searchResults.classList.add("hidden");
			console.log(e);
			window.alert("Unable to reach the api.");
			searchStatus = false;
		});
});

const downloadSub = (postUrl, source) => {
	swal("Success!", "Subtitle will start download shortly.", "success");

	fetch("/download", {
		method: "POST",
		headers: {
			"Content-Type": "application/json",
		},
		body: JSON.stringify({ postUrl, source }),
	})
		.then((res) => {
			if (!res.status) {
				window.alert(res.msg);
				return;
			}
			res.blob().then((blob) => {
				const downloadUrl = window.URL.createObjectURL(blob);
				const link = document.createElement("a");
				link.setAttribute("href", downloadUrl);
				link.setAttribute("download", "subtitle.zip");
				link.style.display = "none";
				document.body.appendChild(link);
				link.click();
				window.URL.revokeObjectURL(link.href);
				document.body.removeChild(link);
			});
		})
		.catch((e) => {
			console.log(e);
			window.alert("Unable to reach the api.");
		});
};

const getReadableSource = (source) => {
	switch (source) {
		case "baiscopelk":
			return "Baiscope.lk";

		case "cineru":
			return "Cineru.lk";
	}
};
