import { startBatch, endBatch } from "./alien.js";
import { flush } from "./atom.js";
function batch(fn) {
  try {
    startBatch();
    fn();
  } finally {
    endBatch();
    flush();
  }
}
export {
  batch
};
//# sourceMappingURL=batch.js.map
