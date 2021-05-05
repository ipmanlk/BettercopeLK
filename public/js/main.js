let searchStatus = false;
const imgLoading = document.getElementById("imgLoading");

document.querySelector("form").addEventListener("submit", (e) => {
	e.preventDefault();

	if (searchStatus) return;

	const keyword = document.getElementById("txtKeyword").value;

	searchResults.innerHTML = "";
	searchStatus = true;
	imgLoading.className = "visible";

	fetch(`./search/${keyword}`)
		.then(async (res) => {
			const response = await res.json();

			if (!response.status) {
				window.alert(response.msg);
				return;
			}

			const searchResults = document.getElementById("searchResults");

			response.data.forEach((result) => {
				searchResults.innerHTML += `
            <div class="panel panel-default">
                        <div class="panel-body">
                            <div class="row">
                                <div class="col-md-4">
                                    <img src="${result.thumbnail}"
                                    width="100%">
                                </div>
                                <div class="col-md-8">
                                    <h4>${result.title}</h4>
                                </div>
                            </div>
                        </div>
                        <div class="panel-footer">
                            <span onClick="downloadSub('${result.postUrl}')" class="btn btn-primary btn-block">Download</span>
                        </div>
                    </div>
            `;
			});

			searchStatus = false;
			imgLoading.className = "hidden";
		})
		.catch((e) => {
			window.alert("Unable to reach the api.");
			searchStatus = false;
			imgLoading.className = "hidden";
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
				link.setAttribute("download", "file");
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
