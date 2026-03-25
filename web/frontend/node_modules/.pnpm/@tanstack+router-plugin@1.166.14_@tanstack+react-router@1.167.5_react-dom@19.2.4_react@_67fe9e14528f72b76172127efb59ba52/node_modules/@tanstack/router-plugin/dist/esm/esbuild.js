import { configSchema } from "./core/config.js";
import { unpluginRouterCodeSplitterFactory } from "./core/router-code-splitter-plugin.js";
import { unpluginRouterGeneratorFactory } from "./core/router-generator-plugin.js";
import { unpluginRouterComposedFactory } from "./core/router-composed-plugin.js";
import { createEsbuildPlugin } from "unplugin";
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
var TanStackRouterGeneratorEsbuild = createEsbuildPlugin(unpluginRouterGeneratorFactory);
/**
* @example
* ```ts
* export default {
*  plugins: [TanStackRouterCodeSplitterEsbuild()],
*  // ...
* }
* ```
*/
var TanStackRouterCodeSplitterEsbuild = createEsbuildPlugin(unpluginRouterCodeSplitterFactory);
/**
* @example
* ```ts
* export default {
*   plugins: [tanstackRouter()],
*   // ...
* }
* ```
*/
var TanStackRouterEsbuild = createEsbuildPlugin(unpluginRouterComposedFactory);
var tanstackRouter = TanStackRouterEsbuild;
//#endregion
export { TanStackRouterCodeSplitterEsbuild, TanStackRouterEsbuild, TanStackRouterEsbuild as default, TanStackRouterGeneratorEsbuild, configSchema, tanstackRouter };

//# sourceMappingURL=esbuild.js.map