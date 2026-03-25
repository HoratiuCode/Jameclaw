const require_runtime = require("./_virtual/_rolldown/runtime.cjs");
const require_CatchBoundary = require("./CatchBoundary.cjs");
const require_ClientOnly = require("./ClientOnly.cjs");
const require_useRouter = require("./useRouter.cjs");
const require_useRouterState = require("./useRouterState.cjs");
const require_matchContext = require("./matchContext.cjs");
const require_not_found = require("./not-found.cjs");
const require_SafeFragment = require("./SafeFragment.cjs");
const require_renderRouteNotFound = require("./renderRouteNotFound.cjs");
const require_scroll_restoration = require("./scroll-restoration.cjs");
let _tanstack_router_core = require("@tanstack/router-core");
let react = require("react");
react = require_runtime.__toESM(react);
let react_jsx_runtime = require("react/jsx-runtime");
let tiny_warning = require("tiny-warning");
tiny_warning = require_runtime.__toESM(tiny_warning);
let tiny_invariant = require("tiny-invariant");
tiny_invariant = require_runtime.__toESM(tiny_invariant);
let _tanstack_router_core_isServer = require("@tanstack/router-core/isServer");
//#region src/Match.tsx
var Match = react.memo(function MatchImpl({ matchId }) {
	const router = require_useRouter.useRouter();
	const matchState = require_useRouterState.useRouterState({
		select: (s) => {
			const matchIndex = s.matches.findIndex((d) => d.id === matchId);
			const match = s.matches[matchIndex];
			(0, tiny_invariant.default)(match, `Could not find match for matchId "${matchId}". Please file an issue!`);
			return {
				routeId: match.routeId,
				ssr: match.ssr,
				_displayPending: match._displayPending,
				resetKey: s.loadedAt,
				parentRouteId: s.matches[matchIndex - 1]?.routeId
			};
		},
		structuralSharing: true
	});
	const route = router.routesById[matchState.routeId];
	const PendingComponent = route.options.pendingComponent ?? router.options.defaultPendingComponent;
	const pendingElement = PendingComponent ? /* @__PURE__ */ (0, react_jsx_runtime.jsx)(PendingComponent, {}) : null;
	const routeErrorComponent = route.options.errorComponent ?? router.options.defaultErrorComponent;
	const routeOnCatch = route.options.onCatch ?? router.options.defaultOnCatch;
	const routeNotFoundComponent = route.isRoot ? route.options.notFoundComponent ?? router.options.notFoundRoute?.options.component : route.options.notFoundComponent;
	const resolvedNoSsr = matchState.ssr === false || matchState.ssr === "data-only";
	const ResolvedSuspenseBoundary = (!route.isRoot || route.options.wrapInSuspense || resolvedNoSsr) && (route.options.wrapInSuspense ?? PendingComponent ?? (route.options.errorComponent?.preload || resolvedNoSsr)) ? react.Suspense : require_SafeFragment.SafeFragment;
	const ResolvedCatchBoundary = routeErrorComponent ? require_CatchBoundary.CatchBoundary : require_SafeFragment.SafeFragment;
	const ResolvedNotFoundBoundary = routeNotFoundComponent ? require_not_found.CatchNotFound : require_SafeFragment.SafeFragment;
	return /* @__PURE__ */ (0, react_jsx_runtime.jsxs)(route.isRoot ? route.options.shellComponent ?? require_SafeFragment.SafeFragment : require_SafeFragment.SafeFragment, { children: [/* @__PURE__ */ (0, react_jsx_runtime.jsx)(require_matchContext.matchContext.Provider, {
		value: matchId,
		children: /* @__PURE__ */ (0, react_jsx_runtime.jsx)(ResolvedSuspenseBoundary, {
			fallback: pendingElement,
			children: /* @__PURE__ */ (0, react_jsx_runtime.jsx)(ResolvedCatchBoundary, {
				getResetKey: () => matchState.resetKey,
				errorComponent: routeErrorComponent || require_CatchBoundary.ErrorComponent,
				onCatch: (error, errorInfo) => {
					if ((0, _tanstack_router_core.isNotFound)(error)) throw error;
					(0, tiny_warning.default)(false, `Error in route match: ${matchId}`);
					routeOnCatch?.(error, errorInfo);
				},
				children: /* @__PURE__ */ (0, react_jsx_runtime.jsx)(ResolvedNotFoundBoundary, {
					fallback: (error) => {
						if (!routeNotFoundComponent || error.routeId && error.routeId !== matchState.routeId || !error.routeId && !route.isRoot) throw error;
						return react.createElement(routeNotFoundComponent, error);
					},
					children: resolvedNoSsr || matchState._displayPending ? /* @__PURE__ */ (0, react_jsx_runtime.jsx)(require_ClientOnly.ClientOnly, {
						fallback: pendingElement,
						children: /* @__PURE__ */ (0, react_jsx_runtime.jsx)(MatchInner, { matchId })
					}) : /* @__PURE__ */ (0, react_jsx_runtime.jsx)(MatchInner, { matchId })
				})
			})
		})
	}), matchState.parentRouteId === _tanstack_router_core.rootRouteId && router.options.scrollRestoration ? /* @__PURE__ */ (0, react_jsx_runtime.jsxs)(react_jsx_runtime.Fragment, { children: [/* @__PURE__ */ (0, react_jsx_runtime.jsx)(OnRendered, {}), /* @__PURE__ */ (0, react_jsx_runtime.jsx)(require_scroll_restoration.ScrollRestoration, {})] }) : null] });
});
function OnRendered() {
	const router = require_useRouter.useRouter();
	const prevLocationRef = react.useRef(void 0);
	return /* @__PURE__ */ (0, react_jsx_runtime.jsx)("script", {
		suppressHydrationWarning: true,
		ref: (el) => {
			if (el && (prevLocationRef.current === void 0 || prevLocationRef.current.href !== router.latestLocation.href)) {
				router.emit({
					type: "onRendered",
					...(0, _tanstack_router_core.getLocationChangeInfo)(router.state)
				});
				prevLocationRef.current = router.latestLocation;
			}
		}
	}, router.latestLocation.state.__TSR_key);
}
var MatchInner = react.memo(function MatchInnerImpl({ matchId }) {
	const router = require_useRouter.useRouter();
	const { match, key, routeId } = require_useRouterState.useRouterState({
		select: (s) => {
			const match = s.matches.find((d) => d.id === matchId);
			const routeId = match.routeId;
			const remountDeps = (router.routesById[routeId].options.remountDeps ?? router.options.defaultRemountDeps)?.({
				routeId,
				loaderDeps: match.loaderDeps,
				params: match._strictParams,
				search: match._strictSearch
			});
			return {
				key: remountDeps ? JSON.stringify(remountDeps) : void 0,
				routeId,
				match: {
					id: match.id,
					status: match.status,
					error: match.error,
					_forcePending: match._forcePending,
					_displayPending: match._displayPending
				}
			};
		},
		structuralSharing: true
	});
	const route = router.routesById[routeId];
	const out = react.useMemo(() => {
		const Comp = route.options.component ?? router.options.defaultComponent;
		if (Comp) return /* @__PURE__ */ (0, react_jsx_runtime.jsx)(Comp, {}, key);
		return /* @__PURE__ */ (0, react_jsx_runtime.jsx)(Outlet, {});
	}, [
		key,
		route.options.component,
		router.options.defaultComponent
	]);
	if (match._displayPending) throw router.getMatch(match.id)?._nonReactive.displayPendingPromise;
	if (match._forcePending) throw router.getMatch(match.id)?._nonReactive.minPendingPromise;
	if (match.status === "pending") {
		const pendingMinMs = route.options.pendingMinMs ?? router.options.defaultPendingMinMs;
		if (pendingMinMs) {
			const routerMatch = router.getMatch(match.id);
			if (routerMatch && !routerMatch._nonReactive.minPendingPromise) {
				if (!(_tanstack_router_core_isServer.isServer ?? router.isServer)) {
					const minPendingPromise = (0, _tanstack_router_core.createControlledPromise)();
					routerMatch._nonReactive.minPendingPromise = minPendingPromise;
					setTimeout(() => {
						minPendingPromise.resolve();
						routerMatch._nonReactive.minPendingPromise = void 0;
					}, pendingMinMs);
				}
			}
		}
		throw router.getMatch(match.id)?._nonReactive.loadPromise;
	}
	if (match.status === "notFound") {
		(0, tiny_invariant.default)((0, _tanstack_router_core.isNotFound)(match.error), "Expected a notFound error");
		return require_renderRouteNotFound.renderRouteNotFound(router, route, match.error);
	}
	if (match.status === "redirected") {
		(0, tiny_invariant.default)((0, _tanstack_router_core.isRedirect)(match.error), "Expected a redirect error");
		throw router.getMatch(match.id)?._nonReactive.loadPromise;
	}
	if (match.status === "error") {
		if (_tanstack_router_core_isServer.isServer ?? router.isServer) return /* @__PURE__ */ (0, react_jsx_runtime.jsx)((route.options.errorComponent ?? router.options.defaultErrorComponent) || require_CatchBoundary.ErrorComponent, {
			error: match.error,
			reset: void 0,
			info: { componentStack: "" }
		});
		throw match.error;
	}
	return out;
});
/**
* Render the next child match in the route tree. Typically used inside
* a route component to render nested routes.
*
* @link https://tanstack.com/router/latest/docs/framework/react/api/router/outletComponent
*/
var Outlet = react.memo(function OutletImpl() {
	const router = require_useRouter.useRouter();
	const matchId = react.useContext(require_matchContext.matchContext);
	const routeId = require_useRouterState.useRouterState({ select: (s) => s.matches.find((d) => d.id === matchId)?.routeId });
	const route = router.routesById[routeId];
	const parentGlobalNotFound = require_useRouterState.useRouterState({ select: (s) => {
		const parentMatch = s.matches.find((d) => d.id === matchId);
		(0, tiny_invariant.default)(parentMatch, `Could not find parent match for matchId "${matchId}"`);
		return parentMatch.globalNotFound;
	} });
	const childMatchId = require_useRouterState.useRouterState({ select: (s) => {
		const matches = s.matches;
		return matches[matches.findIndex((d) => d.id === matchId) + 1]?.id;
	} });
	const pendingElement = router.options.defaultPendingComponent ? /* @__PURE__ */ (0, react_jsx_runtime.jsx)(router.options.defaultPendingComponent, {}) : null;
	if (parentGlobalNotFound) return require_renderRouteNotFound.renderRouteNotFound(router, route, void 0);
	if (!childMatchId) return null;
	const nextMatch = /* @__PURE__ */ (0, react_jsx_runtime.jsx)(Match, { matchId: childMatchId });
	if (routeId === _tanstack_router_core.rootRouteId) return /* @__PURE__ */ (0, react_jsx_runtime.jsx)(react.Suspense, {
		fallback: pendingElement,
		children: nextMatch
	});
	return nextMatch;
});
//#endregion
exports.Match = Match;
exports.Outlet = Outlet;

//# sourceMappingURL=Match.cjs.map