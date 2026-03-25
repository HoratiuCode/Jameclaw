import { CatchBoundary, ErrorComponent } from "./CatchBoundary.js";
import { ClientOnly } from "./ClientOnly.js";
import { useRouter } from "./useRouter.js";
import { useRouterState } from "./useRouterState.js";
import { matchContext } from "./matchContext.js";
import { CatchNotFound } from "./not-found.js";
import { SafeFragment } from "./SafeFragment.js";
import { renderRouteNotFound } from "./renderRouteNotFound.js";
import { ScrollRestoration } from "./scroll-restoration.js";
import { createControlledPromise, getLocationChangeInfo, isNotFound, isRedirect, rootRouteId } from "@tanstack/router-core";
import * as React$1 from "react";
import { Fragment, jsx, jsxs } from "react/jsx-runtime";
import warning from "tiny-warning";
import invariant from "tiny-invariant";
import { isServer } from "@tanstack/router-core/isServer";
//#region src/Match.tsx
var Match = React$1.memo(function MatchImpl({ matchId }) {
	const router = useRouter();
	const matchState = useRouterState({
		select: (s) => {
			const matchIndex = s.matches.findIndex((d) => d.id === matchId);
			const match = s.matches[matchIndex];
			invariant(match, `Could not find match for matchId "${matchId}". Please file an issue!`);
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
	const pendingElement = PendingComponent ? /* @__PURE__ */ jsx(PendingComponent, {}) : null;
	const routeErrorComponent = route.options.errorComponent ?? router.options.defaultErrorComponent;
	const routeOnCatch = route.options.onCatch ?? router.options.defaultOnCatch;
	const routeNotFoundComponent = route.isRoot ? route.options.notFoundComponent ?? router.options.notFoundRoute?.options.component : route.options.notFoundComponent;
	const resolvedNoSsr = matchState.ssr === false || matchState.ssr === "data-only";
	const ResolvedSuspenseBoundary = (!route.isRoot || route.options.wrapInSuspense || resolvedNoSsr) && (route.options.wrapInSuspense ?? PendingComponent ?? (route.options.errorComponent?.preload || resolvedNoSsr)) ? React$1.Suspense : SafeFragment;
	const ResolvedCatchBoundary = routeErrorComponent ? CatchBoundary : SafeFragment;
	const ResolvedNotFoundBoundary = routeNotFoundComponent ? CatchNotFound : SafeFragment;
	return /* @__PURE__ */ jsxs(route.isRoot ? route.options.shellComponent ?? SafeFragment : SafeFragment, { children: [/* @__PURE__ */ jsx(matchContext.Provider, {
		value: matchId,
		children: /* @__PURE__ */ jsx(ResolvedSuspenseBoundary, {
			fallback: pendingElement,
			children: /* @__PURE__ */ jsx(ResolvedCatchBoundary, {
				getResetKey: () => matchState.resetKey,
				errorComponent: routeErrorComponent || ErrorComponent,
				onCatch: (error, errorInfo) => {
					if (isNotFound(error)) throw error;
					warning(false, `Error in route match: ${matchId}`);
					routeOnCatch?.(error, errorInfo);
				},
				children: /* @__PURE__ */ jsx(ResolvedNotFoundBoundary, {
					fallback: (error) => {
						if (!routeNotFoundComponent || error.routeId && error.routeId !== matchState.routeId || !error.routeId && !route.isRoot) throw error;
						return React$1.createElement(routeNotFoundComponent, error);
					},
					children: resolvedNoSsr || matchState._displayPending ? /* @__PURE__ */ jsx(ClientOnly, {
						fallback: pendingElement,
						children: /* @__PURE__ */ jsx(MatchInner, { matchId })
					}) : /* @__PURE__ */ jsx(MatchInner, { matchId })
				})
			})
		})
	}), matchState.parentRouteId === rootRouteId && router.options.scrollRestoration ? /* @__PURE__ */ jsxs(Fragment, { children: [/* @__PURE__ */ jsx(OnRendered, {}), /* @__PURE__ */ jsx(ScrollRestoration, {})] }) : null] });
});
function OnRendered() {
	const router = useRouter();
	const prevLocationRef = React$1.useRef(void 0);
	return /* @__PURE__ */ jsx("script", {
		suppressHydrationWarning: true,
		ref: (el) => {
			if (el && (prevLocationRef.current === void 0 || prevLocationRef.current.href !== router.latestLocation.href)) {
				router.emit({
					type: "onRendered",
					...getLocationChangeInfo(router.state)
				});
				prevLocationRef.current = router.latestLocation;
			}
		}
	}, router.latestLocation.state.__TSR_key);
}
var MatchInner = React$1.memo(function MatchInnerImpl({ matchId }) {
	const router = useRouter();
	const { match, key, routeId } = useRouterState({
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
	const out = React$1.useMemo(() => {
		const Comp = route.options.component ?? router.options.defaultComponent;
		if (Comp) return /* @__PURE__ */ jsx(Comp, {}, key);
		return /* @__PURE__ */ jsx(Outlet, {});
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
				if (!(isServer ?? router.isServer)) {
					const minPendingPromise = createControlledPromise();
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
		invariant(isNotFound(match.error), "Expected a notFound error");
		return renderRouteNotFound(router, route, match.error);
	}
	if (match.status === "redirected") {
		invariant(isRedirect(match.error), "Expected a redirect error");
		throw router.getMatch(match.id)?._nonReactive.loadPromise;
	}
	if (match.status === "error") {
		if (isServer ?? router.isServer) return /* @__PURE__ */ jsx((route.options.errorComponent ?? router.options.defaultErrorComponent) || ErrorComponent, {
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
var Outlet = React$1.memo(function OutletImpl() {
	const router = useRouter();
	const matchId = React$1.useContext(matchContext);
	const routeId = useRouterState({ select: (s) => s.matches.find((d) => d.id === matchId)?.routeId });
	const route = router.routesById[routeId];
	const parentGlobalNotFound = useRouterState({ select: (s) => {
		const parentMatch = s.matches.find((d) => d.id === matchId);
		invariant(parentMatch, `Could not find parent match for matchId "${matchId}"`);
		return parentMatch.globalNotFound;
	} });
	const childMatchId = useRouterState({ select: (s) => {
		const matches = s.matches;
		return matches[matches.findIndex((d) => d.id === matchId) + 1]?.id;
	} });
	const pendingElement = router.options.defaultPendingComponent ? /* @__PURE__ */ jsx(router.options.defaultPendingComponent, {}) : null;
	if (parentGlobalNotFound) return renderRouteNotFound(router, route, void 0);
	if (!childMatchId) return null;
	const nextMatch = /* @__PURE__ */ jsx(Match, { matchId: childMatchId });
	if (routeId === rootRouteId) return /* @__PURE__ */ jsx(React$1.Suspense, {
		fallback: pendingElement,
		children: nextMatch
	});
	return nextMatch;
});
//#endregion
export { Match, Outlet };

//# sourceMappingURL=Match.js.map