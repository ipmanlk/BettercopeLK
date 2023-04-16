const BAISCOPELK_REGEX = /filename\*=UTF-8''([^;]+)/;

export const getBaiscopelkFilename = (contentDisposition: string) => {
  const result = contentDisposition.match(BAISCOPELK_REGEX);
  return result ? result[1] : "subtitle.zip";
};

export const getFilenameFromUrl = (url: string) => {
  const filename = url.trim().replace(/\/$/, "").split("/").pop();
  if (!filename) return "subtitle.zip";
  return !/.+\..{1,4}$/.test(filename) ? `${filename}.zip` : filename;
};
