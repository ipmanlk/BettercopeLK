const HEADER_REGEX = /filename=["']([^"']+)["']/;

export const getFilenameFromHeader = (contentDisposition: string) => {
  const result = contentDisposition.match(HEADER_REGEX);
  return result ? result[1] : "subtitle.zip";
};

export const getFilenameFromUrl = (url: string) => {
  const filename = url.trim().replace(/\/$/, "").split("/").pop();
  if (!filename) return "subtitle.zip";
  return !/.+\..{1,4}$/.test(filename) ? `${filename}.zip` : filename;
};
