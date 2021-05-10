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

			if (!response.status) {
				window.alert(response.msg);
				return;
			}

			searchResults.innerHTML = "";
			let html = "";
			response.data.forEach((result) => {
				html += `
				<div class="s-result">
					<h3 onClick="downloadSub('${result.postUrl}')">${result.title}</h3>
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

const downloadSub = (postUrl) => {
	fetch("/download", {
		method: "POST",
		headers: {
			"Content-Type": "application/json",
		},
		body: JSON.stringify({ postUrl: postUrl }),
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
