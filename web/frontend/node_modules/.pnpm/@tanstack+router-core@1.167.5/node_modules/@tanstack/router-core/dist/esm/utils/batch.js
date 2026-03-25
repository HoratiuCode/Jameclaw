import { batch } from "@tanstack/store";
import { isServer } from "@tanstack/router-core/isServer";
//#region src/utils/batch.ts
function batch$1(fn) {
	if (isServer) return fn();
	let result;
	batch(() => {
		result = fn();
	});
	return result;
}
//#endregion
export { batch$1 as batch };

//# sourceMappingURL=batch.js.map