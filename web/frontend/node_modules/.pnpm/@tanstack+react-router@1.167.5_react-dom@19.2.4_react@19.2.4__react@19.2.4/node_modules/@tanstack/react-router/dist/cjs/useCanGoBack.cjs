const require_useRouterState = require("./useRouterState.cjs");
//#region src/useCanGoBack.ts
function useCanGoBack() {
	return require_useRouterState.useRouterState({ select: (s) => s.location.state.__TSR_index !== 0 });
}
//#endregion
exports.useCanGoBack = useCanGoBack;

//# sourceMappingURL=useCanGoBack.cjs.map