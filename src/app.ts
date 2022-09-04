import express from "express";
import { downloadSubtitle } from "./download";
import { searchSites } from "./sites";

const app = express();
const port = process.env.PORT || 3000;

app.use(express.json());
app.use("/", express.static(`${__dirname}/../public`));

app.get("/search/:keyword", async (req, res) => {
	searchSites(req.params.keyword)
		.then((results) => {
			res.json({ data: results });
		})
		.catch((e) => {
			console.error(e);
			res.json({ error: e?.toString() });
		});
});

app.post("/download", (req, res) => {
	downloadSubtitle(req.body.postUrl || "", req.body.source || "")
		.then((buffer) => {
			res.set("Content-Type", "application/zip");
			res.send(buffer);
		})
		.catch((e) => {
			console.error(e);
			res.json({ error: e?.toString() });
		});
});

app.listen(port, () => {
	console.info(`Server is running on port ${port}`);
});
