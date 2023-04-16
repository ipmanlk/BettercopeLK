export type Source = "baiscopelk" | "cineru";

export type SearchSource = {
  url: string;
  name: Source;
};

export type SearchResult = { title: string; postUrl: string; source: Source };
