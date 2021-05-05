const fetch = require("node-fetch");
const cheerio = require("cheerio");

const download = async (postUrl) => {
	const res = await fetch(postUrl);
	const html = await res.text();
	const $ = cheerio.load(html);
	const dLink = $("img[src='https://baiscopelk.com/download.png']")
		.parent()
		.attr("href");
	const fileResponse = await fetch(dLink, { method: "POST" });
	return await fileResponse.buffer();
};

module.exports = {
	download,
};
