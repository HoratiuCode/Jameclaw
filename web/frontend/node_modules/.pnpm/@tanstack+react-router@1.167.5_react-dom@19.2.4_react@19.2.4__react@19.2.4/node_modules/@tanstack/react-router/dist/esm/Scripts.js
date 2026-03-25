import { useRouter } from "./useRouter.js";
import { useRouterState } from "./useRouterState.js";
import { Asset } from "./Asset.js";
import { createElement } from "react";
import { Fragment, jsx } from "react/jsx-runtime";
//#region src/Scripts.tsx
/**
* Render body script tags collected from route matches and SSR manifests.
* Should be placed near the end of the document body.
*/
var Scripts = () => {
	const router = useRouter();
	const nonce = router.options.ssr?.nonce;
	const assetScripts = useRouterState({
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
	const { scripts } = useRouterState({
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
	return /* @__PURE__ */ jsx(Fragment, { children: allScripts.map((asset, i) => /* @__PURE__ */ createElement(Asset, {
		...asset,
		key: `tsr-scripts-${asset.tag}-${i}`
	})) });
};
//#endregion
export { Scripts };

//# sourceMappingURL=Scripts.js.map