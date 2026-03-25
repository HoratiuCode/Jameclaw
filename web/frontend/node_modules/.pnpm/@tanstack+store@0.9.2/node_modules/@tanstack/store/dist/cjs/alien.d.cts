export interface ReactiveNode {
    deps?: Link;
    depsTail?: Link;
    subs?: Link;
    subsTail?: Link;
    flags: ReactiveFlags;
}
export interface Link {
    version: number;
    dep: ReactiveNode;
    sub: ReactiveNode;
    prevSub: Link | undefined;
    nextSub: Link | undefined;
    prevDep: Link | undefined;
    nextDep: Link | undefined;
}
export declare const enum ReactiveFlags {
    None = 0,
    Mutable = 1,
    Watching = 2,
    RecursedCheck = 4,
    Recursed = 8,
    Dirty = 16,
    Pending = 32
}
export declare function createReactiveSystem({ update, notify, unwatched, }: {
    update(sub: ReactiveNode): boolean;
    notify(sub: ReactiveNode): void;
    unwatched(sub: ReactiveNode): void;
}): {
    link: (dep: ReactiveNode, sub: ReactiveNode, version: number) => void;
    unlink: (link: Link, sub?: ReactiveNode) => Link | undefined;
    propagate: (link: Link) => void;
    checkDirty: (link: Link, sub: ReactiveNode) => boolean;
    shallowPropagate: (link: Link) => void;
};
export declare function getActiveSub(): ReactiveNode | undefined;
export declare function setActiveSub(sub?: ReactiveNode): ReactiveNode | undefined;
export declare function getBatchDepth(): number;
export declare function startBatch(): void;
export declare function endBatch(): void;
export declare function isSignal(fn: () => void): boolean;
export declare function isComputed(fn: () => void): boolean;
export declare function isEffect(fn: () => void): boolean;
export declare function isEffectScope(fn: () => void): boolean;
export declare function signal<T>(): {
    (): T | undefined;
    (value: T | undefined): void;
};
export declare function signal<T>(initialValue: T): {
    (): T;
    (value: T): void;
};
export declare function computed<T>(getter: (previousValue?: T) => T): () => T;
export declare function effect(fn: () => void): () => void;
export declare function effectScope(fn: () => void): () => void;
export declare function trigger(fn: () => void): void;
