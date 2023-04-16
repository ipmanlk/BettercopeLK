import cheerio from "cheerio";
import { request } from "undici";
import { SearchResult, Source } from "./types";

const searchBaiscopelk = async (keyword: string): Promise<SearchResult[]> => {
  const { body } = await request(
    `https://www.baiscopelk.com/?s=${encodeURIComponent(keyword)}`,
    { bodyTimeout: 30000, headersTimeout: 30000 }
  );

  const resultsHtml = await body.text();
  const $ = cheerio.load(resultsHtml);

  const searchResults: { title: string; postUrl: string; source: Source }[] =
    [];

  $(".item-list").each((_, element) => {
    const postBox = $(element).find(".post-box-title a");

    const title = $(postBox).text().trim();
    if (title === "Collection") return;

    const postUrl = $(postBox).attr("href");
    if (!postUrl) return;

    searchResults.push({ title, postUrl, source: "baiscopelk" });
  });

  return searchResults;
};

const searchCineru = async (keyword: string): Promise<SearchResult[]> => {
  const { body } = await request(
    `https://cineru.lk/?s=${encodeURIComponent(keyword)}`,
    { bodyTimeout: 30000, headersTimeout: 30000 }
  );

  const resultsHtml = await body.text();
  const $ = cheerio.load(resultsHtml);

  const searchResults: { title: string; postUrl: string; source: Source }[] =
    [];

  $(".item-list").each((_, element) => {
    const postBox = $(element).find(".post-box-title a");

    const title = $(postBox).text().trim();
    if (title === "Collection") return;

    const postUrl = $(postBox).attr("href");
    if (!postUrl) return;
    searchResults.push({ title, postUrl, source: "cineru" });
  });

  return searchResults;
};

export const searchSites = async (keyword: string) => {
  let searchResults: SearchResult[] = [];

  for (const search of [searchBaiscopelk, searchCineru]) {
    try {
      const results = await search(keyword);
      searchResults = [...searchResults, ...results];
    } catch (e) {
      console.error(e);
    }
  }

  return searchResults;
};
