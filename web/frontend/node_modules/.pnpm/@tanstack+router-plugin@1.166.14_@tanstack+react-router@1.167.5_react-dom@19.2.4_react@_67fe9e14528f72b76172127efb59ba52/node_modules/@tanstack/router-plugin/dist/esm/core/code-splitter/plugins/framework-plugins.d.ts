import { ReferenceRouteCompilerPlugin } from '../plugins.js';
import { Config } from '../../config.js';
export declare function getReferenceRouteCompilerPlugins(opts: {
    targetFramework: Config['target'];
    addHmr?: boolean;
}): Array<ReferenceRouteCompilerPlugin> | undefined;
