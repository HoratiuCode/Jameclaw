const require_runtime = require("./_virtual/_rolldown/runtime.cjs");
const require_utils = require("./utils.cjs");
const require_useRouter = require("./useRouter.cjs");
const require_useRouterState = require("./useRouterState.cjs");
let _tanstack_router_core = require("@tanstack/router-core");
let react = require("react");
react = require_runtime.__toESM(react);
//#region src/Transitioner.tsx
function Transitioner() {
	const router = require_useRouter.useRouter();
	const mountLoadForRouter = react.useRef({
		router,
		mounted: false
	});
	const [isTransitioning, setIsTransitioning] = react.useState(false);
	const { hasPendingMatches, isLoading } = require_useRouterState.useRouterState({
		select: (s) => ({
			isLoading: s.isLoading,
			hasPendingMatches: s.matches.some((d) => d.status === "pending")
		}),
		structuralSharing: true
	});
	const previousIsLoading = require_utils.usePrevious(isLoading);
	const isAnyPending = isLoading || isTransitioning || hasPendingMatches;
	const previousIsAnyPending = require_utils.usePrevious(isAnyPending);
	const isPagePending = isLoading || hasPendingMatches;
	const previousIsPagePending = require_utils.usePrevious(isPagePending);
	router.startTransition = (fn) => {
		setIsTransitioning(true);
		react.startTransition(() => {
			fn();
			setIsTransitioning(false);
		});
	};
	react.useEffect(() => {
		const unsub = router.history.subscribe(router.load);
		const nextLocation = router.buildLocation({
			to: router.latestLocation.pathname,
			search: true,
			params: true,
			hash: true,
			state: true,
			_includeValidateSearch: true
		});
		if ((0, _tanstack_router_core.trimPathRight)(router.latestLocation.publicHref) !== (0, _tanstack_router_core.trimPathRight)(nextLocation.publicHref)) router.commitLocation({
			...nextLocation,
			replace: true
		});
		return () => {
			unsub();
		};
	}, [router, router.history]);
	require_utils.useLayoutEffect(() => {
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
	require_utils.useLayoutEffect(() => {
		if (previousIsLoading && !isLoading) router.emit({
			type: "onLoad",
			...(0, _tanstack_router_core.getLocationChangeInfo)(router.state)
		});
	}, [
		previousIsLoading,
		router,
		isLoading
	]);
	require_utils.useLayoutEffect(() => {
		if (previousIsPagePending && !isPagePending) router.emit({
			type: "onBeforeRouteMount",
			...(0, _tanstack_router_core.getLocationChangeInfo)(router.state)
		});
	}, [
		isPagePending,
		previousIsPagePending,
		router
	]);
	require_utils.useLayoutEffect(() => {
		if (previousIsAnyPending && !isAnyPending) {
			const changeInfo = (0, _tanstack_router_core.getLocationChangeInfo)(router.state);
			router.emit({
				type: "onResolved",
				...changeInfo
			});
			router.__store.setState((s) => ({
				...s,
				status: "idle",
				resolvedLocation: s.location
			}));
			if (changeInfo.hrefChanged) (0, _tanstack_router_core.handleHashScroll)(router);
		}
	}, [
		isAnyPending,
		previousIsAnyPending,
		router
	]);
	return null;
}
//#endregion
exports.Transitioner = Transitioner;

//# sourceMappingURL=Transitioner.cjs.map