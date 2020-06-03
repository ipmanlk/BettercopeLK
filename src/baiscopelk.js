const fetch = require("node-fetch");
const cheerio = require("cheerio");

const search = async (keyword) => {
    const results = await fetch(`https://www.baiscopelk.com/?s=${keyword}`).catch(e => {
        throw ("Error!. Unable to reach baiscopelk.");
    });

    const resultsHtml = await results.text().catch(e => {
        throw ("Error!. Unable to extract baiscopelk html response.");
    });

    const $ = cheerio.load(resultsHtml);

    const searchResults = [];

    try {
        $(".item-list").each((i, element) => {
            const postBox = $(element).find(".post-box-title a");
            const title = $(postBox).text();
            const postUrl = $(postBox).attr("href");
            const thumbnail = $(element).find(".post-thumbnail a img").attr("src");
            searchResults.push({ title, postUrl, thumbnail });
        });
    } catch (e) {
        throw ("Error!. Unable to parse baiscopelk search results.");
    }

    if (searchResults.length == 0) {
        throw ("No subtitles found for that keyword!.");
    }

    return searchResults;
}

module.exports = {
    search
}