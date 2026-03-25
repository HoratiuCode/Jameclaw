import { TanStackRouterDevtoolsPanelCore } from "@tanstack/router-devtools-core";
import { useEffect, useRef, useState } from "react";
import { useRouter, useRouterState } from "@tanstack/react-router";
import { Fragment as Fragment$1, jsx } from "react/jsx-runtime";
//#region src/TanStackRouterDevtoolsPanel.tsx
var TanStackRouterDevtoolsPanel = (props) => {
	const { router: propsRouter, ...rest } = props;
	const hookRouter = useRouter({ warn: false });
	const activeRouter = propsRouter ?? hookRouter;
	const activeRouterState = useRouterState({ router: activeRouter });
	const devToolRef = useRef(null);
	const [devtools] = useState(() => new TanStackRouterDevtoolsPanelCore({
		...rest,
		router: activeRouter,
		routerState: activeRouterState
	}));
	useEffect(() => {
		devtools.setRouter(activeRouter);
	}, [devtools, activeRouter]);
	useEffect(() => {
		devtools.setRouterState(activeRouterState);
	}, [devtools, activeRouterState]);
	useEffect(() => {
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
	useEffect(() => {
		if (devToolRef.current) devtools.mount(devToolRef.current);
		return () => {
			devtools.unmount();
		};
	}, [devtools]);
	return /* @__PURE__ */ jsx(Fragment$1, { children: /* @__PURE__ */ jsx("div", { ref: devToolRef }) });
};
//#endregion
export { TanStackRouterDevtoolsPanel };

//# sourceMappingURL=TanStackRouterDevtoolsPanel.js.map