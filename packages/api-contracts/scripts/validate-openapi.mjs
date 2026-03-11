import fs from "node:fs";
import path from "node:path";

const root = path.resolve(process.cwd(), "packages", "api-contracts", "openapi", "openapi.v1.yaml");

if (!fs.existsSync(root)) {
  console.error(`[contracts] missing file: ${root}`);
  process.exit(1);
}

const body = fs.readFileSync(root, "utf8");
if (!body.includes("openapi:")) {
  console.error("[contracts] openapi header not found");
  process.exit(1);
}
if (!body.includes("/v1/admin/")) {
  console.error("[contracts] expected admin routes not found");
  process.exit(1);
}

console.log("[contracts] basic OpenAPI checks passed");
