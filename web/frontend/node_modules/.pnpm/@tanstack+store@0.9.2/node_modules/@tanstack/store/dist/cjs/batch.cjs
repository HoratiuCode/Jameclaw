"use strict";
Object.defineProperty(exports, Symbol.toStringTag, { value: "Module" });
const alien = require("./alien.cjs");
const atom = require("./atom.cjs");
function batch(fn) {
  try {
    alien.startBatch();
    fn();
  } finally {
    alien.endBatch();
    atom.flush();
  }
}
exports.batch = batch;
//# sourceMappingURL=batch.cjs.map
