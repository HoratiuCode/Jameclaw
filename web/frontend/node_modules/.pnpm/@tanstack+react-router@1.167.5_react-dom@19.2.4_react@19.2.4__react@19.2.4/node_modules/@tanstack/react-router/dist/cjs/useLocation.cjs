const require_useRouterState = require("./useRouterState.cjs");
//#region src/useLocation.tsx
/**
* Read the current location from the router state with optional selection.
* Useful for subscribing to just the pieces of location you care about.
*
* Options:
* - `select`: Project the `location` object to a derived value
* - `structuralSharing`: Enable structural sharing for stable references
*
* @returns The current location (or selected value).
* @link https://tanstack.com/router/latest/docs/framework/react/api/router/useLocationHook
*/
function useLocation(opts) {
	return require_useRouterState.useRouterState({ select: (state) => opts?.select ? opts.select(state.location) : state.location });
}
//#endregion
exports.useLocation = useLocation;

//# sourceMappingURL=useLocation.cjs.map