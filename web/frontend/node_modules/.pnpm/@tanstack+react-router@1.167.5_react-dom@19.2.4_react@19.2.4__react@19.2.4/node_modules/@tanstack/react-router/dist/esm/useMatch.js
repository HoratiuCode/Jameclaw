import { useRouterState } from "./useRouterState.js";
import { dummyMatchContext, matchContext } from "./matchContext.js";
import * as React$1 from "react";
import invariant from "tiny-invariant";
//#region src/useMatch.tsx
/**
* Read and select the nearest or targeted route match.
* @link https://tanstack.com/router/latest/docs/framework/react/api/router/useMatchHook
*/
function useMatch(opts) {
	const nearestMatchId = React$1.useContext(opts.from ? dummyMatchContext : matchContext);
	return useRouterState({
		select: (state) => {
			const match = state.matches.find((d) => opts.from ? opts.from === d.routeId : d.id === nearestMatchId);
			invariant(!((opts.shouldThrow ?? true) && !match), `Could not find ${opts.from ? `an active match from "${opts.from}"` : "a nearest match!"}`);
			if (match === void 0) return;
			return opts.select ? opts.select(match) : match;
		},
		structuralSharing: opts.structuralSharing
	});
}
//#endregion
export { useMatch };

//# sourceMappingURL=useMatch.js.map