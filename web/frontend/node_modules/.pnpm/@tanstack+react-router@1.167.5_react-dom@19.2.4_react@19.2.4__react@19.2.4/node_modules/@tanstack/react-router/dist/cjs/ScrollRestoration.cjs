require("./_virtual/_rolldown/runtime.cjs");
const require_useRouter = require("./useRouter.cjs");
let _tanstack_router_core = require("@tanstack/router-core");
//#region src/ScrollRestoration.tsx
function useScrollRestoration() {
	(0, _tanstack_router_core.setupScrollRestoration)(require_useRouter.useRouter(), true);
}
/**
* @deprecated Use the `scrollRestoration` router option instead.
*/
function ScrollRestoration(_props) {
	useScrollRestoration();
	if (process.env.NODE_ENV === "development") console.warn("The ScrollRestoration component is deprecated. Use createRouter's `scrollRestoration` option instead.");
	return null;
}
function useElementScrollRestoration(options) {
	useScrollRestoration();
	const router = require_useRouter.useRouter();
	const getKey = options.getKey || _tanstack_router_core.defaultGetScrollRestorationKey;
	let elementSelector = "";
	if (options.id) elementSelector = `[data-scroll-restoration-id="${options.id}"]`;
	else {
		const element = options.getElement?.();
		if (!element) return;
		elementSelector = element instanceof Window ? "window" : (0, _tanstack_router_core.getCssSelector)(element);
	}
	const restoreKey = getKey(router.latestLocation);
	return (_tanstack_router_core.scrollRestorationCache?.state[restoreKey])?.[elementSelector];
}
//#endregion
exports.ScrollRestoration = ScrollRestoration;
exports.useElementScrollRestoration = useElementScrollRestoration;

//# sourceMappingURL=ScrollRestoration.cjs.map