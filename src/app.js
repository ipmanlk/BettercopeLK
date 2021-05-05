const express = require("express");
const baiscopelk = require("./baiscopelk");
const downloader = require("./download");

const app = express();
app.use(express.json());

const port = process.env.PORT || 3000;

app.use("/", express.static("public"));

app.get("/search/:keyword", async (req, res) => {
	baiscopelk
		.search(req.params.keyword || "")
		.then((subs) => {
			res.json({ status: true, data: subs });
		})
		.catch((e) => {
			console.log(e);
			res.json({ status: false, msg: e });
		});
});

app.post("/download", (req, res) => {
	downloader
		.download(req.body.postUrl || "")
		.then((buffer) => {
			res.set("Content-Type", "application/zip");
			res.send(buffer);
		})
		.catch((e) => {
			console.log(e);
			res.json({ status: false, msg: e });
		});
});

app.listen(port, () =>
	console.log(`App listening at http://localhost:${port}`)
);
