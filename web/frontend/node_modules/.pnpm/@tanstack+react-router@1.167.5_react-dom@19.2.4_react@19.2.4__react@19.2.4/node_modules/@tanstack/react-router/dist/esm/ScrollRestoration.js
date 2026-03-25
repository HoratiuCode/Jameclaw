import { useRouter } from "./useRouter.js";
import { defaultGetScrollRestorationKey, getCssSelector, scrollRestorationCache, setupScrollRestoration } from "@tanstack/router-core";
//#region src/ScrollRestoration.tsx
function useScrollRestoration() {
	setupScrollRestoration(useRouter(), true);
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
	const router = useRouter();
	const getKey = options.getKey || defaultGetScrollRestorationKey;
	let elementSelector = "";
	if (options.id) elementSelector = `[data-scroll-restoration-id="${options.id}"]`;
	else {
		const element = options.getElement?.();
		if (!element) return;
		elementSelector = element instanceof Window ? "window" : getCssSelector(element);
	}
	const restoreKey = getKey(router.latestLocation);
	return (scrollRestorationCache?.state[restoreKey])?.[elementSelector];
}
//#endregion
export { ScrollRestoration, useElementScrollRestoration };

//# sourceMappingURL=ScrollRestoration.js.map