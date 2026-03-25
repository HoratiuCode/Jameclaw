require("./_virtual/_rolldown/runtime.cjs");
const require_utils = require("./utils.cjs");
let _tanstack_router_core_isServer = require("@tanstack/router-core/isServer");
//#region src/scroll-restoration.ts
function getSafeSessionStorage() {
	try {
		if (typeof window !== "undefined" && typeof window.sessionStorage === "object") return window.sessionStorage;
	} catch {}
}
/** SessionStorage key used to persist scroll restoration state. */
/** SessionStorage key used to store scroll positions across navigations. */
/** SessionStorage key used to store scroll positions across navigations. */
var storageKey = "tsr-scroll-restoration-v1_3";
var throttle = (fn, wait) => {
	let timeout;
	return (...args) => {
		if (!timeout) timeout = setTimeout(() => {
			fn(...args);
			timeout = null;
		}, wait);
	};
};
function createScrollRestorationCache() {
	const safeSessionStorage = getSafeSessionStorage();
	if (!safeSessionStorage) return null;
	const persistedState = safeSessionStorage.getItem(storageKey);
	let state = persistedState ? JSON.parse(persistedState) : {};
	return {
		state,
		set: (updater) => {
			state = require_utils.functionalUpdate(updater, state) || state;
			try {
				safeSessionStorage.setItem(storageKey, JSON.stringify(state));
			} catch {
				console.warn("[ts-router] Could not persist scroll restoration state to sessionStorage.");
			}
		}
	};
}
/** In-memory handle to the persisted scroll restoration cache. */
var scrollRestorationCache = createScrollRestorationCache();
/**
* The default `getKey` function for `useScrollRestoration`.
* It returns the `key` from the location state or the `href` of the location.
*
* The `location.href` is used as a fallback to support the use case where the location state is not available like the initial render.
*/
/**
* Default scroll restoration cache key: location state key or full href.
*/
var defaultGetScrollRestorationKey = (location) => {
	return location.state.__TSR_key || location.href;
};
/** Best-effort nth-child CSS selector for a given element. */
function getCssSelector(el) {
	const path = [];
	let parent;
	while (parent = el.parentNode) {
		path.push(`${el.tagName}:nth-child(${Array.prototype.indexOf.call(parent.children, el) + 1})`);
		el = parent;
	}
	return `${path.reverse().join(" > ")}`.toLowerCase();
}
var ignoreScroll = false;
function restoreScroll({ storageKey, key, behavior, shouldScrollRestoration, scrollToTopSelectors, location }) {
	let byKey;
	try {
		byKey = JSON.parse(sessionStorage.getItem(storageKey) || "{}");
	} catch (error) {
		console.error(error);
		return;
	}
	const resolvedKey = key || window.history.state?.__TSR_key;
	const elementEntries = byKey[resolvedKey];
	ignoreScroll = true;
	scroll: {
		if (shouldScrollRestoration && elementEntries && Object.keys(elementEntries).length > 0) {
			for (const elementSelector in elementEntries) {
				const entry = elementEntries[elementSelector];
				if (elementSelector === "window") window.scrollTo({
					top: entry.scrollY,
					left: entry.scrollX,
					behavior
				});
				else if (elementSelector) {
					const element = document.querySelector(elementSelector);
					if (element) {
						element.scrollLeft = entry.scrollX;
						element.scrollTop = entry.scrollY;
					}
				}
			}
			break scroll;
		}
		const hash = (location ?? window.location).hash.split("#", 2)[1];
		if (hash) {
			const hashScrollIntoViewOptions = window.history.state?.__hashScrollIntoViewOptions ?? true;
			if (hashScrollIntoViewOptions) {
				const el = document.getElementById(hash);
				if (el) el.scrollIntoView(hashScrollIntoViewOptions);
			}
			break scroll;
		}
		const scrollOptions = {
			top: 0,
			left: 0,
			behavior
		};
		window.scrollTo(scrollOptions);
		if (scrollToTopSelectors) for (const selector of scrollToTopSelectors) {
			if (selector === "window") continue;
			const element = typeof selector === "function" ? selector() : document.querySelector(selector);
			if (element) element.scrollTo(scrollOptions);
		}
	}
	ignoreScroll = false;
}
/** Setup global listeners and hooks to support scroll restoration. */
/** Setup global listeners and hooks to support scroll restoration. */
function setupScrollRestoration(router, force) {
	if (!scrollRestorationCache && !(_tanstack_router_core_isServer.isServer ?? router.isServer)) return;
	if (force ?? router.options.scrollRestoration ?? false) router.isScrollRestoring = true;
	if ((_tanstack_router_core_isServer.isServer ?? router.isServer) || router.isScrollRestorationSetup || !scrollRestorationCache) return;
	router.isScrollRestorationSetup = true;
	ignoreScroll = false;
	const getKey = router.options.getScrollRestorationKey || defaultGetScrollRestorationKey;
	window.history.scrollRestoration = "manual";
	const onScroll = (event) => {
		if (ignoreScroll || !router.isScrollRestoring) return;
		let elementSelector = "";
		if (event.target === document || event.target === window) elementSelector = "window";
		else {
			const attrId = event.target.getAttribute("data-scroll-restoration-id");
			if (attrId) elementSelector = `[data-scroll-restoration-id="${attrId}"]`;
			else elementSelector = getCssSelector(event.target);
		}
		const restoreKey = getKey(router.state.location);
		scrollRestorationCache.set((state) => {
			const keyEntry = state[restoreKey] ||= {};
			const elementEntry = keyEntry[elementSelector] ||= {};
			if (elementSelector === "window") {
				elementEntry.scrollX = window.scrollX || 0;
				elementEntry.scrollY = window.scrollY || 0;
			} else if (elementSelector) {
				const element = document.querySelector(elementSelector);
				if (element) {
					elementEntry.scrollX = element.scrollLeft || 0;
					elementEntry.scrollY = element.scrollTop || 0;
				}
			}
			return state;
		});
	};
	if (typeof document !== "undefined") document.addEventListener("scroll", throttle(onScroll, 100), true);
	router.subscribe("onRendered", (event) => {
		const cacheKey = getKey(event.toLocation);
		if (!router.resetNextScroll) {
			router.resetNextScroll = true;
			return;
		}
		if (typeof router.options.scrollRestoration === "function") {
			if (!router.options.scrollRestoration({ location: router.latestLocation })) return;
		}
		restoreScroll({
			storageKey,
			key: cacheKey,
			behavior: router.options.scrollRestorationBehavior,
			shouldScrollRestoration: router.isScrollRestoring,
			scrollToTopSelectors: router.options.scrollToTopSelectors,
			location: router.history.location
		});
		if (router.isScrollRestoring) scrollRestorationCache.set((state) => {
			state[cacheKey] ||= {};
			return state;
		});
	});
}
/**
* @private
* Handles hash-based scrolling after navigation completes.
* To be used in framework-specific <Transitioner> components during the onResolved event.
*
* Provides hash scrolling for programmatic navigation when default browser handling is prevented.
* @param router The router instance containing current location and state
*/
/**
* @private
* Handles hash-based scrolling after navigation completes.
* To be used in framework-specific Transitioners.
*/
function handleHashScroll(router) {
	if (typeof document !== "undefined" && document.querySelector) {
		const hashScrollIntoViewOptions = router.state.location.state.__hashScrollIntoViewOptions ?? true;
		if (hashScrollIntoViewOptions && router.state.location.hash !== "") {
			const el = document.getElementById(router.state.location.hash);
			if (el) el.scrollIntoView(hashScrollIntoViewOptions);
		}
	}
}
//#endregion
exports.defaultGetScrollRestorationKey = defaultGetScrollRestorationKey;
exports.getCssSelector = getCssSelector;
exports.handleHashScroll = handleHashScroll;
exports.restoreScroll = restoreScroll;
exports.scrollRestorationCache = scrollRestorationCache;
exports.setupScrollRestoration = setupScrollRestoration;
exports.storageKey = storageKey;

//# sourceMappingURL=scroll-restoration.cjs.map