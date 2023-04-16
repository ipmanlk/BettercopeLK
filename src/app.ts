import express from "express";
import { downloadSubtitle } from "./download";
import { searchSites } from "./sites";

const app = express();
const port = process.env.PORT || 3000;

app.use(express.urlencoded({ extended: false }));

app.use("/", express.static(`${__dirname}/../public`));

app.get("/search/:keyword", async (req, res) => {
  try {
    const results = await searchSites(req.params.keyword);
    res.json({ data: results });
  } catch (e) {
    console.error(e);
    res.json({ error: e?.toString() });
  }
});

app.post("/download", async (req, res) => {
  if (!req.body.postUrl || !req.body.source) {
    res.statusMessage = "Invalid data provided";
    return res.status(400).end();
  }

  try {
    const data = await downloadSubtitle(req.body.postUrl, req.body.source);
    res.set("Content-Type", "application/zip");
    res.set("Content-Disposition", `attachment; filename="${data.filename}"`);
    res.send(data.file);
  } catch (e) {
    console.error(e);
    res.json({ error: e?.toString() });
  }
});

app.listen(port, () => {
  console.info(`Server is running on port ${port}`);
});
