require("./_virtual/_rolldown/runtime.cjs");
const require_useRouter = require("./useRouter.cjs");
const require_useRouterState = require("./useRouterState.cjs");
const require_Asset = require("./Asset.cjs");
let react = require("react");
let react_jsx_runtime = require("react/jsx-runtime");
//#region src/Scripts.tsx
/**
* Render body script tags collected from route matches and SSR manifests.
* Should be placed near the end of the document body.
*/
var Scripts = () => {
	const router = require_useRouter.useRouter();
	const nonce = router.options.ssr?.nonce;
	const assetScripts = require_useRouterState.useRouterState({
		select: (state) => {
			const assetScripts = [];
			const manifest = router.ssr?.manifest;
			if (!manifest) return [];
			state.matches.map((match) => router.looseRoutesById[match.routeId]).forEach((route) => manifest.routes[route.id]?.assets?.filter((d) => d.tag === "script").forEach((asset) => {
				assetScripts.push({
					tag: "script",
					attrs: {
						...asset.attrs,
						nonce
					},
					children: asset.children
				});
			}));
			return assetScripts;
		},
		structuralSharing: true
	});
	const { scripts } = require_useRouterState.useRouterState({
		select: (state) => ({ scripts: state.matches.map((match) => match.scripts).flat(1).filter(Boolean).map(({ children, ...script }) => ({
			tag: "script",
			attrs: {
				...script,
				suppressHydrationWarning: true,
				nonce
			},
			children
		})) }),
		structuralSharing: true
	});
	let serverBufferedScript = void 0;
	if (router.serverSsr) serverBufferedScript = router.serverSsr.takeBufferedScripts();
	const allScripts = [...scripts, ...assetScripts];
	if (serverBufferedScript) allScripts.unshift(serverBufferedScript);
	return /* @__PURE__ */ (0, react_jsx_runtime.jsx)(react_jsx_runtime.Fragment, { children: allScripts.map((asset, i) => /* @__PURE__ */ (0, react.createElement)(require_Asset.Asset, {
		...asset,
		key: `tsr-scripts-${asset.tag}-${i}`
	})) });
};
//#endregion
exports.Scripts = Scripts;

//# sourceMappingURL=Scripts.cjs.map