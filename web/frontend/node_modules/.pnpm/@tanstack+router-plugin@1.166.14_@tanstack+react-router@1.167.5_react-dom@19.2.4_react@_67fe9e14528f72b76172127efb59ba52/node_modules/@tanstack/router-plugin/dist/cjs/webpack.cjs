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
//#region src/webpack.ts
/**
* @example
* ```ts
* export default {
*   // ...
*   plugins: [TanStackRouterGeneratorWebpack()],
* }
* ```
*/
var TanStackRouterGeneratorWebpack = /* @__PURE__ */ (0, unplugin.createWebpackPlugin)(require_router_generator_plugin.unpluginRouterGeneratorFactory);
/**
* @example
* ```ts
* export default {
*   // ...
*   plugins: [TanStackRouterCodeSplitterWebpack()],
* }
* ```
*/
var TanStackRouterCodeSplitterWebpack = /* @__PURE__ */ (0, unplugin.createWebpackPlugin)(require_router_code_splitter_plugin.unpluginRouterCodeSplitterFactory);
/**
* @example
* ```ts
* export default {
*   // ...
*   plugins: [tanstackRouter()],
* }
* ```
*/
var TanStackRouterWebpack = /* @__PURE__ */ (0, unplugin.createWebpackPlugin)(require_router_composed_plugin.unpluginRouterComposedFactory);
var tanstackRouter = TanStackRouterWebpack;
//#endregion
exports.TanStackRouterCodeSplitterWebpack = TanStackRouterCodeSplitterWebpack;
exports.TanStackRouterGeneratorWebpack = TanStackRouterGeneratorWebpack;
exports.TanStackRouterWebpack = TanStackRouterWebpack;
exports.default = TanStackRouterWebpack;
exports.configSchema = require_config.configSchema;
exports.tanstackRouter = tanstackRouter;

//# sourceMappingURL=webpack.cjs.map