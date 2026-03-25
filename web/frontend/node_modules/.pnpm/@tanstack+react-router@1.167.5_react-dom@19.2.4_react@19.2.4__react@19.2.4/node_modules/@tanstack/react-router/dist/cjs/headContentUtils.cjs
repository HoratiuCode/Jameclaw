const require_runtime = require("./_virtual/_rolldown/runtime.cjs");
const require_useRouter = require("./useRouter.cjs");
const require_useRouterState = require("./useRouterState.cjs");
let _tanstack_router_core = require("@tanstack/router-core");
let react = require("react");
react = require_runtime.__toESM(react);
//#region src/headContentUtils.tsx
/**
* Build the list of head/link/meta/script tags to render for active matches.
* Used internally by `HeadContent`.
*/
var useTags = () => {
	const router = require_useRouter.useRouter();
	const nonce = router.options.ssr?.nonce;
	const routeMeta = require_useRouterState.useRouterState({ select: (state) => {
		return state.matches.map((match) => match.meta).filter(Boolean);
	} });
	const meta = react.useMemo(() => {
		const resultMeta = [];
		const metaByAttribute = {};
		let title;
		for (let i = routeMeta.length - 1; i >= 0; i--) {
			const metas = routeMeta[i];
			for (let j = metas.length - 1; j >= 0; j--) {
				const m = metas[j];
				if (!m) continue;
				if (m.title) {
					if (!title) title = {
						tag: "title",
						children: m.title
					};
				} else if ("script:ld+json" in m) try {
					const json = JSON.stringify(m["script:ld+json"]);
					resultMeta.push({
						tag: "script",
						attrs: { type: "application/ld+json" },
						children: (0, _tanstack_router_core.escapeHtml)(json)
					});
				} catch {}
				else {
					const attribute = m.name ?? m.property;
					if (attribute) if (metaByAttribute[attribute]) continue;
					else metaByAttribute[attribute] = true;
					resultMeta.push({
						tag: "meta",
						attrs: {
							...m,
							nonce
						}
					});
				}
			}
		}
		if (title) resultMeta.push(title);
		if (nonce) resultMeta.push({
			tag: "meta",
			attrs: {
				property: "csp-nonce",
				content: nonce
			}
		});
		resultMeta.reverse();
		return resultMeta;
	}, [routeMeta, nonce]);
	const links = require_useRouterState.useRouterState({
		select: (state) => {
			const constructed = state.matches.map((match) => match.links).filter(Boolean).flat(1).map((link) => ({
				tag: "link",
				attrs: {
					...link,
					nonce
				}
			}));
			const manifest = router.ssr?.manifest;
			const assets = state.matches.map((match) => manifest?.routes[match.routeId]?.assets ?? []).filter(Boolean).flat(1).filter((asset) => asset.tag === "link").map((asset) => ({
				tag: "link",
				attrs: {
					...asset.attrs,
					suppressHydrationWarning: true,
					nonce
				}
			}));
			return [...constructed, ...assets];
		},
		structuralSharing: true
	});
	const preloadLinks = require_useRouterState.useRouterState({
		select: (state) => {
			const preloadLinks = [];
			state.matches.map((match) => router.looseRoutesById[match.routeId]).forEach((route) => router.ssr?.manifest?.routes[route.id]?.preloads?.filter(Boolean).forEach((preload) => {
				preloadLinks.push({
					tag: "link",
					attrs: {
						rel: "modulepreload",
						href: preload,
						nonce
					}
				});
			}));
			return preloadLinks;
		},
		structuralSharing: true
	});
	const styles = require_useRouterState.useRouterState({
		select: (state) => state.matches.map((match) => match.styles).flat(1).filter(Boolean).map(({ children, ...attrs }) => ({
			tag: "style",
			attrs: {
				...attrs,
				nonce
			},
			children
		})),
		structuralSharing: true
	});
	const headScripts = require_useRouterState.useRouterState({
		select: (state) => state.matches.map((match) => match.headScripts).flat(1).filter(Boolean).map(({ children, ...script }) => ({
			tag: "script",
			attrs: {
				...script,
				nonce
			},
			children
		})),
		structuralSharing: true
	});
	return uniqBy([
		...meta,
		...preloadLinks,
		...links,
		...styles,
		...headScripts
	], (d) => {
		return JSON.stringify(d);
	});
};
function uniqBy(arr, fn) {
	const seen = /* @__PURE__ */ new Set();
	return arr.filter((item) => {
		const key = fn(item);
		if (seen.has(key)) return false;
		seen.add(key);
		return true;
	});
}
//#endregion
exports.useTags = useTags;

//# sourceMappingURL=headContentUtils.cjs.map