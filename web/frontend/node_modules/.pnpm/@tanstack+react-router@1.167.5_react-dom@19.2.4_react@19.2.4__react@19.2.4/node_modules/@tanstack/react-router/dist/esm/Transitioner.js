import { useLayoutEffect, usePrevious } from "./utils.js";
import { useRouter } from "./useRouter.js";
import { useRouterState } from "./useRouterState.js";
import { getLocationChangeInfo, handleHashScroll, trimPathRight } from "@tanstack/router-core";
import * as React$1 from "react";
//#region src/Transitioner.tsx
function Transitioner() {
	const router = useRouter();
	const mountLoadForRouter = React$1.useRef({
		router,
		mounted: false
	});
	const [isTransitioning, setIsTransitioning] = React$1.useState(false);
	const { hasPendingMatches, isLoading } = useRouterState({
		select: (s) => ({
			isLoading: s.isLoading,
			hasPendingMatches: s.matches.some((d) => d.status === "pending")
		}),
		structuralSharing: true
	});
	const previousIsLoading = usePrevious(isLoading);
	const isAnyPending = isLoading || isTransitioning || hasPendingMatches;
	const previousIsAnyPending = usePrevious(isAnyPending);
	const isPagePending = isLoading || hasPendingMatches;
	const previousIsPagePending = usePrevious(isPagePending);
	router.startTransition = (fn) => {
		setIsTransitioning(true);
		React$1.startTransition(() => {
			fn();
			setIsTransitioning(false);
		});
	};
	React$1.useEffect(() => {
		const unsub = router.history.subscribe(router.load);
		const nextLocation = router.buildLocation({
			to: router.latestLocation.pathname,
			search: true,
			params: true,
			hash: true,
			state: true,
			_includeValidateSearch: true
		});
		if (trimPathRight(router.latestLocation.publicHref) !== trimPathRight(nextLocation.publicHref)) router.commitLocation({
			...nextLocation,
			replace: true
		});
		return () => {
			unsub();
		};
	}, [router, router.history]);
	useLayoutEffect(() => {
		if (typeof window !== "undefined" && router.ssr || mountLoadForRouter.current.router === router && mountLoadForRouter.current.mounted) return;
		mountLoadForRouter.current = {
			router,
			mounted: true
		};
		const tryLoad = async () => {
			try {
				await router.load();
			} catch (err) {
				console.error(err);
			}
		};
		tryLoad();
	}, [router]);
	useLayoutEffect(() => {
		if (previousIsLoading && !isLoading) router.emit({
			type: "onLoad",
			...getLocationChangeInfo(router.state)
		});
	}, [
		previousIsLoading,
		router,
		isLoading
	]);
	useLayoutEffect(() => {
		if (previousIsPagePending && !isPagePending) router.emit({
			type: "onBeforeRouteMount",
			...getLocationChangeInfo(router.state)
		});
	}, [
		isPagePending,
		previousIsPagePending,
		router
	]);
	useLayoutEffect(() => {
		if (previousIsAnyPending && !isAnyPending) {
			const changeInfo = getLocationChangeInfo(router.state);
			router.emit({
				type: "onResolved",
				...changeInfo
			});
			router.__store.setState((s) => ({
				...s,
				status: "idle",
				resolvedLocation: s.location
			}));
			if (changeInfo.hrefChanged) handleHashScroll(router);
		}
	}, [
		isAnyPending,
		previousIsAnyPending,
		router
	]);
	return null;
}
//#endregion
export { Transitioner };

//# sourceMappingURL=Transitioner.js.map