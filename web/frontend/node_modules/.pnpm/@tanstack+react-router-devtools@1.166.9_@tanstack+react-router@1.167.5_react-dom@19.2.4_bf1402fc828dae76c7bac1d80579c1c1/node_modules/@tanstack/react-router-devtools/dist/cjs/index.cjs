Object.defineProperty(exports, Symbol.toStringTag, { value: "Module" });
const require_TanStackRouterDevtools = require("./TanStackRouterDevtools.cjs");
const require_TanStackRouterDevtoolsPanel = require("./TanStackRouterDevtoolsPanel.cjs");
//#region src/index.ts
var TanStackRouterDevtools = process.env.NODE_ENV !== "development" ? function() {
	return null;
} : require_TanStackRouterDevtools.TanStackRouterDevtools;
var TanStackRouterDevtoolsInProd = require_TanStackRouterDevtools.TanStackRouterDevtools;
var TanStackRouterDevtoolsPanel = process.env.NODE_ENV !== "development" ? function() {
	return null;
} : require_TanStackRouterDevtoolsPanel.TanStackRouterDevtoolsPanel;
var TanStackRouterDevtoolsPanelInProd = require_TanStackRouterDevtoolsPanel.TanStackRouterDevtoolsPanel;
//#endregion
exports.TanStackRouterDevtools = TanStackRouterDevtools;
exports.TanStackRouterDevtoolsInProd = TanStackRouterDevtoolsInProd;
exports.TanStackRouterDevtoolsPanel = TanStackRouterDevtoolsPanel;
exports.TanStackRouterDevtoolsPanelInProd = TanStackRouterDevtoolsPanelInProd;

//# sourceMappingURL=index.cjs.map