import chokidar from "chokidar";
import esbuild from "esbuild";
import fs from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import postcss from "postcss";
import tailwindcss from "@tailwindcss/postcss";

const root = path.dirname(fileURLToPath(import.meta.url));
const watchMode = process.argv.includes("--watch") ||
  process.argv.includes("-w");
const devMode = process.argv.includes("--dev") || process.argv.includes("-D");
const distDir = path.join(root, "dist");

await fs.rm(distDir, { recursive: true, force: true });
await fs.mkdir(distDir, { recursive: true });

async function buildCSS() {
  const source = path.join(root, "assets/main.css");
  const output = path.join(distDir, "main.css");
  const css = await fs.readFile(source, "utf8");
  const result = await postcss([tailwindcss({ optimize: !devMode })]).process(
    css,
    {
      from: source,
      to: output,
      map: { inline: false },
    },
  );
  await fs.writeFile(output, result.css);
  if (result.map) {
    await fs.writeFile(`${output}.map`, result.map.toString());
  }
}

async function buildJS() {
  await esbuild.build({
    entryPoints: [path.join(root, "assets/main.ts")],
    bundle: true,
    outfile: path.join(distDir, "main.js"),
    sourcemap: true,
    minify: !devMode,
    define: {
      "process.env.NODE_ENV": devMode ? '"development"' : '"production"',
    },
    format: "esm",
    platform: "browser",
  });
}

console.log("Building...");
await buildJS();
await buildCSS();
console.log("Done!");

if (watchMode) {
  // Chokidar 4+ watches directories instead of glob patterns.
  const watcher = chokidar.watch([
    path.join(root, "assets"),
    path.join(root, "templates"),
  ], {
    ignoreInitial: true,
    ignored: (file, stats) =>
      path.basename(file).startsWith(".") ||
      (stats?.isFile() && !/\.(css|html|js|ts)$/.test(file)),
  });
  // Serialize rebuilds so rapid edits cannot overwrite newer output.
  let pending = Promise.resolve();
  watcher.on("all", (event, file) => {
    if (!["add", "change", "unlink"].includes(event)) return;
    pending = pending.then(async () => {
      console.log(`Rebuilding ${file}...`);
      await (/\.(js|ts)$/.test(file) ? buildJS() : buildCSS());
      console.log("Done!");
    }).catch((error) => console.error("Build failed:", error));
  });
  watcher.on("error", (error) => {
    console.error("Watcher failed:", error);
    process.exitCode = 1;
    void watcher.close();
  });
  for (const signal of ["SIGINT", "SIGTERM"]) {
    process.on(signal, async () => {
      await watcher.close();
      await pending;
    });
  }
  console.log("Watching assets and templates...");
}
