const express = require("express");
const baiscopelk = require("./baiscopelk");

const app = express();
const port = process.env.PORT || 3000;

app.use("/", express.static("public"));

app.get("/search/:keyword", async (req, res) => {
    baiscopelk.search(req.params.keyword || "").then(subs => {
        res.json({ status: true, data: subs });
    }).catch(e => {
        res.json({ status: false, msg: e });
    });
});


app.listen(port, () => console.log(`App listening at http://localhost:${port}`));