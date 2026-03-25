const require_runtime = require("../_virtual/_rolldown/runtime.cjs");
const require_config = require("./config.cjs");
const require_utils = require("./utils.cjs");
let _tanstack_router_utils = require("@tanstack/router-utils");
let _babel_core = require("@babel/core");
_babel_core = require_runtime.__toESM(_babel_core);
let _babel_template = require("@babel/template");
_babel_template = require_runtime.__toESM(_babel_template);
//#region src/core/route-autoimport-plugin.ts
/**
* This plugin adds imports for createFileRoute and createLazyFileRoute to the file route.
*/
var unpluginRouteAutoImportFactory = (options = {}) => {
	let ROOT = process.cwd();
	let userConfig;
	function initUserConfig() {
		if (typeof options === "function") userConfig = options();
		else userConfig = require_config.getConfig(options, ROOT);
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
				const normalizedId = require_utils.normalizePath(id);
				if (!globalThis.TSR_ROUTES_BY_ID_MAP?.has(normalizedId)) return null;
				let routeType;
				if (code.includes("createFileRoute(")) routeType = "createFileRoute";
				else if (code.includes("createLazyFileRoute(")) routeType = "createLazyFileRoute";
				else return null;
				const routerImportPath = `@tanstack/${userConfig.target}-router`;
				const ast = (0, _tanstack_router_utils.parseAst)({ code });
				let isCreateRouteFunctionImported = false;
				_babel_core.traverse(ast, { Program: { enter(programPath) {
					programPath.traverse({ ImportDeclaration(path) {
						if (path.node.specifiers.map((specifier) => specifier.local.name).includes(routeType) && path.node.source.value === routerImportPath) isCreateRouteFunctionImported = true;
					} });
				} } });
				if (!isCreateRouteFunctionImported) {
					if (require_utils.debug) console.info("Adding autoimports to route ", normalizedId);
					const autoImportStatement = _babel_template.statement(`import { ${routeType} } from '${routerImportPath}'`)();
					ast.program.body.unshift(autoImportStatement);
					const result = (0, _tanstack_router_utils.generateFromAst)(ast, {
						sourceMaps: true,
						filename: normalizedId,
						sourceFileName: normalizedId
					});
					if (require_utils.debug) {
						(0, _tanstack_router_utils.logDiff)(code, result.code);
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
exports.unpluginRouteAutoImportFactory = unpluginRouteAutoImportFactory;

//# sourceMappingURL=route-autoimport-plugin.cjs.map