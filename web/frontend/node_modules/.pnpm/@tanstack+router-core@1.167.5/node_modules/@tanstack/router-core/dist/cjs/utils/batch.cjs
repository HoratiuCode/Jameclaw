require("../_virtual/_rolldown/runtime.cjs");
let _tanstack_store = require("@tanstack/store");
let _tanstack_router_core_isServer = require("@tanstack/router-core/isServer");
//#region src/utils/batch.ts
function batch(fn) {
	if (_tanstack_router_core_isServer.isServer) return fn();
	let result;
	(0, _tanstack_store.batch)(() => {
		result = fn();
	});
	return result;
}
//#endregion
exports.batch = batch;

//# sourceMappingURL=batch.cjs.map