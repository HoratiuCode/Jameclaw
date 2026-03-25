import { unpluginRouterCodeSplitterFactory } from "./router-code-splitter-plugin.js";
import { unpluginRouterGeneratorFactory } from "./router-generator-plugin.js";
import { unpluginRouterHmrFactory } from "./router-hmr-plugin.js";
import { unpluginRouteAutoImportFactory } from "./route-autoimport-plugin.js";
import { getConfig } from "@tanstack/router-generator";
//#region src/core/router-composed-plugin.ts
var unpluginRouterComposedFactory = (options = {}, meta) => {
	const userConfig = getConfig(options, process.cwd());
	const getPlugin = (pluginFactory) => {
		const plugin = pluginFactory(options, meta);
		if (!Array.isArray(plugin)) return [plugin];
		return plugin;
	};
	const routerGenerator = getPlugin(unpluginRouterGeneratorFactory);
	const routerCodeSplitter = getPlugin(unpluginRouterCodeSplitterFactory);
	const routeAutoImport = getPlugin(unpluginRouteAutoImportFactory);
	const result = [...routerGenerator];
	if (userConfig.autoCodeSplitting) result.push(...routerCodeSplitter);
	if (userConfig.verboseFileRoutes === false) result.push(...routeAutoImport);
	if (!(process.env.NODE_ENV === "production") && !userConfig.autoCodeSplitting) {
		const routerHmr = getPlugin(unpluginRouterHmrFactory);
		result.push(...routerHmr);
	}
	return result;
};
//#endregion
export { unpluginRouterComposedFactory };

//# sourceMappingURL=router-composed-plugin.js.map