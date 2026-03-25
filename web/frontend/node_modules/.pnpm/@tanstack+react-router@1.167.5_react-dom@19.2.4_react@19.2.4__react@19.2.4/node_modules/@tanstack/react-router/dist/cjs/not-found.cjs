const require_runtime = require("./_virtual/_rolldown/runtime.cjs");
const require_CatchBoundary = require("./CatchBoundary.cjs");
const require_useRouterState = require("./useRouterState.cjs");
let _tanstack_router_core = require("@tanstack/router-core");
let react = require("react");
react = require_runtime.__toESM(react);
let react_jsx_runtime = require("react/jsx-runtime");
//#region src/not-found.tsx
function CatchNotFound(props) {
	const resetKey = require_useRouterState.useRouterState({ select: (s) => `not-found-${s.location.pathname}-${s.status}` });
	return /* @__PURE__ */ (0, react_jsx_runtime.jsx)(require_CatchBoundary.CatchBoundary, {
		getResetKey: () => resetKey,
		onCatch: (error, errorInfo) => {
			if ((0, _tanstack_router_core.isNotFound)(error)) props.onCatch?.(error, errorInfo);
			else throw error;
		},
		errorComponent: ({ error }) => {
			if ((0, _tanstack_router_core.isNotFound)(error)) return props.fallback?.(error);
			else throw error;
		},
		children: props.children
	});
}
function DefaultGlobalNotFound() {
	return /* @__PURE__ */ (0, react_jsx_runtime.jsx)("p", { children: "Not Found" });
}
//#endregion
exports.CatchNotFound = CatchNotFound;
exports.DefaultGlobalNotFound = DefaultGlobalNotFound;

//# sourceMappingURL=not-found.cjs.map