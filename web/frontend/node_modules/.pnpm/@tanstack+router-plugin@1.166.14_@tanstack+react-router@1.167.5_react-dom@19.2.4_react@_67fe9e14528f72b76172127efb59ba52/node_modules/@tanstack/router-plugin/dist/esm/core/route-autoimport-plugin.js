import { getConfig } from "./config.js";
import { debug, normalizePath } from "./utils.js";
import { generateFromAst, logDiff, parseAst } from "@tanstack/router-utils";
import * as babel from "@babel/core";
import * as template from "@babel/template";
//#region src/core/route-autoimport-plugin.ts
/**
* This plugin adds imports for createFileRoute and createLazyFileRoute to the file route.
*/
var unpluginRouteAutoImportFactory = (options = {}) => {
	let ROOT = process.cwd();
	let userConfig;
	function initUserConfig() {
		if (typeof options === "function") userConfig = options();
		else userConfig = getConfig(options, ROOT);
	}
	return {
		name: "tanstack-router:autoimport",
		enforce: "pre",
		transform: {
			filter: {
				id: /\.(m|c)?(j|t)sx?$/,
				code: /createFileRoute\(|createLazyFileRoute\(/
			},
			handler(code, id) {
				const normalizedId = normalizePath(id);
				if (!globalThis.TSR_ROUTES_BY_ID_MAP?.has(normalizedId)) return null;
				let routeType;
				if (code.includes("createFileRoute(")) routeType = "createFileRoute";
				else if (code.includes("createLazyFileRoute(")) routeType = "createLazyFileRoute";
				else return null;
				const routerImportPath = `@tanstack/${userConfig.target}-router`;
				const ast = parseAst({ code });
				let isCreateRouteFunctionImported = false;
				babel.traverse(ast, { Program: { enter(programPath) {
					programPath.traverse({ ImportDeclaration(path) {
						if (path.node.specifiers.map((specifier) => specifier.local.name).includes(routeType) && path.node.source.value === routerImportPath) isCreateRouteFunctionImported = true;
					} });
				} } });
				if (!isCreateRouteFunctionImported) {
					if (debug) console.info("Adding autoimports to route ", normalizedId);
					const autoImportStatement = template.statement(`import { ${routeType} } from '${routerImportPath}'`)();
					ast.program.body.unshift(autoImportStatement);
					const result = generateFromAst(ast, {
						sourceMaps: true,
						filename: normalizedId,
						sourceFileName: normalizedId
					});
					if (debug) {
						logDiff(code, result.code);
						console.log("Output:\n", result.code + "\n\n");
					}
					return result;
				}
				return null;
			}
		},
		vite: {
			configResolved(config) {
				ROOT = config.root;
				initUserConfig();
			},
			applyToEnvironment() {
				return userConfig.verboseFileRoutes === false;
			}
		},
		rspack() {
			ROOT = process.cwd();
			initUserConfig();
		},
		webpack() {
			ROOT = process.cwd();
			initUserConfig();
		}
	};
};
//#endregion
export { unpluginRouteAutoImportFactory };

//# sourceMappingURL=route-autoimport-plugin.js.map