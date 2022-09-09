export const getFilename = (url: string) => {
	const filename = url.trim().replace(/\/$/, "").split("/").pop();
	if (!filename) return "subtitle.zip";
	return !/.+\..{1,4}$/.test(filename) ? `${filename}.zip` : filename;
};
