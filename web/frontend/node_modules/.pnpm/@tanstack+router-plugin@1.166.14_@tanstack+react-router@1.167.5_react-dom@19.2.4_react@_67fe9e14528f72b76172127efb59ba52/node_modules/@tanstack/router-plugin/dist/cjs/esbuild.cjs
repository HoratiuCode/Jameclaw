Object.defineProperties(exports, {
	__esModule: { value: true },
	[Symbol.toStringTag]: { value: "Module" }
});
require("./_virtual/_rolldown/runtime.cjs");
const require_config = require("./core/config.cjs");
const require_router_code_splitter_plugin = require("./core/router-code-splitter-plugin.cjs");
const require_router_generator_plugin = require("./core/router-generator-plugin.cjs");
const require_router_composed_plugin = require("./core/router-composed-plugin.cjs");
let unplugin = require("unplugin");
//#region src/esbuild.ts
/**
* @example
* ```ts
* export default {
*   plugins: [TanStackRouterGeneratorEsbuild()],
*   // ...
* }
* ```
*/
var TanStackRouterGeneratorEsbuild = (0, unplugin.createEsbuildPlugin)(require_router_generator_plugin.unpluginRouterGeneratorFactory);
/**
* @example
* ```ts
* export default {
*  plugins: [TanStackRouterCodeSplitterEsbuild()],
*  // ...
* }
* ```
*/
var TanStackRouterCodeSplitterEsbuild = (0, unplugin.createEsbuildPlugin)(require_router_code_splitter_plugin.unpluginRouterCodeSplitterFactory);
/**
* @example
* ```ts
* export default {
*   plugins: [tanstackRouter()],
*   // ...
* }
* ```
*/
var TanStackRouterEsbuild = (0, unplugin.createEsbuildPlugin)(require_router_composed_plugin.unpluginRouterComposedFactory);
var tanstackRouter = TanStackRouterEsbuild;
//#endregion
exports.TanStackRouterCodeSplitterEsbuild = TanStackRouterCodeSplitterEsbuild;
exports.TanStackRouterEsbuild = TanStackRouterEsbuild;
exports.default = TanStackRouterEsbuild;
exports.TanStackRouterGeneratorEsbuild = TanStackRouterGeneratorEsbuild;
exports.configSchema = require_config.configSchema;
exports.tanstackRouter = tanstackRouter;

//# sourceMappingURL=esbuild.cjs.map