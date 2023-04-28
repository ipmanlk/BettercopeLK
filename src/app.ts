import express from "express";
import { downloadSubtitle } from "./download";
import { SiteCrawler } from "./sites";

const app = express();
const port = process.env.PORT || 3000;

app.use(express.urlencoded({ extended: false }));

app.use("/", express.static(`${__dirname}/../public`));

app.get("/search/:keyword", async (req, res) => {
  res.setHeader("Content-Type", "text/event-stream");
  res.setHeader("Cache-Control", "no-cache");
  res.setHeader("Connection", "keep-alive");
  res.flushHeaders();

  if (!req.params.keyword) {
    res.write(`data: ${JSON.stringify({ error: "Invalid keyword" })}\n\n`);
    res.end();
    return;
  }

  const siteCrawler = new SiteCrawler(req.params.keyword);

  siteCrawler.on("data", (data) => {
    res.write(`data: ${JSON.stringify(data)}\n\n`);
  });

  siteCrawler.on("end", () => {
    res.write("event: end\n");
    res.write("data: end\n\n");
    res.end();
  });

  siteCrawler.start();
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
