import { load } from "cheerio";
import { request } from "undici";
import { Source } from "./types";
import { getFilename } from "./util";

const downloadBaiscopelk = async (postUrl: string) => {
	// extract html from the webpage
	const { body } = await request(postUrl);
	const html = await body.text();

	// find download link
	const $ = load(html);
	const dLink = $("img[src='https://baiscopelk.com/download.png']")
		.parent()
		.attr("href") as string;

	// download sub to memory
	const { body: downloadBody } = await request(dLink, { method: "POST" });

	// return it as a buffer
	return {
		filename: getFilename(dLink),
		file: Buffer.from(await downloadBody.arrayBuffer()),
	};
};

const downloadCineru = async (postUrl: string) => {
	const { body } = await request(postUrl);
	const html = await body.text();

	const $ = load(html);
	const dLink = $("#btn-download").data("link") as string;

	const { body: downloadBody } = await request(dLink);

	return {
		filename: getFilename(dLink),
		file: Buffer.from(await downloadBody.arrayBuffer()),
	};
};

export const downloadSubtitle = async (postUrl: string, source: Source) => {
	switch (source) {
		case "baiscopelk":
			return downloadBaiscopelk(postUrl);

		case "cineru":
			return downloadCineru(postUrl);
	}
};
