import { load } from "cheerio";
import { request } from "undici";
import { SearchResult, SearchSource, Source } from "./types";
import { EventEmitter } from "stream";

const scrapeWpSite = async (url: string, source: Source) => {
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

const scrapeBaiscopeLk = async (url: string) => {
  const { body } = await request(url, {
    bodyTimeout: 30000,
    headersTimeout: 30000,
  });

  const resultsHtml = await body.text();
  const $ = load(resultsHtml);

  const searchResults: SearchResult[] = [];

  $("article.post").each((_, element) => {
    const entryLink = $(element).find(".entry-title a");
    if (!entryLink) return;

    const title = $(entryLink).text().trim();
    if (!title || title === "Collection") return;

    const postUrl = $(entryLink).attr("href");
    if (!postUrl) return;

    searchResults.push({ title, postUrl, source: "baiscopelk" });
  });

  return searchResults;
};

const getSources = (keyword: string): SearchSource[] => {
  return [
    {
      url: encodeURI(`https://www.baiscope.lk/?s=${keyword}`),
      name: "baiscopelk",
    },
    {
      url: encodeURI(`https://cineru.lk/?s=${keyword}`),
      name: "cineru",
    },
    {
      url: encodeURI(`https://piratelk.com/?s=${keyword}`),
      name: "piratelk",
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

    await Promise.allSettled(
      sources.map(async (source) => {
        try {
          let results;

          if (source.name === "baiscopelk") {
            results = await scrapeBaiscopeLk(source.url);
          } else {
            results = await scrapeWpSite(source.url, source.name);
          }

          this.emit("data", results);
        } catch (e) {
          console.error(e);
        }
      })
    );

    this.emit("end");
  }
}
