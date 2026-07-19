// Must run before any Web Awesome component module is evaluated, so this lives in its
// own module and is imported first — ES import bindings are hoisted above module bodies,
// so calling these from app.ts directly would run too late.
import { setBasePath, setIconPath } from "@awesome.me/webawesome/dist/webawesome.js";

setBasePath("/assets/wa");

// wa-icon resolves from the Font Awesome CDN by default; musixx is local-only, so point
// it at the SVGs vendored by `just bundle-wa`. Only the icons listed in that recipe exist.
setIconPath("/assets/wa/icons");
