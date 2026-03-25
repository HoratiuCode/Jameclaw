const require_runtime = require("./_virtual/_rolldown/runtime.cjs");
const require_useRouterState = require("./useRouterState.cjs");
const require_matchContext = require("./matchContext.cjs");
let react = require("react");
react = require_runtime.__toESM(react);
let tiny_invariant = require("tiny-invariant");
tiny_invariant = require_runtime.__toESM(tiny_invariant);
//#region src/useMatch.tsx
/**
* Read and select the nearest or targeted route match.
* @link https://tanstack.com/router/latest/docs/framework/react/api/router/useMatchHook
*/
function useMatch(opts) {
	const nearestMatchId = react.useContext(opts.from ? require_matchContext.dummyMatchContext : require_matchContext.matchContext);
	return require_useRouterState.useRouterState({
		select: (state) => {
			const match = state.matches.find((d) => opts.from ? opts.from === d.routeId : d.id === nearestMatchId);
			(0, tiny_invariant.default)(!((opts.shouldThrow ?? true) && !match), `Could not find ${opts.from ? `an active match from "${opts.from}"` : "a nearest match!"}`);
			if (match === void 0) return;
			return opts.select ? opts.select(match) : match;
		},
		structuralSharing: opts.structuralSharing
	});
}
//#endregion
exports.useMatch = useMatch;

//# sourceMappingURL=useMatch.cjs.map