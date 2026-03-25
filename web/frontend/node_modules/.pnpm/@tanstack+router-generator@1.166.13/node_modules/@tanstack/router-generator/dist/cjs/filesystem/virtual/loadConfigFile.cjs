require("../../_virtual/_rolldown/runtime.cjs");
let node_url = require("node:url");
let tsx_esm_api = require("tsx/esm/api");
//#region src/filesystem/virtual/loadConfigFile.ts
async function loadConfigFile(filePath) {
	const fileURL = (0, node_url.pathToFileURL)(filePath).href;
	return await (0, tsx_esm_api.tsImport)(fileURL, "./");
}
//#endregion
exports.loadConfigFile = loadConfigFile;

//# sourceMappingURL=loadConfigFile.cjs.map