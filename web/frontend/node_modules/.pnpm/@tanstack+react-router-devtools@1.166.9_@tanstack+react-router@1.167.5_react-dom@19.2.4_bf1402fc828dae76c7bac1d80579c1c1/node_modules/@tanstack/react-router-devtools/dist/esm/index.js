import { TanStackRouterDevtools as TanStackRouterDevtools$1 } from "./TanStackRouterDevtools.js";
import { TanStackRouterDevtoolsPanel as TanStackRouterDevtoolsPanel$1 } from "./TanStackRouterDevtoolsPanel.js";
//#region src/index.ts
var TanStackRouterDevtools = process.env.NODE_ENV !== "development" ? function() {
	return null;
} : TanStackRouterDevtools$1;
var TanStackRouterDevtoolsInProd = TanStackRouterDevtools$1;
var TanStackRouterDevtoolsPanel = process.env.NODE_ENV !== "development" ? function() {
	return null;
} : TanStackRouterDevtoolsPanel$1;
var TanStackRouterDevtoolsPanelInProd = TanStackRouterDevtoolsPanel$1;
//#endregion
export { TanStackRouterDevtools, TanStackRouterDevtoolsInProd, TanStackRouterDevtoolsPanel, TanStackRouterDevtoolsPanelInProd };

//# sourceMappingURL=index.js.map