export type Source = "baiscopelk" | "cineru" | "piratelk";

export type SearchSource = {
  url: string;
  name: Source;
};

export type SearchResult = { title: string; postUrl: string; source: Source };
