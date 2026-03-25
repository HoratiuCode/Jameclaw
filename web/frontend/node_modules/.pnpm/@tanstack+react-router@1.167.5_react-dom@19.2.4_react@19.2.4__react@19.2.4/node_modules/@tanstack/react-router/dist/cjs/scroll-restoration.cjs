require("./_virtual/_rolldown/runtime.cjs");
const require_useRouter = require("./useRouter.cjs");
const require_ScriptOnce = require("./ScriptOnce.cjs");
let _tanstack_router_core = require("@tanstack/router-core");
let react_jsx_runtime = require("react/jsx-runtime");
let _tanstack_router_core_isServer = require("@tanstack/router-core/isServer");
//#region src/scroll-restoration.tsx
function ScrollRestoration() {
	const router = require_useRouter.useRouter();
	if (!router.isScrollRestoring || !(_tanstack_router_core_isServer.isServer ?? router.isServer)) return null;
	if (typeof router.options.scrollRestoration === "function") {
		if (!router.options.scrollRestoration({ location: router.latestLocation })) return null;
	}
	const userKey = (router.options.getScrollRestorationKey || _tanstack_router_core.defaultGetScrollRestorationKey)(router.latestLocation);
	const resolvedKey = userKey !== (0, _tanstack_router_core.defaultGetScrollRestorationKey)(router.latestLocation) ? userKey : void 0;
	const restoreScrollOptions = {
		storageKey: _tanstack_router_core.storageKey,
		shouldScrollRestoration: true
	};
	if (resolvedKey) restoreScrollOptions.key = resolvedKey;
	return /* @__PURE__ */ (0, react_jsx_runtime.jsx)(require_ScriptOnce.ScriptOnce, { children: `(${_tanstack_router_core.restoreScroll.toString()})(${(0, _tanstack_router_core.escapeHtml)(JSON.stringify(restoreScrollOptions))})` });
}
//#endregion
exports.ScrollRestoration = ScrollRestoration;

//# sourceMappingURL=scroll-restoration.cjs.map