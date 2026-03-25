import { useRouter } from "./useRouter.js";
import { ScriptOnce } from "./ScriptOnce.js";
import { defaultGetScrollRestorationKey, escapeHtml, restoreScroll, storageKey } from "@tanstack/router-core";
import { jsx } from "react/jsx-runtime";
import { isServer } from "@tanstack/router-core/isServer";
//#region src/scroll-restoration.tsx
function ScrollRestoration() {
	const router = useRouter();
	if (!router.isScrollRestoring || !(isServer ?? router.isServer)) return null;
	if (typeof router.options.scrollRestoration === "function") {
		if (!router.options.scrollRestoration({ location: router.latestLocation })) return null;
	}
	const userKey = (router.options.getScrollRestorationKey || defaultGetScrollRestorationKey)(router.latestLocation);
	const resolvedKey = userKey !== defaultGetScrollRestorationKey(router.latestLocation) ? userKey : void 0;
	const restoreScrollOptions = {
		storageKey,
		shouldScrollRestoration: true
	};
	if (resolvedKey) restoreScrollOptions.key = resolvedKey;
	return /* @__PURE__ */ jsx(ScriptOnce, { children: `(${restoreScroll.toString()})(${escapeHtml(JSON.stringify(restoreScrollOptions))})` });
}
//#endregion
export { ScrollRestoration };

//# sourceMappingURL=scroll-restoration.js.map