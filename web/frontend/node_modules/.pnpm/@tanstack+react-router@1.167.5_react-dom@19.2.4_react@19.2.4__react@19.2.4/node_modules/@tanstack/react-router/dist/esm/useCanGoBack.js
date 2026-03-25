import { useRouterState } from "./useRouterState.js";
//#region src/useCanGoBack.ts
function useCanGoBack() {
	return useRouterState({ select: (s) => s.location.state.__TSR_index !== 0 });
}
//#endregion
export { useCanGoBack };

//# sourceMappingURL=useCanGoBack.js.map