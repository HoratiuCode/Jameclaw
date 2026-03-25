import { ReferenceRouteCompilerPlugin } from '../plugins.cjs';
import { Config } from '../../config.cjs';
export declare function getReferenceRouteCompilerPlugins(opts: {
    targetFramework: Config['target'];
    addHmr?: boolean;
}): Array<ReferenceRouteCompilerPlugin> | undefined;
