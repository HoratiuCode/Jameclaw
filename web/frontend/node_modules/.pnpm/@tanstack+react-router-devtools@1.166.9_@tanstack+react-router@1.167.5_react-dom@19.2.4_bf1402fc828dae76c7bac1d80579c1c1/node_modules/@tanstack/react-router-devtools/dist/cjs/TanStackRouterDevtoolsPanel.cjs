const require_runtime = require("./_virtual/_rolldown/runtime.cjs");
let _tanstack_router_devtools_core = require("@tanstack/router-devtools-core");
let react = require("react");
react = require_runtime.__toESM(react);
let _tanstack_react_router = require("@tanstack/react-router");
let react_jsx_runtime = require("react/jsx-runtime");
//#region src/TanStackRouterDevtoolsPanel.tsx
var TanStackRouterDevtoolsPanel = (props) => {
	const { router: propsRouter, ...rest } = props;
	const hookRouter = (0, _tanstack_react_router.useRouter)({ warn: false });
	const activeRouter = propsRouter ?? hookRouter;
	const activeRouterState = (0, _tanstack_react_router.useRouterState)({ router: activeRouter });
	const devToolRef = (0, react.useRef)(null);
	const [devtools] = (0, react.useState)(() => new _tanstack_router_devtools_core.TanStackRouterDevtoolsPanelCore({
		...rest,
		router: activeRouter,
		routerState: activeRouterState
	}));
	(0, react.useEffect)(() => {
		devtools.setRouter(activeRouter);
	}, [devtools, activeRouter]);
	(0, react.useEffect)(() => {
		devtools.setRouterState(activeRouterState);
	}, [devtools, activeRouterState]);
	(0, react.useEffect)(() => {
		devtools.setOptions({
			className: props.className,
			style: props.style,
			shadowDOMTarget: props.shadowDOMTarget
		});
	}, [
		devtools,
		props.className,
		props.style,
		props.shadowDOMTarget
	]);
	(0, react.useEffect)(() => {
		if (devToolRef.current) devtools.mount(devToolRef.current);
		return () => {
			devtools.unmount();
		};
	}, [devtools]);
	return /* @__PURE__ */ (0, react_jsx_runtime.jsx)(react_jsx_runtime.Fragment, { children: /* @__PURE__ */ (0, react_jsx_runtime.jsx)("div", { ref: devToolRef }) });
};
//#endregion
exports.TanStackRouterDevtoolsPanel = TanStackRouterDevtoolsPanel;

//# sourceMappingURL=TanStackRouterDevtoolsPanel.cjs.map