import { pathToFileURL } from "node:url";
import { tsImport } from "tsx/esm/api";
//#region src/filesystem/virtual/loadConfigFile.ts
async function loadConfigFile(filePath) {
	const fileURL = pathToFileURL(filePath).href;
	return await tsImport(fileURL, "./");
}
//#endregion
export { loadConfigFile };

//# sourceMappingURL=loadConfigFile.js.map