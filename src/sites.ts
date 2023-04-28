import { load } from "cheerio";
import { request } from "undici";
import { SearchResult, SearchSource, Source } from "./types";
import { EventEmitter } from "stream";

const searchWpSite = async (url: string, source: Source) => {
  const { body } = await request(url, {
    bodyTimeout: 30000,
    headersTimeout: 30000,
  });

  const resultsHtml = await body.text();
  const $ = load(resultsHtml);

  const searchResults: SearchResult[] = [];

  $(".item-list").each((_, element) => {
    const postBox = $(element).find(".post-box-title a");

    const title = $(postBox).text().trim();
    if (title === "Collection") return;

    const postUrl = $(postBox).attr("href");
    if (!postUrl) return;

    searchResults.push({ title, postUrl, source });
  });

  return searchResults;
};

const getSources = (keyword: string): SearchSource[] => {
  return [
    {
      url: `https://www.baiscopelk.com/?s=${encodeURIComponent(keyword)}`,
      name: "baiscopelk",
    },
    {
      url: `https://cineru.lk/?s=${encodeURIComponent(keyword)}`,
      name: "cineru",
    },
  ];
};

export class SiteCrawler extends EventEmitter {
  private keyword: string;

  constructor(keyword: string) {
    super();
    this.keyword = keyword;
  }

  public async start() {
    const sources = getSources(this.keyword);

    for (const source of sources) {
      try {
        const results = await searchWpSite(source.url, source.name);
        this.emit("data", results);
      } catch (e) {
        console.error(e);
      }
    }

    this.emit("end");
  }
}
